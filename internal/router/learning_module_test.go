// Test modul pembelajaran (materi PDF): buat, list, detail, unduh file, ubah, hapus.
package router

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// samplePDF adalah isi PDF minimal (cukup untuk dikenali sebagai PDF dari isinya).
func samplePDF(text string) []byte {
	return []byte("%PDF-1.4\n% " + text + "\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
}

// formField = 1 field form (boleh berulang, contoh plan_ids).
type formField struct{ Key, Value string }

// sendModuleForm mengirim multipart ke /api/v1/learning-modules (POST) atau /:id (PUT).
// fileData nil = tanpa field "file".
func sendModuleForm(r *gin.Engine, method, url string, fields []formField, fileName string, fileData []byte, accessToken string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, f := range fields {
		_ = writer.WriteField(f.Key, f.Value)
	}
	if fileData != nil {
		part, _ := writer.CreateFormFile("file", fileName)
		_, _ = part.Write(fileData)
	}
	_ = writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

// newPlan membuat paket akses (subscription_plans) aktif, mengembalikan ID-nya.
func newPlan(t *testing.T, db *gorm.DB, name string, sortOrder int) string {
	t.Helper()
	var id string
	err := db.Raw(`INSERT INTO subscription_plans (name, price_monthly, has_module, sort_order) VALUES (?, 100000, TRUE, ?) RETURNING id`, name, sortOrder).Scan(&id).Error
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type moduleData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Placement   struct {
		EducationLevel listRef  `json:"education_level"`
		Grade          listRef  `json:"grade"`
		Subject        listRef  `json:"subject"`
		SubjectField   listRef  `json:"subject_field"`
		Chapter        *listRef `json:"chapter"`
	} `json:"placement"`
	File struct {
		Name      string `json:"name"`
		MimeType  string `json:"mime_type"`
		SizeBytes int64  `json:"size_bytes"`
		PageCount *int   `json:"page_count"`
		URL       string `json:"url"`
	} `json:"file"`
	Plans     []listRef `json:"plans"`
	Status    string    `json:"status"`
	UpdatedBy *listRef  `json:"updated_by"`
}

// moduleFiles = semua file PDF yang tersimpan di folder modules (untuk cek file lama terhapus).
func moduleFiles(t *testing.T, storageDir string) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(filepath.Join(storageDir, "modules"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func TestLearningModule_CRUD(t *testing.T) {
	app := setupApp(t)
	r, db := app.Router, app.DB
	admin := adminToken(t, db)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	other := seedPlacementInGrade(t, db, "GRADE_5") // mapel lain
	reguler := newPlan(t, db, "Uji Reguler SD", 1)
	intensif := newPlan(t, db, "Uji Intensif SD", 2)
	inactive := newPlan(t, db, "Uji Paket Lama", 3)
	db.Exec(`UPDATE subscription_plans SET status = 'inactive' WHERE id = ?`, inactive)

	base := []formField{
		{"name", " Modul Pecahan "}, {"description", "Materi pecahan sederhana kelas 5"},
		{"subject_id", p.SubjectID}, {"subject_field_id", p.FieldID}, {"chapter_id", p.ChapterID},
		{"plan_ids", intensif}, {"plan_ids", reguler}, {"page_count", "12"},
	}

	// ---- Buat -------------------------------------------------------------------
	pdf := samplePDF("versi 1")
	res := sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", base, "pecahan.pdf", pdf, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("buat modul: status %d. body: %s", res.Code, res.Body)
	}
	var m moduleData
	decodeData(t, res, &m)
	if m.Name != "Modul Pecahan" || m.Placement.Grade.ID != p.GradeID || m.Placement.EducationLevel.ID != p.LevelID ||
		m.Placement.Chapter == nil || m.Placement.Chapter.ID != p.ChapterID || m.Status != "published" ||
		m.File.Name != "pecahan.pdf" || m.File.MimeType != "application/pdf" || m.File.SizeBytes != int64(len(pdf)) ||
		m.File.URL != testAppURL+"/api/v1/learning-modules/"+m.ID+"/file" || len(m.Plans) != 2 || m.Plans[0].ID != reguler {
		t.Fatalf("modul baru: %+v", m)
	}

	// Unduh PDF: wajib login & admin; isi sama dengan yang diupload. Tidak ada URL publik.
	res = sendRequest(r, http.MethodGet, "/api/v1/learning-modules/"+m.ID+"/file", "", admin)
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "application/pdf" || !bytes.Equal(res.Body.Bytes(), pdf) {
		t.Errorf("unduh PDF: status %d, type %q", res.Code, res.Header().Get("Content-Type"))
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/learning-modules/"+m.ID+"/file", "", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("unduh tanpa login: status %d", res.Code)
	}
	if files := moduleFiles(t, app.StorageDir); len(files) != 1 {
		t.Fatalf("file tersimpan: %v", files)
	}

	// ---- Validasi ------------------------------------------------------------------
	bad := []formField{
		{"name", "Modul X"}, {"subject_id", other.SubjectID}, {"subject_field_id", p.FieldID},
		{"subchapter_id", p.SubchapterID}, {"plan_ids", inactive}, {"plan_ids", uuid.NewString()},
	}
	res = sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", bad, "gambar.pdf", []byte("\x89PNG\r\n\x1a\nbukan pdf"), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["file"] != "gambar.pdf: harus file PDF" ||
		f["subject_field_id"] != "bidang materi bukan bagian dari mapel yang dipilih" || f["chapter_id"] != "wajib diisi jika subchapter_id diisi" ||
		f["plan_ids[0]"] != "paket akses tidak ditemukan atau tidak aktif" || f["plan_ids[1]"] == "" {
		t.Errorf("validasi: status %d, errors %v", res.Code, f)
	}
	res = sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", base, "", nil, admin)
	if errorFields(t, res.Body.String())["file"] != "wajib diisi (file PDF, maksimal 20 MB)" {
		t.Errorf("tanpa file: %s", res.Body)
	}
	res = sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", []formField{{"name", "Y"}}, "y.pdf", pdf, admin)
	if f := errorFields(t, res.Body.String()); f["subject_id"] == "" || f["subject_field_id"] == "" || f["plan_ids"] == "" {
		t.Errorf("field wajib: %v", f)
	}
	if files := moduleFiles(t, app.StorageDir); len(files) != 1 {
		t.Errorf("upload gagal tidak boleh meninggalkan file: %v", files)
	}

	// ---- Ubah tanpa ganti PDF, lalu ganti PDF --------------------------------------
	update := []formField{
		{"name", "Modul Pecahan (revisi)"}, {"subject_id", p.SubjectID}, {"subject_field_id", p.FieldID},
		{"plan_ids", reguler}, {"status", "draft"},
	}
	res = sendModuleForm(r, http.MethodPut, "/api/v1/learning-modules/"+m.ID, update, "", nil, admin)
	decodeData(t, res, &m)
	if res.Code != http.StatusOK || m.Name != "Modul Pecahan (revisi)" || m.Description != nil || m.Placement.Chapter != nil ||
		len(m.Plans) != 1 || m.Status != "draft" || m.File.Name != "pecahan.pdf" || m.UpdatedBy == nil {
		t.Errorf("ubah tanpa file: status %d, %+v", res.Code, m)
	}
	// Ubah tanpa PDF & tanpa page_count: jumlah halaman tetap (12).
	if m.File.PageCount == nil || *m.File.PageCount != 12 {
		t.Errorf("jumlah halaman setelah ubah tanpa PDF: %v", m.File.PageCount)
	}
	oldFiles := moduleFiles(t, app.StorageDir)
	pdf2 := samplePDF("versi 2 yang lebih panjang")
	res = sendModuleForm(r, http.MethodPut, "/api/v1/learning-modules/"+m.ID, update, "pecahan-v2.pdf", pdf2, admin)
	decodeData(t, res, &m)
	newFiles := moduleFiles(t, app.StorageDir)
	if res.Code != http.StatusOK || m.File.Name != "pecahan-v2.pdf" || len(newFiles) != 1 || newFiles[0] == oldFiles[0] {
		t.Errorf("ganti PDF: status %d, file %v → %v", res.Code, oldFiles, newFiles)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/learning-modules/"+m.ID+"/file", "", admin); !bytes.Equal(res.Body.Bytes(), pdf2) {
		t.Error("unduh setelah ganti PDF seharusnya file baru")
	}
	if res := sendModuleForm(r, http.MethodPut, "/api/v1/learning-modules/"+uuid.NewString(), update, "", nil, admin); res.Code != http.StatusNotFound {
		t.Errorf("ubah modul tidak ada: status %d", res.Code)
	}

	// ---- List & filter ----------------------------------------------------------------
	second := sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", []formField{
		{"name", "Modul IPA Cahaya"}, {"subject_id", other.SubjectID}, {"subject_field_id", other.FieldID}, {"plan_ids", intensif},
	}, "cahaya.pdf", samplePDF("ipa"), admin)
	var ipa moduleData
	decodeData(t, second, &ipa)
	for query, want := range map[string]int{
		"":                           2,
		"?plan_id=" + intensif:       1,
		"?subject_id=" + p.SubjectID: 1,
		"?grade_id=" + p.GradeID:     2,
		"?search=cahaya":             1,
		"?status=draft":              1,
	} {
		var items []moduleData
		res := sendRequest(r, http.MethodGet, "/api/v1/learning-modules"+query, "", admin)
		decodeData(t, res, &items)
		if len(items) != want || decodeMeta(t, res).Total != want {
			t.Errorf("list %q: %d modul, seharusnya %d", query, len(items), want)
		}
	}

	// ---- Hapus (soft delete) ---------------------------------------------------------
	res = sendRequest(r, http.MethodDelete, "/api/v1/learning-modules", fmt.Sprintf(`{"ids":[%q,%q]}`, m.ID, uuid.NewString()), admin)
	if errorFields(t, res.Body.String())["ids[1]"] != "modul tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus dengan ID salah: %s", res.Body)
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/learning-modules", fmt.Sprintf(`{"ids":[%q]}`, m.ID), admin); res.Code != http.StatusOK {
		t.Fatalf("hapus: status %d. body: %s", res.Code, res.Body)
	}
	for _, url := range []string{"/api/v1/learning-modules/" + m.ID, "/api/v1/learning-modules/" + m.ID + "/file"} {
		if res := sendRequest(r, http.MethodGet, url, "", admin); res.Code != http.StatusNotFound {
			t.Errorf("%s setelah dihapus: status %d", url, res.Code)
		}
	}
	var items []moduleData
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/learning-modules", "", admin), &items)
	if len(items) != 1 || items[0].ID != ipa.ID {
		t.Errorf("list setelah hapus: %+v", items)
	}

	// ---- Hak akses ------------------------------------------------------------------
	_, student := namedStudent(t, db, "Ani", "GRADE_5")
	for _, tok := range []string{student, tokenFor(t, loginmodel.RoleTutor)} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/learning-modules", "", tok); res.Code != http.StatusForbidden {
			t.Errorf("list oleh bukan admin: status %d", res.Code)
		}
	}

	// ---- Dropdown paket akses ----------------------------------------------------------
	var plans []struct {
		Value, Name string
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/dropdowns/subscription_plan", "", admin), &plans)
	if len(plans) != 2 || plans[0].Value != reguler || plans[1].Name != "Uji Intensif SD" {
		t.Errorf("dropdown paket: %+v", plans)
	}
}

