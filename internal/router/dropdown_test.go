// Test endpoint GET /dropdowns/:type (pilihan dari master data).
package router

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

type dropdownItem struct {
	Value    string  `json:"value"`
	ID       string  `json:"id"`
	Code     *string `json:"code"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

// getDropdown memanggil GET /dropdowns/<path> sebagai siswa. Status harus 200.
func getDropdown(t *testing.T, r *gin.Engine, path string) []dropdownItem {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/dropdowns/"+path, "", tokenFor(t, loginmodel.RoleStudent))
	if res.Code != http.StatusOK {
		t.Fatalf("GET /dropdowns/%s: status = %d, seharusnya 200. body: %s", path, res.Code, res.Body)
	}
	var items []dropdownItem
	decodeData(t, res, &items)
	if items == nil {
		t.Fatalf("data seharusnya array (bukan null). body: %s", res.Body)
	}
	return items
}

// pairs mengubah daftar pilihan menjadi "value=name" supaya mudah dibandingkan.
func pairs(items []dropdownItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.Value+"="+it.Name)
	}
	return strings.Join(parts, ", ")
}

func TestDropdown_TypeYangDisimpanSebagaiKode(t *testing.T) {
	r, _ := setupRouter(t)

	// value = code, sama persis dengan nilai yang diterima API soal (type, used_for, difficulty).
	cases := map[string]string{
		"question_type": "single_choice=Pilihan ganda, multiple_choice=PG kompleks, true_false=Benar / Salah, free_text=Isian",
		"quiz_type":     "daily_quiz=Daily Quiz, uji_kom=Uji Kom, tryout=Tryout, rumath=RUMATH",
		"difficulty":    "easy=Mudah, medium=Sedang, hard=Sulit",
	}
	for itemType, want := range cases {
		items := getDropdown(t, r, itemType)
		if got := pairs(items); got != want {
			t.Errorf("%s = %q, seharusnya %q", itemType, got, want)
		}
		for _, it := range items {
			if it.ID == "" || it.Code == nil || *it.Code != it.Value || it.ParentID != nil {
				t.Errorf("%s: item %+v seharusnya punya id, code = value, tanpa parent", itemType, it)
			}
		}
	}
}

func TestDropdown_JenjangDanKelas(t *testing.T) {
	r, _ := setupRouter(t)

	levels := getDropdown(t, r, "education_level")
	if got := pairs(levels); !strings.HasSuffix(got, "=SMA") || len(levels) != 4 {
		t.Fatalf("jenjang = %q, seharusnya 4 jenjang (PraSD, SD, SMP, SMA)", got)
	}
	var sdID string
	for _, l := range levels {
		if l.Value != l.ID { // jenjang dikirim sebagai id
			t.Errorf("jenjang %s: value %s seharusnya = id %s", l.Name, l.Value, l.ID)
		}
		if l.Name == "SD" {
			sdID = l.ID
		}
	}

	if all := getDropdown(t, r, "grade"); len(all) != 14 {
		t.Errorf("semua kelas = %d, seharusnya 14 (Nol Kecil, Nol Besar, Kelas 1-12)", len(all))
	}
	sd := getDropdown(t, r, "grade?education_level_id="+sdID)
	names := make([]string, 0, len(sd))
	for _, g := range sd {
		names = append(names, g.Name)
		if g.ParentID == nil || *g.ParentID != sdID {
			t.Errorf("kelas %s: parent_id seharusnya SD", g.Name)
		}
	}
	if got := strings.Join(names, ","); got != "Kelas 1,Kelas 2,Kelas 3,Kelas 4,Kelas 5,Kelas 6" {
		t.Errorf("kelas SD = %s", got)
	}
}

func TestDropdown_HierarkiMapel(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)

	// Item nonaktif tidak tampil.
	db.Exec(`INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id, status)
		VALUES ('chapter', 'Bab nonaktif', ?, ?, ?, ?, 'inactive')`, p.FieldID, p.SubjectID, p.LevelID, p.GradeID)
	t.Cleanup(func() { db.Exec(`DELETE FROM master_data_items WHERE name = 'Bab nonaktif'`) })

	subjects := getDropdown(t, r, "subject")
	found := false
	for _, s := range subjects {
		found = found || s.ID == p.SubjectID
	}
	if !found {
		t.Errorf("mapel %s tidak ada di dropdown subject", p.SubjectID)
	}

	cases := []struct{ path, want string }{
		{"subject_field?subject_id=" + p.SubjectID, p.FieldID + "=Bilangan"},
		{"subject_field?subject_id=" + p.SubjectID + "&grade_id=" + p.GradeID, p.FieldID + "=Bilangan"},
		{"subject_field?subject_id=" + p.SubjectID + "&grade_id=" + p.OtherGradeID, ""},
		{"chapter?subject_field_id=" + p.FieldID, p.ChapterID + "=KPK dan FPB"},
		{"subchapter?chapter_id=" + p.ChapterID, p.SubchapterID + "=Menentukan KPK dan FPB"},
		{"topic?subchapter_id=" + p.SubchapterID, p.TopicID + "=Menentukan KPK"},
		{"topic?subchapter_id=" + uuid.NewString(), ""}, // induk tidak ada → kosong
	}
	for _, tc := range cases {
		if got := pairs(getDropdown(t, r, tc.path)); got != tc.want {
			t.Errorf("%s = %q, seharusnya %q", tc.path, got, tc.want)
		}
	}
}

func TestDropdown_ValidasiDanHakAkses(t *testing.T) {
	r, _ := setupRouter(t)
	student := tokenFor(t, loginmodel.RoleStudent)

	// Induk wajib untuk bidang materi s/d topik.
	required := map[string]string{
		"subject_field": "subject_id",
		"chapter":       "subject_field_id",
		"subchapter":    "chapter_id",
		"topic":         "subchapter_id",
	}
	for itemType, field := range required {
		res := sendRequest(r, http.MethodGet, "/api/v1/dropdowns/"+itemType, "", student)
		if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())[field] != "wajib diisi" {
			t.Errorf("%s tanpa %s: status = %d, body: %s", itemType, field, res.Code, res.Body)
		}
	}

	// Format UUID salah.
	res := sendRequest(r, http.MethodGet, "/api/v1/dropdowns/chapter?subject_field_id=abc", "", student)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["subject_field_id"] == "" {
		t.Errorf("uuid salah: status = %d, body: %s", res.Code, res.Body)
	}

	// Type tidak dikenal → 404 berisi daftar type yang ada.
	res = sendRequest(r, http.MethodGet, "/api/v1/dropdowns/kurikulum", "", student)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "difficulty") {
		t.Errorf("type tidak dikenal: status = %d, body: %s", res.Code, res.Body)
	}

	// Wajib login.
	if res := sendRequest(r, http.MethodGet, "/api/v1/dropdowns/difficulty", "", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("tanpa login: status = %d, seharusnya 401", res.Code)
	}
}
