// Test modul pembelajaran untuk siswa: list sesuai jenjang + paket (Basic / Pro / Ultra), buka modul,
// unduh PDF, dan folder file mengikuti hierarki mapel.
package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// subscribe memberi siswa paket aktif (mulai kemarin, berakhir 30 hari lagi).
func subscribe(t *testing.T, db *gorm.DB, studentID, planID string) {
	t.Helper()
	err := db.Exec(`
		INSERT INTO student_subscriptions (student_id, plan_id, channel, period, price_paid, start_date, end_date)
		VALUES (?, ?, 'online', 'monthly', 100000, CURRENT_DATE - 1, CURRENT_DATE + 30)`, studentID, planID).Error
	if err != nil {
		t.Fatal(err)
	}
}

type studentModule struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Locked bool   `json:"locked"`
	File   struct {
		URL *string `json:"url"`
	} `json:"file"`
	Plans  []listRef `json:"plans"`
	Opened *struct {
		OpenCount int `json:"open_count"`
	} `json:"opened"`
}

type studentModuleList struct {
	Access struct {
		EducationLevel *listRef `json:"education_level"`
		Plan           *listRef `json:"plan"`
		ModuleFeature  bool     `json:"module_feature"`
		AllModules     bool     `json:"all_modules"`
	} `json:"access"`
	Modules []studentModule `json:"modules"`
}

func listStudentModules(t *testing.T, r *gin.Engine, token, query string) studentModuleList {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/student/learning-modules"+query, "", token)
	if res.Code != http.StatusOK {
		t.Fatalf("list modul siswa: status %d. body: %s", res.Code, res.Body)
	}
	var data studentModuleList
	decodeData(t, res, &data)
	return data
}

// unlockedNames = nama modul yang bisa dibuka (urut list: kurikulum, lalu nama), contoh "Latihan,Ringkasan".
func unlockedNames(l studentModuleList) string {
	names := []string{}
	for _, m := range l.Modules {
		if !m.Locked {
			names = append(names, m.Name)
		}
	}
	return strings.Join(names, ",")
}

