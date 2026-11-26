// Test tanggal bergabung / berakhir akun dan cek file Excel tambah user (POST /users/import/check).
package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

const userImportURL = "/api/v1/users/import/check"

// sendUserExcel mengirim file Excel ke cek import user (multipart, field "file").
func sendUserExcel(r *gin.Engine, fileName string, data []byte, accessToken string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	_, _ = part.Write(data)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, userImportURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func TestUsers_TanggalBerakhirMemblokirLogin(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	branch := newBranch(t, db, "Surabaya Timur")
	today := time.Now().In(clock.Jakarta)
	yesterday := today.AddDate(0, 0, -1).Format(time.DateOnly)

	// join_date wajib (format dicek binding); end_date sebelum join_date ditolak.
	res := sendRequest(r, http.MethodPost, "/api/v1/users", fmt.Sprintf(`{"users":[
		{"role":"tutor","username":"tutor.b","name":"B","password":"password123","branch_id":%q}]}`, branch), admin)
	if f := errorFields(t, res.Body.String()); f["users[0].join_date"] != "wajib diisi" {
		t.Errorf("join_date kosong: status %d, %v", res.Code, f)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/users", fmt.Sprintf(`{"users":[
		{"role":"tutor","username":"tutor.a","name":"A","password":"password123","branch_id":%q,"join_date":"2026-02-01","end_date":"2026-01-31"}]}`, branch), admin)
	if f := errorFields(t, res.Body.String()); f["users[0].end_date"] != "tidak boleh sebelum join_date (2026-02-01)" {
		t.Errorf("end_date sebelum join_date: status %d, %v", res.Code, f)
	}

	// Berakhir hari ini → masih bisa login (inklusif). Berakhir kemarin → tidak bisa.
	res = sendRequest(r, http.MethodPost, "/api/v1/users", fmt.Sprintf(`{"users":[
		{"role":"tutor","username":"tutor.hariini","name":"Hari Ini","password":"password123","branch_id":%[1]q,"join_date":"2025-01-01","end_date":%[2]q},
		{"role":"tutor","username":"tutor.kemarin","name":"Kemarin","password":"password123","branch_id":%[1]q,"join_date":"2025-01-01","end_date":%[3]q}]}`,
		branch, today.Format(time.DateOnly), yesterday), admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("tambah: status %d. body: %s", res.Code, res.Body)
	}
	var created []struct {
		ID       string  `json:"id"`
		JoinDate string  `json:"join_date"`
		EndDate  *string `json:"end_date"`
		Ended    bool    `json:"ended"`
	}
	decodeData(t, res, &created)
	if created[0].JoinDate != "2025-01-01" || created[0].Ended || !created[1].Ended || created[1].EndDate == nil || *created[1].EndDate != yesterday {
		t.Errorf("tanggal di respons: %+v", created)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.hariini","password":"password123"}`, ""); res.Code != http.StatusOK {
		t.Errorf("berakhir hari ini: status %d, seharusnya masih bisa login", res.Code)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.kemarin","password":"password123"}`, "")
	if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "Masa aktif akun Anda sudah berakhir") {
		t.Errorf("berakhir kemarin: status %d, body %s", res.Code, res.Body)
	}

	// Diubah: end_date dimundurkan ke kemarin → sesi login yang sedang berjalan dicabut.
	var session struct {
		RefreshToken string `json:"refresh_token"`
	}
	decodeData(t, sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.hariini","password":"password123"}`, ""), &session)
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+created[0].ID, fmt.Sprintf(
		`{"username":"tutor.hariini","name":"Hari Ini","branch_id":%q,"status":"active","join_date":"2025-01-01","end_date":%q}`, branch, yesterday), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah end_date: status %d. body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, ""); res.Code == http.StatusOK {
		t.Error("refresh token akun yang sudah berakhir seharusnya ditolak")
	}
}

func TestUsers_CekExcelValid(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	branch := newBranch(t, db, "Surabaya Timur")
	grade5 := masterID(t, db, "grade", "GRADE_5")
	sd := masterID(t, db, "education_level", "SD")

	data := buildExcel(t, "user", []excelRow{
		{"role": "Siswa", "username": "Siswa.Andi", "name": "Andi Wijaya", "password": "password123",
			"phone": "81234567890", "parent_phone": "081398765432", "email": "andi@example.com",
			"branch_name": "surabaya timur", "grade_name": "kelas 5", "school_origin": "SDN 1",
			"join_date": "46037", "end_date": "31/12/2026", "date_of_birth": "2014-02-01", "target_school": "SMPN 1"},
		nil, // baris kosong dilewati
		{"role": "tutor", "username": "tutor.sari", "name": "Sari Dewi", "password": "rahasia123",
			"branch_name": "Surabaya Timur", "join_date": "2026-01-10", "status": "nonaktif"},
	}, nil)
	res := sendUserExcel(r, "user.xlsx", data, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("cek excel: status %d. body: %s", res.Code, res.Body)
	}
	var checked struct {
		Total int              `json:"total"`
		Users []map[string]any `json:"users"`
	}
	decodeData(t, res, &checked)
	andi, sari := checked.Users[0], checked.Users[1]
	if checked.Total != 2 || andi["row"] != float64(2) || sari["row"] != float64(4) ||
		andi["role"] != "student" || andi["username"] != "siswa.andi" || andi["phone"] != "081234567890" ||
		andi["branch_id"] != branch || andi["grade_id"] != grade5 || andi["education_level_id"] != sd ||
		andi["join_date"] != "2026-01-15" || andi["end_date"] != "2026-12-31" || andi["status"] != "active" ||
		sari["status"] != "inactive" || sari["grade_id"] != nil {
		t.Fatalf("hasil cek: %+v", checked)
	}

	// Data hasil cek langsung bisa dikirim ke tambah user.
	payload, _ := json.Marshal(map[string]any{"users": checked.Users})
	res = sendRequest(r, http.MethodPost, "/api/v1/users", string(payload), admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("simpan hasil cek: status %d. body: %s", res.Code, res.Body)
	}
	var n int
	db.Raw(`SELECT COUNT(*) FROM users WHERE username IN ('siswa.andi', 'tutor.sari')`).Scan(&n)
	if n != 2 {
		t.Errorf("tersimpan %d user, seharusnya 2", n)
	}
}

