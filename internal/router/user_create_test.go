// Test tambah user (1-100 sekaligus) di menu User.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func TestUsers_TambahBanyakSekaligus(t *testing.T) {
	r, db := setupRouter(t)
	adminID, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	branch := newBranch(t, db, "Surabaya Timur")
	sd, grade5 := masterID(t, db, "education_level", "SD"), masterID(t, db, "grade", "GRADE_5")
	smp, grade7 := masterID(t, db, "education_level", "SMP"), masterID(t, db, "grade", "GRADE_7")

	body := fmt.Sprintf(`{"users":[
		{"join_date":"2026-01-01","role":"student","username":" Siswa.Andi ","name":"Andi Wijaya","password":"password123","branch_id":%[1]q,
		 "education_level_id":%[2]q,"grade_id":%[3]q,"date_of_birth":"2014-02-01","target_school":"SMPN 1 Surabaya"},
		{"join_date":"2026-01-01","role":"tutor","username":"tutor.sari","name":"Sari Dewi","password":"rahasia123","branch_id":%[1]q,"email":"sari@example.com"},
		{"join_date":"2026-01-01","role":"student","username":"siswa.bela","name":"Bela","password":"password123","branch_id":%[1]q,
		 "education_level_id":%[4]q,"grade_id":%[5]q,"status":"inactive"}
	]}`, branch, sd, grade5, smp, grade7)
	res := sendRequest(r, http.MethodPost, "/api/v1/users", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("tambah user: status %d. body: %s", res.Code, res.Body)
	}
	var created []userItem
	decodeData(t, res, &created)
	// Urutan sesuai request; username dirapikan; data siswa (kode siswa, kelas) ikut.
	if len(created) != 3 || created[0].Username != "siswa.andi" || created[1].Role != "tutor" || created[2].Status != "inactive" ||
		created[0].StudentCode == nil || created[0].Grade == nil || created[0].Grade.ID != grade5 ||
		created[1].StudentCode != nil || created[1].Branch == nil || created[1].Branch.ID != branch {
		t.Fatalf("hasil tambah: %+v", created)
	}
	var d userItem
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/users/"+created[0].ID, "", admin), &d)
	if d.DateOfBirth == nil || *d.DateOfBirth != "2014-02-01" || d.TargetSchool == nil {
		t.Errorf("data siswa tersimpan: %+v", d)
	}
	var createdBy string
	db.Raw(`SELECT created_by FROM users WHERE id = ?`, created[1].ID).Scan(&createdBy)
	if createdBy != adminID {
		t.Errorf("created_by = %q, seharusnya admin pembuat", createdBy)
	}
	// Akun baru langsung bisa login dengan password-nya; akun nonaktif tidak.
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.sari","password":"rahasia123"}`, ""); res.Code != http.StatusOK {
		t.Errorf("login akun baru: status %d", res.Code)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"siswa.bela","password":"password123"}`, ""); res.Code == http.StatusOK {
		t.Error("akun nonaktif seharusnya tidak bisa login")
	}
}

func TestUsers_TambahValidasiSemuaAtauTidakSamaSekali(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	_, superAdmin := staffUser(t, db, loginmodel.RoleSuperAdmin, "super.utama", "Super Utama")
	branch := newBranch(t, db, "Surabaya Timur")
	closed := newBranch(t, db, "Cabang Tutup")
	db.Exec(`UPDATE branches SET status = 'inactive' WHERE id = ?`, closed)
	sd, grade7 := masterID(t, db, "education_level", "SD"), masterID(t, db, "grade", "GRADE_7")
	gone := createUser(t, db, loginmodel.RoleStudent, "siswa.lama", loginmodel.UserStatusActive)
	db.Exec(`UPDATE users SET deleted_at = NOW() WHERE id = ?`, gone)

	var before int
	db.Raw(`SELECT COUNT(*) FROM users`).Scan(&before)

	body := fmt.Sprintf(`{"users":[
		{"join_date":"2026-01-01","role":"tutor","username":"tutor.baru","name":"Tutor Baru","password":"password123","branch_id":%[1]q},
		{"join_date":"2026-01-01","role":"admin","username":"admin.baru","name":"Admin Baru","password":"password123","branch_id":%[1]q},
		{"join_date":"2026-01-01","role":"tutor","username":"TUTOR.BARU","name":"Kembar","password":"password123","branch_id":%[1]q},
		{"join_date":"2026-01-01","role":"student","username":"siswa.lama","name":"Siswa","password":"password123","branch_id":%[2]q,
		 "education_level_id":%[3]q,"grade_id":%[4]q},
		{"join_date":"2026-01-01","role":"tutor","username":"x","name":" ","password":"password123","grade_id":%[4]q}
	]}`, branch, closed, sd, grade7)
	res := sendRequest(r, http.MethodPost, "/api/v1/users", body, admin)
	f := errorFields(t, res.Body.String())
	want := map[string]string{
		"users[1].role":      "admin tidak boleh menambah akun admin",
		"users[2].username":  "sama dengan users[0]",
		"users[3].username":  "sudah dipakai akun lain", // termasuk akun yang sudah dihapus
		"users[3].branch_id": "cabang tidak ditemukan atau tidak aktif",
		"users[3].grade_id":  "Kelas 7 bukan bagian dari jenjang SD",
		"users[4].username":  "3-30 karakter, hanya huruf kecil, angka, titik, atau garis bawah",
		"users[4].name":      "wajib diisi",
		"users[4].branch_id": "wajib diisi untuk tutor",
		"users[4].grade_id":  "hanya untuk siswa, kosongkan untuk tutor",
	}
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	for field, msg := range want {
		if f[field] != msg {
			t.Errorf("%s = %q, seharusnya %q", field, f[field], msg)
		}
	}
	var after int
	db.Raw(`SELECT COUNT(*) FROM users`).Scan(&after)
	if after != before {
		t.Errorf("jumlah user %d → %d, seharusnya tidak ada yang tersimpan", before, after)
	}

	// Validasi format dari binding: password pendek & role salah → path per akun.
	res = sendRequest(r, http.MethodPost, "/api/v1/users", `{"users":[{"role":"guru","username":"a.b","name":"A","password":"123"}]}`, admin)
	if f := errorFields(t, res.Body.String()); f["users[0].role"] == "" || f["users[0].password"] == "" {
		t.Errorf("validasi format: %v", f)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/users", `{"users":[]}`, admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("tanpa user: status %d", res.Code)
	}

	// Super admin boleh menambah admin & super admin (super admin tanpa cabang).
	res = sendRequest(r, http.MethodPost, "/api/v1/users", fmt.Sprintf(`{"users":[
		{"join_date":"2026-01-01","role":"admin","username":"admin.baru","name":"Admin Baru","password":"password123","branch_id":%q},
		{"join_date":"2026-01-01","role":"super_admin","username":"super.baru","name":"Super Baru","password":"password123"}]}`, branch), superAdmin)
	if res.Code != http.StatusCreated {
		t.Errorf("super admin menambah admin: status %d. body: %s", res.Code, res.Body)
	}
	// Siswa & tutor tidak boleh.
	if res := sendRequest(r, http.MethodPost, "/api/v1/users", body, tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tambah oleh tutor: status %d", res.Code)
	}
}

func TestUsers_TambahMaksimal100(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	branch := newBranch(t, db, "Surabaya Timur")
	sd, grade5 := masterID(t, db, "education_level", "SD"), masterID(t, db, "grade", "GRADE_5")

	items := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"join_date":"2026-01-01","role":"student","username":"siswa.massal%03d","name":"Siswa %d","password":"password%03d",
				"branch_id":%q,"education_level_id":%q,"grade_id":%q}`, i, i, i, branch, sd, grade5)
		}
		return `{"users":[` + strings.Join(parts, ",") + `]}`
	}

	if res := sendRequest(r, http.MethodPost, "/api/v1/users", items(101), admin); res.Code != http.StatusUnprocessableEntity ||
		errorFields(t, res.Body.String())["users"] == "" {
		t.Errorf("101 user: status %d, body %.200s", res.Code, res.Body)
	}

	start := time.Now()
	res := sendRequest(r, http.MethodPost, "/api/v1/users", items(100), admin)
	elapsed := time.Since(start)
	if res.Code != http.StatusCreated {
		t.Fatalf("100 user: status %d. body: %.300s", res.Code, res.Body)
	}
	var created []userItem
	decodeData(t, res, &created)
	if len(created) != 100 || created[0].Username != "siswa.massal000" || created[99].Username != "siswa.massal099" {
		t.Errorf("100 user: %d dibuat", len(created))
	}
	// Kode siswa urut sesuai urutan di request.
	if *created[0].StudentCode >= *created[99].StudentCode {
		t.Errorf("kode siswa tidak urut: %s .. %s", *created[0].StudentCode, *created[99].StudentCode)
	}
	t.Logf("tambah 100 user: %s", elapsed.Round(time.Millisecond))
}
