// Test menu Master Data (mapel s/d topik, kelas, sekolah) dan Cabang: tambah, list, jumlah tab,
// ubah, hapus (soft delete) dengan aturan pemakaian & data sistem.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

type masterItem struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	TypeLabel      string   `json:"type_label"`
	Code           *string  `json:"code"`
	Name           string   `json:"name"`
	Parent         *listRef `json:"parent"`
	Subject        *listRef `json:"subject"`
	EducationLevel *listRef `json:"education_level"`
	Grade          *listRef `json:"grade"`
	SortOrder      int      `json:"sort_order"`
	Status         string   `json:"status"`
	IsSystem       bool     `json:"is_system"`
	CanDelete      bool     `json:"can_delete"`
	Usage          struct {
		Children, Questions, Modules int
	} `json:"usage"`
	CreatedBy *listRef `json:"created_by"`
	UpdatedBy *listRef `json:"updated_by"`
}

func createMaster(t *testing.T, r *gin.Engine, admin, body string) masterItem {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/master-data", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("tambah master data %s: status %d. body: %s", body, res.Code, res.Body)
	}
	var item masterItem
	decodeData(t, res, &item)
	return item
}

func listMaster(t *testing.T, r *gin.Engine, admin, query string) []masterItem {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/master-data"+query, "", admin)
	if res.Code != http.StatusOK {
		t.Fatalf("list master data %s: status %d. body: %s", query, res.Code, res.Body)
	}
	var items []masterItem
	decodeData(t, res, &items)
	return items
}