func TestUsers_CekExcelError(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	newBranch(t, db, "Surabaya Timur")
	createUser(t, db, loginmodel.RoleTutor, "tutor.lama", loginmodel.UserStatusActive)

	// Kolom: A role, B username, C name, D password, ..., H branch_name, I education_level_name,
	// J grade_name, L join_date, M end_date.
	data := buildExcel(t, "user", []excelRow{
		{"role": "guru besar", "username": "siswa.x", "name": "X", "password": "password123", "branch_name": "Cabang Hilang", "join_date": "2026-01-01"},
		{"role": "admin", "username": "admin.baru", "name": "Admin", "password": "password123", "branch_name": "Surabaya Timur", "join_date": "2026-01-01"},
		{"role": "student", "username": "siswa.y", "name": "Y", "password": "pendek", "branch_name": "Surabaya Timur",
			"education_level_name": "SD", "grade_name": "Kelas 7", "join_date": "2026-02-01", "end_date": "2026-01-01"},
		{"role": "tutor", "username": "tutor.lama", "name": "Lama", "password": "password123", "branch_name": "Surabaya Timur",
			"grade_name": "Kelas 5", "join_date": "kemarin"},
		{"role": "tutor", "username": "SISWA.X", "name": "Kembar", "password": "password123", "branch_name": "Surabaya Timur", "join_date": "2026-01-01"},
	}, nil)
	res := sendUserExcel(r, "user.xlsx", data, admin)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	var body struct {
		Message string `json:"message"`
		Errors  []struct {
			Field, Message, Column string
			Row                    int
		} `json:"errors"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	got := map[string]string{}
	for _, e := range body.Errors {
		got[fmt.Sprintf("%s%d", e.Column, e.Row)] = e.Message
	}
	want := map[string]string{
		"A2": "isi salah satu: student (siswa), tutor, admin, super_admin",
		"H2": `cabang "Cabang Hilang" tidak ditemukan`,
		"A3": "admin tidak boleh menambah akun admin",
		"D4": "minimal 8 karakter",
		"J4": "Kelas 7 bukan bagian dari jenjang SD",
		"M4": "tidak boleh sebelum join_date (2026-02-01)",
		"B5": "sudah dipakai akun lain",
		"J5": "hanya untuk siswa, kosongkan untuk tutor",
		"L5": "tanggal tidak valid, contoh 2026-01-15",
		"B6": "sama dengan username di baris 2",
	}
	for cell, msg := range want {
		if got[cell] != msg {
			t.Errorf("%s = %q, seharusnya %q", cell, got[cell], msg)
		}
	}
	if !strings.HasPrefix(body.Message, "File Excel berisi") {
		t.Errorf("pesan: %q", body.Message)
	}

	// Header salah: kolom tidak dikenal & kolom wajib hilang.
	data = buildExcel(t, "user", []excelRow{{"role": "tutor"}}, func(f *excelize.File) {
		f.SetCellStr("Template", "B1", "nama_user") // username hilang, kolom tidak dikenal
	})
	res = sendUserExcel(r, "user.xlsx", data, admin)
	if !strings.Contains(res.Body.String(), "kolom tidak dikenal") || !strings.Contains(res.Body.String(), "kolom wajib tidak ada di baris 1") {
		t.Errorf("header salah: %s", res.Body)
	}

	// Bukan xlsx, file kosong, dan lebih dari 100 baris.
	if res := sendUserExcel(r, "user.csv", []byte("a,b"), admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("bukan xlsx: status %d", res.Code)
	}
	if res := sendUserExcel(r, "user.xlsx", buildExcel(t, "user", nil, nil), admin); !strings.Contains(res.Body.String(), "tidak berisi data user") {
		t.Errorf("tanpa data: %s", res.Body)
	}
	many := make([]excelRow, 101)
	for i := range many {
		many[i] = excelRow{"role": "tutor", "username": fmt.Sprintf("tutor.m%d", i)}
	}
	if res := sendUserExcel(r, "user.xlsx", buildExcel(t, "user", many, nil), admin); !strings.Contains(res.Body.String(), "Maksimal 100 user per file") {
		t.Errorf("101 baris: %.200s", res.Body)
	}
	// Siswa & tutor tidak boleh.
	if res := sendUserExcel(r, "user.xlsx", data, tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("cek oleh tutor: status %d", res.Code)
	}
}