// PDF berisi 2 objek halaman: page_count dihitung otomatis kalau admin tidak mengisinya.
func TestLearningModule_JumlahHalamanOtomatis(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	plan := newPlan(t, db, "Uji Reguler SD", 1)
	pdf := []byte("%PDF-1.4\n1 0 obj << /Type /Pages /Kids [2 0 R 3 0 R] /Count 2 >> endobj\n" +
		"2 0 obj << /Type /Page /Parent 1 0 R >> endobj\n3 0 obj << /Type/Page /Parent 1 0 R >> endobj\n%%EOF\n")
	fields := []formField{{"name", "Modul 2 halaman"}, {"subject_id", p.SubjectID}, {"subject_field_id", p.FieldID}, {"plan_ids", plan}}

	res := sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", fields, "dua.pdf", pdf, admin)
	var m moduleData
	decodeData(t, res, &m)
	if res.Code != http.StatusCreated || m.File.PageCount == nil || *m.File.PageCount != 2 {
		t.Errorf("hitung halaman otomatis: status %d, page_count %v", res.Code, m.File.PageCount)
	}
	// PDF tanpa objek halaman yang terbaca → null (admin bisa mengisi page_count).
	res = sendModuleForm(r, http.MethodPost, "/api/v1/learning-modules", fields, "x.pdf", samplePDF("tanpa halaman"), admin)
	decodeData(t, res, &m)
	if res.Code != http.StatusCreated || m.File.PageCount != nil {
		t.Errorf("tanpa halaman terbaca: status %d, page_count %v", res.Code, m.File.PageCount)
	}
}