func TestMasterData_HierarkiMapel(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	sfx := uuid.NewString()[:6]
	grade4, grade5 := masterID(t, db, "grade", "GRADE_4"), masterID(t, db, "grade", "GRADE_5")
	sdID := masterID(t, db, "education_level", "SD")

	subject := createMaster(t, r, admin, fmt.Sprintf(`{"type":"subject","name":"IPA %s","code":"IPA_%s"}`, sfx, sfx))
	cleanupSubject(t, db, subject.ID)
	if subject.TypeLabel != "Mapel" || subject.Parent != nil || subject.Status != "active" || subject.IsSystem || subject.CreatedBy == nil || subject.SortOrder < 1 {
		t.Errorf("mapel baru: %+v", subject)
	}

	// Validasi induk & kelas (dikumpulkan sekaligus).
	for body, want := range map[string]map[string]string{
		`{"type":"subject_field","name":"Fisika"}`:                                                             {"parent_id": "wajib diisi (mapel)"},
		fmt.Sprintf(`{"type":"subject_field","name":"Fisika","parent_id":%q}`, subject.ID):                     {"grade_id": "wajib diisi (kelas bidang materi)"},
		fmt.Sprintf(`{"type":"subject","name":" ","parent_id":%q,"code":"a b"}`, subject.ID):                   {"name": "wajib diisi", "parent_id": "Mapel tidak punya induk, kosongkan", "code": "hanya huruf, angka, titik, garis bawah, dan strip (maks 30)"},
		fmt.Sprintf(`{"type":"chapter","name":"Gaya","parent_id":%q}`, subject.ID):                             {"parent_id": "induk harus bidang materi, bukan mapel"},
		fmt.Sprintf(`{"type":"subject_field","name":"Fisika","parent_id":%q,"grade_id":%q}`, subject.ID, sdID): {"grade_id": "kelas tidak ditemukan"},
	} {
		res := sendRequest(r, http.MethodPost, "/api/v1/master-data", body, admin)
		f := errorFields(t, res.Body.String())
		for field, msg := range want {
			if res.Code != http.StatusUnprocessableEntity || f[field] != msg {
				t.Errorf("validasi %s: status %d, %s = %q (seharusnya %q)", body, res.Code, field, f[field], msg)
			}
		}
	}

	// Bidang materi: jenjang mengikuti kelas; nama sama boleh di kelas lain, tidak boleh di kelas sama.
	field := createMaster(t, r, admin, fmt.Sprintf(`{"type":"subject_field","name":"Fisika","parent_id":%q,"grade_id":%q}`, subject.ID, grade4))
	if field.Subject == nil || field.Subject.ID != subject.ID || field.Grade.Name != "Kelas 4" || field.EducationLevel.Name != "SD" || field.Parent.ID != subject.ID {
		t.Errorf("bidang materi: %+v", field)
	}
	createMaster(t, r, admin, fmt.Sprintf(`{"type":"subject_field","name":"Fisika","parent_id":%q,"grade_id":%q}`, subject.ID, grade5))
	res := sendRequest(r, http.MethodPost, "/api/v1/master-data", fmt.Sprintf(`{"type":"subject_field","name":"fisika","parent_id":%q,"grade_id":%q}`, subject.ID, grade4), admin)
	if f := errorFields(t, res.Body.String()); f["name"] != "nama bidang materi sudah dipakai di induk dan kelas yang sama" {
		t.Errorf("bidang materi kembar: %d %v", res.Code, f)
	}

	// Bab s/d topik: cukup pilih induk, letak disalin; urutan otomatis bertambah.
	chapter := createMaster(t, r, admin, fmt.Sprintf(`{"type":"chapter","name":"Gaya","parent_id":%q}`, field.ID))
	chapter2 := createMaster(t, r, admin, fmt.Sprintf(`{"type":"chapter","name":"Energi","parent_id":%q}`, field.ID))
	sub := createMaster(t, r, admin, fmt.Sprintf(`{"type":"subchapter","name":"Gaya Gesek","parent_id":%q}`, chapter.ID))
	topic := createMaster(t, r, admin, fmt.Sprintf(`{"type":"topic","name":"Gesek Statis","parent_id":%q,"sort_order":7}`, sub.ID))
	if chapter.Grade == nil || chapter.Grade.ID != grade4 || chapter.Subject.ID != subject.ID || chapter2.SortOrder != chapter.SortOrder+1 ||
		topic.Grade.ID != grade4 || topic.SortOrder != 7 || topic.Parent.Name != "Gaya Gesek" {
		t.Errorf("bab/topik: %+v %+v %+v", chapter, chapter2, topic)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/master-data", fmt.Sprintf(`{"type":"chapter","name":"X","parent_id":%q,"grade_id":%q}`, field.ID, grade5), admin)
	if f := errorFields(t, res.Body.String()); !strings.Contains(f["grade_id"], "hanya diisi untuk bidang materi") {
		t.Errorf("bab dengan grade_id: %v", f)
	}

	// List: filter type + mapel + kelas, cari, urutan; pemakaian (anak).
	items := listMaster(t, r, admin, "?type=chapter&subject_id="+subject.ID+"&grade_id="+grade4)
	if len(items) != 2 || items[0].Name != "Gaya" || items[1].Name != "Energi" {
		t.Errorf("list bab: %+v", items)
	}
	items = listMaster(t, r, admin, "?search=IPA%20"+sfx)
	if len(items) != 1 || items[0].ID != subject.ID || items[0].Usage.Children != 2 || items[0].CanDelete {
		t.Errorf("cari mapel + pemakaian: %+v", items)
	}
	items = listMaster(t, r, admin, "?parent_id="+subject.ID)
	if len(items) != 2 || items[0].Grade.Name != "Kelas 4" || items[1].Grade.Name != "Kelas 5" {
		t.Errorf("list per induk (urut kelas): %+v", items)
	}
	items = listMaster(t, r, admin, "?type=grade&education_level_id="+sdID)
	if len(items) != 6 || !items[0].IsSystem || items[0].EducationLevel == nil || items[0].EducationLevel.Name != "SD" {
		t.Errorf("list kelas SD: %d %+v", len(items), items)
	}
	var counts map[string]int
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/master-data/counts?search="+sfx, "", admin), &counts)
	if counts["subject"] != 1 || counts["all"] != 1 {
		t.Errorf("jumlah tab: %v", counts)
	}

	// Ubah: nama, kode, urutan, status. Data sistem: hanya nama & urutan.
	res = sendRequest(r, http.MethodPut, "/api/v1/master-data/"+chapter.ID, `{"name":"Gaya dan Gerak","sort_order":9,"status":"inactive"}`, admin)
	var updated masterItem
	decodeData(t, res, &updated)
	if res.Code != http.StatusOK || updated.Name != "Gaya dan Gerak" || updated.Status != "inactive" || updated.SortOrder != 9 || updated.UpdatedBy == nil || updated.Grade.ID != grade4 {
		t.Errorf("ubah bab: %d %+v", res.Code, updated)
	}
	var gradeName string
	db.Raw(`SELECT name FROM master_data_items WHERE id = ?`, grade5).Scan(&gradeName)
	res = sendRequest(r, http.MethodPut, "/api/v1/master-data/"+grade5, `{"name":"Kelas 5 SD","code":"GRADE_X","sort_order":7,"status":"inactive"}`, admin)
	if f := errorFields(t, res.Body.String()); res.Code != 422 || f["code"] == "" || f["status"] == "" {
		t.Errorf("ubah kode/status data sistem: %d %v", res.Code, f)
	}
	res = sendRequest(r, http.MethodPut, "/api/v1/master-data/"+grade5, `{"name":"Kelas 5 SD","sort_order":7,"status":"active"}`, admin)
	decodeData(t, res, &updated)
	if res.Code != http.StatusOK || updated.Name != "Kelas 5 SD" || updated.Code == nil || *updated.Code != "GRADE_5" {
		t.Errorf("ubah nama kelas sistem: %d %+v", res.Code, updated)
	}
	db.Exec(`UPDATE master_data_items SET name = ?, sort_order = 7 WHERE id = ?`, gradeName, grade5) // kembalikan

	// Hapus: data sistem & yang masih punya anak ditolak; dari bawah ke atas berhasil.
	res = sendRequest(r, http.MethodDelete, "/api/v1/master-data", fmt.Sprintf(`{"ids":[%q,%q,%q]}`, grade4, field.ID, uuid.NewString()), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != 422 || !strings.Contains(f["ids[0]"], "data sistem") || !strings.Contains(f["ids[1]"], "masih dipakai 2 item anak") || f["ids[2]"] != "data tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus ditolak: %d %v", res.Code, f)
	}
	for _, id := range []string{topic.ID, sub.ID, chapter.ID} {
		if res := sendRequest(r, http.MethodDelete, "/api/v1/master-data", fmt.Sprintf(`{"ids":[%q]}`, id), admin); res.Code != http.StatusOK {
			t.Fatalf("hapus %s: %d %s", id, res.Code, res.Body)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/master-data/"+chapter.ID, "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail bab terhapus: %d", res.Code)
	}
	// Nama bab yang sudah dihapus boleh dipakai lagi.
	createMaster(t, r, admin, fmt.Sprintf(`{"type":"chapter","name":"Gaya","parent_id":%q}`, field.ID))

	// Bab yang dipakai soal tidak bisa dihapus.
	p := seedPlacementInGrade(t, db, "GRADE_4")
	createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	res = sendRequest(r, http.MethodDelete, "/api/v1/master-data", fmt.Sprintf(`{"ids":[%q]}`, p.ChapterID), admin)
	if f := errorFields(t, res.Body.String()); res.Code != 422 || !strings.Contains(f["ids[0]"], "1 soal") {
		t.Errorf("hapus bab yang dipakai soal: %d %v", res.Code, f)
	}
	var detail masterItem
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/master-data/"+p.ChapterID, "", admin), &detail)
	if detail.Usage.Questions != 1 || detail.CanDelete {
		t.Errorf("pemakaian bab: %+v", detail.Usage)
	}

	// Hanya admin & super admin.
	if res := sendRequest(r, http.MethodGet, "/api/v1/master-data", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor membuka master data: %d", res.Code)
	}
}

func TestMasterData_SekolahDanCabang(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	sfx := uuid.NewString()[:6]

	// Sekolah: tanpa induk, muncul di dropdown untuk semua user yang login.
	school := createMaster(t, r, admin, fmt.Sprintf(`{"type":"school","name":"SMPN Uji %s"}`, sfx))
	t.Cleanup(func() { db.Exec(`DELETE FROM master_data_items WHERE id = ?`, school.ID) })
	res := sendRequest(r, http.MethodGet, "/api/v1/dropdowns/school", "", tokenFor(t, loginmodel.RoleStudent))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), school.ID) {
		t.Errorf("dropdown sekolah: %d %s", res.Code, res.Body)
	}

	// Cabang: tambah, nama kembar, list, ubah nonaktif (hilang dari dropdown), hapus.
	type branch struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Status    string   `json:"status"`
		Users     int      `json:"users"`
		CanDelete bool     `json:"can_delete"`
		CreatedBy *listRef `json:"created_by"`
	}
	createBranch := func(body string) (int, branch) {
		res := sendRequest(r, http.MethodPost, "/api/v1/branches", body, admin)
		var b branch
		if res.Code == http.StatusCreated {
			decodeData(t, res, &b)
		}
		return res.Code, b
	}
	code, gresik := createBranch(`{"name":"Gresik"}`)
	if code != http.StatusCreated || gresik.Status != "active" || gresik.CreatedBy == nil || !gresik.CanDelete {
		t.Fatalf("tambah cabang: %d %+v", code, gresik)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/branches", `{"name":"  gresik "}`, admin)
	if f := errorFields(t, res.Body.String()); f["name"] != "nama cabang sudah dipakai" {
		t.Errorf("cabang kembar: %d %v", res.Code, f)
	}
	_, malang := createBranch(`{"name":"Malang"}`)
	student := createUser(t, db, loginmodel.RoleStudent, "siswa.cabang", loginmodel.UserStatusActive)
	db.Exec(`UPDATE users SET branch_id = ? WHERE id = ?`, malang.ID, student)

	var list []branch
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/branches", "", admin), &list)
	if len(list) != 2 || list[0].Name != "Gresik" || list[1].Users != 1 || list[1].CanDelete {
		t.Errorf("list cabang: %+v", list)
	}

	res = sendRequest(r, http.MethodPut, "/api/v1/branches/"+gresik.ID, `{"name":"Gresik Kota","status":"inactive"}`, admin)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"status":"inactive"`) {
		t.Errorf("ubah cabang: %d %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/dropdowns/branch", "", admin)
	if res.Code != http.StatusOK || strings.Contains(res.Body.String(), gresik.ID) || !strings.Contains(res.Body.String(), malang.ID) {
		t.Errorf("dropdown cabang (tanpa yang nonaktif): %d %s", res.Code, res.Body)
	}

	var counts map[string]int
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/master-data/counts", "", admin), &counts)
	if counts["branch"] != 2 || counts["school"] < 1 {
		t.Errorf("jumlah tab cabang/sekolah: %v", counts)
	}

	res = sendRequest(r, http.MethodDelete, "/api/v1/branches", fmt.Sprintf(`{"ids":[%q,%q]}`, gresik.ID, malang.ID), admin)
	if f := errorFields(t, res.Body.String()); res.Code != 422 || !strings.Contains(f["ids[1]"], "masih dipakai 1 akun") || f["ids[0]"] != "" {
		t.Errorf("hapus cabang dipakai: %d %v", res.Code, f)
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/branches", fmt.Sprintf(`{"ids":[%q]}`, gresik.ID), admin); res.Code != http.StatusOK {
		t.Errorf("hapus cabang: %d %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/branches/"+gresik.ID, "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail cabang terhapus: %d", res.Code)
	}
	// Nama cabang yang dihapus boleh dipakai lagi.
	if code, _ := createBranch(`{"name":"Gresik Kota"}`); code != http.StatusCreated {
		t.Errorf("tambah ulang nama cabang terhapus: %d", code)
	}
}

// cleanupSubject menghapus permanen 1 mapel buatan test beserta seluruh turunannya (dari topik ke
// atas) setelah test selesai, karena master_data_items tidak dikosongkan antar test.
func cleanupSubject(t *testing.T, db *gorm.DB, subjectID string) {
	t.Cleanup(func() {
		for _, itemType := range []string{"topic", "subchapter", "chapter", "subject_field"} {
			db.Exec(`DELETE FROM master_data_items WHERE subject_id = ? AND type = ?`, subjectID, itemType)
		}
		db.Exec(`DELETE FROM master_data_items WHERE id = ?`, subjectID)
	})
}