func TestLearningModule_AksesSiswaPerPaketDanJenjang(t *testing.T) {
	app := setupApp(t)
	r, db := app.Router, app.DB
	admin := adminToken(t, db)
	sd := seedPlacementInGrade(t, db, "GRADE_5")
	smp := seedPlacementInGrade(t, db, "GRADE_7")
	basic, pro, ultra := newPlan(t, db, "Uji Basic", 1), newPlan(t, db, "Uji Pro", 2), newPlan(t, db, "Uji Ultra", 3)
	reguler := newPlan(t, db, "Reguler Tanpa Modul", 4)
	db.Exec(`UPDATE subscription_plans SET has_module = FALSE WHERE id = ?`, reguler)

	create := func(name string, p testPlacement, withChapter bool, status string, plans ...string) moduleData {
		t.Helper()
		fields := []formField{{"name", name}, {"subject_id", p.SubjectID}, {"subject_field_id", p.FieldID}, {"status", status}}
		if withChapter {
			fields = append(fields, formField{"chapter_id", p.ChapterID})
		}
		for _, id := range plans {
			fields = append(fields, formField{"plan_ids", id})
		}
		res := sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", fields, name+".pdf", samplePDF(name), admin)
		if res.Code != http.StatusCreated {
			t.Fatalf("buat modul %s: status %d. body: %s", name, res.Code, res.Body)
		}
		var m moduleData
		decodeData(t, res, &m)
		return m
	}
	ringkasan := create("Ringkasan", sd, true, "published", basic, pro, ultra)
	latihan := create("Latihan", sd, true, "published", pro, ultra)
	pembahasan := create("Pembahasan", sd, true, "published", ultra)
	smpModule := create("Ringkasan SMP", smp, false, "published", basic, pro, ultra)
	draft := create("Draf", sd, true, "draft", basic, pro, ultra)

	// ---- Folder file mengikuti hierarki mapel -----------------------------------------
	var storagePath string
	db.Raw(`SELECT storage_path FROM learning_modules WHERE id = ?`, ringkasan.ID).Scan(&storagePath)
	wantDir := "modules/matematika-" + strings.ToLower(strings.TrimPrefix(firstSubjectName(t, db, sd.SubjectID), "Matematika ")) + "/sd-kelas-5/bilangan/kpk-dan-fpb/"
	if !strings.HasPrefix(storagePath, wantDir) || !strings.HasSuffix(storagePath, ".pdf") {
		t.Errorf("lokasi file: %s, seharusnya di %s", storagePath, wantDir)
	}
	if _, err := os.Stat(filepath.Join(app.StorageDir, filepath.FromSlash(storagePath))); err != nil {
		t.Errorf("file tidak ada di lokasi hierarki: %v", err)
	}

	// ---- Siswa per paket -----------------------------------------------------------------
	student := func(name, grade, plan string) (string, string) {
		id, token := namedStudent(t, db, name, grade)
		if plan != "" {
			subscribe(t, db, id, plan)
		}
		return id, token
	}
	_, basicToken := student("Siswa Basic", "GRADE_5", basic)
	_, proToken := student("Siswa Pro", "GRADE_5", pro)
	_, ultraToken := student("Siswa Ultra", "GRADE_5", ultra)
	_, ultraSMPToken := student("Siswa Ultra SMP", "GRADE_7", ultra)
	_, noPlanToken := student("Siswa Tanpa Paket", "GRADE_5", "")
	_, regulerToken := student("Siswa Reguler", "GRADE_5", reguler)
	overrideID, overrideToken := student("Siswa Akses Khusus", "GRADE_5", "")
	db.Exec(`INSERT INTO student_feature_overrides (student_id, feature, is_enabled) VALUES (?, 'module', TRUE)`, overrideID)

	for _, tc := range []struct {
		name, token, wantUnlocked string
		wantVisible               int
	}{
		{"basic: terbatas", basicToken, "Ringkasan", 3},
		{"pro: lebih banyak", proToken, "Latihan,Ringkasan", 3},
		{"ultra: semua di jenjangnya", ultraToken, "Latihan,Pembahasan,Ringkasan", 3},
		{"ultra SMP: hanya modul SMP", ultraSMPToken, "Ringkasan SMP", 1},
		{"tanpa paket: semua terkunci", noPlanToken, "", 3},
		{"paket tanpa fitur modul: semua terkunci", regulerToken, "", 3},
		{"akses khusus admin: semua di jenjangnya", overrideToken, "Latihan,Pembahasan,Ringkasan", 3},
	} {
		l := listStudentModules(t, r, tc.token, "")
		if len(l.Modules) != tc.wantVisible || unlockedNames(l) != tc.wantUnlocked {
			t.Errorf("%s: %d modul tampil, terbuka %q; seharusnya %d tampil, terbuka %q", tc.name, len(l.Modules), unlockedNames(l), tc.wantVisible, tc.wantUnlocked)
		}
		for _, m := range l.Modules {
			if m.Locked != (m.File.URL == nil) {
				t.Errorf("%s: modul %s locked %v tapi url %v", tc.name, m.Name, m.Locked, m.File.URL)
			}
		}
	}
	l := listStudentModules(t, r, basicToken, "?access=locked")
	if len(l.Modules) != 2 || l.Access.Plan == nil || l.Access.Plan.Name != "Uji Basic" || !l.Access.ModuleFeature || l.Access.AllModules ||
		l.Modules[0].Name != "Latihan" || len(l.Modules[0].Plans) != 2 {
		t.Errorf("basic, filter terkunci: %+v", l)
	}
	if l := listStudentModules(t, r, overrideToken, ""); !l.Access.AllModules || l.Access.Plan != nil {
		t.Errorf("akses khusus: %+v", l.Access)
	}

	// ---- Buka modul & unduh PDF ------------------------------------------------------------
	res := sendRequest(r, http.MethodPost, "/api/v1/student/learning-modules/"+latihan.ID+"/open", "", basicToken)
	var locked struct {
		Message string `json:"message"`
		Data    struct {
			Plans []listRef `json:"plans"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &locked)
	if res.Code != http.StatusForbidden || locked.Message != "Modul ini tidak termasuk paket Uji Basic. Modul ini tersedia di paket Uji Pro, Uji Ultra" || len(locked.Data.Plans) != 2 {
		t.Errorf("buka modul di luar paket: status %d, %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/learning-modules/"+latihan.ID+"/file", "", basicToken); res.Code != http.StatusForbidden {
		t.Errorf("unduh modul di luar paket: status %d", res.Code)
	}

	var opened studentModule
	for i := 1; i <= 2; i++ {
		res = sendRequest(r, http.MethodPost, "/api/v1/student/learning-modules/"+ringkasan.ID+"/open", "", basicToken)
		decodeData(t, res, &opened)
		if res.Code != http.StatusOK || opened.Locked || opened.Opened == nil || opened.Opened.OpenCount != i ||
			opened.File.URL == nil || *opened.File.URL != testAppURL+"/api/v1/student/learning-modules/"+ringkasan.ID+"/file" {
			t.Errorf("buka ke-%d: status %d, %+v", i, res.Code, opened)
		}
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/student/learning-modules/"+ringkasan.ID+"/file", "", basicToken)
	if res.Code != http.StatusOK || !bytes.Equal(res.Body.Bytes(), samplePDF("Ringkasan")) {
		t.Errorf("unduh PDF siswa: status %d", res.Code)
	}
	// Riwayat buka tampil di list.
	for _, m := range listStudentModules(t, r, basicToken, "").Modules {
		if m.ID == ringkasan.ID && (m.Opened == nil || m.Opened.OpenCount != 2) {
			t.Errorf("riwayat buka di list: %+v", m.Opened)
		}
	}

	// Jenjang lain, draf, tidak ada → 404 (siswa tidak tahu modul itu ada).
	for _, tc := range []struct{ token, id string }{
		{ultraSMPToken, ringkasan.ID}, {ultraToken, smpModule.ID}, {ultraToken, draft.ID}, {ultraToken, "bukan-uuid"},
	} {
		if res := sendRequest(r, http.MethodPost, "/api/v1/student/learning-modules/"+tc.id+"/open", "", tc.token); res.Code != http.StatusNotFound {
			t.Errorf("buka %s: status %d, seharusnya 404", tc.id, res.Code)
		}
	}
	// Admin tidak memakai endpoint siswa; siswa tidak memakai endpoint admin.
	if res := sendRequest(r, http.MethodGet, "/api/v1/student/learning-modules", "", admin); res.Code != http.StatusForbidden {
		t.Errorf("admin ke endpoint siswa: status %d", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/learning-modules/"+pembahasan.ID+"/file", "", ultraToken); res.Code != http.StatusForbidden {
		t.Errorf("siswa ke endpoint file admin: status %d", res.Code)
	}

	// ---- Letak diubah → file ikut pindah ke folder hierarki baru ------------------------------
	res = sendModuleForm(r, http.MethodPut, "/api/v1/learning-modules/"+ringkasan.ID, []formField{
		{"name", "Ringkasan"}, {"subject_id", sd.SubjectID}, {"subject_field_id", sd.FieldID}, {"plan_ids", basic},
	}, "", nil, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah letak: status %d. body: %s", res.Code, res.Body)
	}
	var moved string
	db.Raw(`SELECT storage_path FROM learning_modules WHERE id = ?`, ringkasan.ID).Scan(&moved)
	if moved == storagePath || strings.Contains(moved, "kpk-dan-fpb") || filepath.Base(moved) != filepath.Base(storagePath) {
		t.Errorf("file setelah letak berubah: %s (sebelumnya %s)", moved, storagePath)
	}
	if _, err := os.Stat(filepath.Join(app.StorageDir, filepath.FromSlash(moved))); err != nil {
		t.Errorf("file di lokasi baru tidak ada: %v", err)
	}
	if _, err := os.Stat(filepath.Join(app.StorageDir, filepath.FromSlash(storagePath))); !os.IsNotExist(err) {
		t.Error("file di lokasi lama seharusnya sudah dipindah")
	}
}

// firstSubjectName mengambil nama mapel (dipakai menyusun nama folder yang diharapkan).
func firstSubjectName(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var name string
	db.Raw(`SELECT name FROM master_data_items WHERE id = ?`, id).Scan(&name)
	return name
}
