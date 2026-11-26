// Test menu User (panel admin): list + filter + jumlah per role, detail, ubah, hapus (soft delete).
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// staffUser membuat user (bukan siswa) bernama dan mengembalikan id & token-nya.
func staffUser(t *testing.T, db *gorm.DB, role, username, name string) (string, string) {
	t.Helper()
	id := createUser(t, db, role, username, loginmodel.UserStatusActive)
	db.Exec(`UPDATE users SET real_name = ? WHERE id = ?`, name, id)
	accessToken, err := token.NewAccessToken(id, role, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return id, accessToken
}

// masterID mengambil id master data (jenjang / kelas) dari code-nya.
func masterID(t *testing.T, db *gorm.DB, itemType, code string) string {
	t.Helper()
	var id string
	db.Raw(`SELECT id FROM master_data_items WHERE type = ? AND code = ?`, itemType, code).Scan(&id)
	if id == "" {
		t.Fatalf("master data %s %s tidak ada", itemType, code)
	}
	return id
}

// newBranch membuat cabang aktif dan mengembalikan id-nya.
func newBranch(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	var id string
	if err := db.Raw(`INSERT INTO branches (name) VALUES (?) RETURNING id`, name).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

type userItem struct {
	ID             string   `json:"id"`
	Username       string   `json:"username"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	Branch         *listRef `json:"branch"`
	EducationLevel *listRef `json:"education_level"`
	Grade          *listRef `json:"grade"`
	StudentCode    *string  `json:"student_code"`
	CoinBalance    *int     `json:"coin_balance"`
	Status         string   `json:"status"`
	// Detail
	ParentPhone  *string  `json:"parent_phone"`
	DateOfBirth  *string  `json:"date_of_birth"`
	TargetSchool *string  `json:"target_school"`
	UpdatedBy    *listRef `json:"updated_by"`
}

func listUsers(t *testing.T, r *gin.Engine, accessToken, query string) ([]userItem, listMeta) {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/users"+query, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("list user %s: status %d. body: %s", query, res.Code, res.Body)
	}
	var items []userItem
	decodeData(t, res, &items)
	return items, decodeMeta(t, res)
}

func userNames(items []userItem) string {
	names := []string{}
	for _, u := range items {
		names = append(names, u.Name)
	}
	return strings.Join(names, ",")
}

func TestUsers_ListFilterDanJumlahPerRole(t *testing.T) {
	r, db := setupRouter(t)
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	staffUser(t, db, loginmodel.RoleSuperAdmin, "super.utama", "Super Utama")
	staffUser(t, db, loginmodel.RoleTutor, "tutor.uji", "Rina Tutor")
	_, studentToken := namedStudent(t, db, "Ani Lestari", "GRADE_5")
	namedStudent(t, db, "Budi Santoso", "GRADE_5")
	namedStudent(t, db, "Citra Ayu", "GRADE_7")
	deleted, _ := namedStudent(t, db, "Dodi Terhapus", "GRADE_5")
	db.Exec(`UPDATE users SET deleted_at = NOW() WHERE id = ?`, deleted)
	// Urutan terbaru dulu: dibuat berurutan di atas, jadi bedakan created_at supaya pasti.
	db.Exec(`UPDATE users SET created_at = NOW() - (INTERVAL '1 minute' * (SELECT COUNT(*) FROM users u2 WHERE u2.username > users.username))`)

	all, meta := listUsers(t, r, admin, "")
	if meta.Total != 6 || len(all) != 6 {
		t.Errorf("semua: %d user (total %d), seharusnya 6 (yang dihapus tidak ikut)", len(all), meta.Total)
	}
	grade5 := masterID(t, db, "grade", "GRADE_5")
	sd := masterID(t, db, "education_level", "SD")
	for _, tc := range []struct {
		query, want string
	}{
		{"?role=student&search=a", "Ani Lestari,Budi Santoso,Citra Ayu"},
		{"?grade_id=" + grade5, "Ani Lestari,Budi Santoso"},
		{"?education_level_id=" + sd + "&role=student", "Ani Lestari,Budi Santoso"},
		{"?search=LESTARI", "Ani Lestari"},
		{"?search=tutor.uj", "Rina Tutor"}, // username
		{"?role=super_admin", "Super Utama"},
	} {
		items, _ := listUsers(t, r, admin, tc.query)
		got := strings.Split(userNames(items), ",")
		if len(items) == 0 || len(got) != len(strings.Split(tc.want, ",")) || !sameSet(got, strings.Split(tc.want, ",")) {
			t.Errorf("filter %s: %s, seharusnya %s", tc.query, userNames(items), tc.want)
		}
	}

	// Data siswa ikut di list: kelas, jenjang, kode siswa, coin.
	students, _ := listUsers(t, r, admin, "?search=Citra")
	if s := students[0]; s.Grade == nil || s.Grade.Name != "Kelas 7" || s.EducationLevel == nil || s.StudentCode == nil || s.CoinBalance == nil {
		t.Errorf("data siswa di list: %+v", s)
	}

	// Pagination.
	page2, meta := listUsers(t, r, admin, "?per_page=4&page=2")
	if len(page2) != 2 || meta.Total != 6 {
		t.Errorf("halaman 2: %d user, total %d", len(page2), meta.Total)
	}

	// Jumlah per role (tab), filter selain role tetap berlaku.
	var counts struct {
		All, SuperAdmin, Admin, Tutor, Student int
	}
	var raw map[string]int
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/users/role-counts?role=tutor", "", admin), &raw)
	counts.All, counts.SuperAdmin, counts.Admin, counts.Tutor, counts.Student = raw["all"], raw["super_admin"], raw["admin"], raw["tutor"], raw["student"]
	if counts.All != 6 || counts.SuperAdmin != 1 || counts.Admin != 1 || counts.Tutor != 1 || counts.Student != 3 {
		t.Errorf("jumlah per role: %v", raw)
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/users/role-counts?grade_id="+grade5, "", admin), &raw)
	if raw["all"] != 2 || raw["student"] != 2 || raw["tutor"] != 0 {
		t.Errorf("jumlah per role filter kelas: %v", raw)
	}

	// Validasi query & hak akses.
	if res := sendRequest(r, http.MethodGet, "/api/v1/users?role=guru", "", admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("role tidak valid: status %d", res.Code)
	}
	for _, tok := range []string{studentToken, tokenFor(t, loginmodel.RoleTutor)} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/users", "", tok); res.Code != http.StatusForbidden {
			t.Errorf("list oleh bukan admin: status %d", res.Code)
		}
	}
}

// sameSet = true kalau a & b berisi elemen yang sama (urutan bebas).
func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, x := range a {
		seen[x]++
	}
	for _, x := range b {
		seen[x]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestUsers_DetailDanUbah(t *testing.T) {
	r, db := setupRouter(t)
	superID, superAdmin := staffUser(t, db, loginmodel.RoleSuperAdmin, "super.utama", "Super Utama")
	_, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	otherAdmin, _ := staffUser(t, db, loginmodel.RoleAdmin, "admin.lain", "Admin Lain")
	tutor, _ := staffUser(t, db, loginmodel.RoleTutor, "tutor.uji", "Rina Tutor")
	ani, _ := namedStudent(t, db, "Ani", "GRADE_5")
	branch := newBranch(t, db, "Surabaya Timur")
	sd, smp := masterID(t, db, "education_level", "SD"), masterID(t, db, "education_level", "SMP")
	grade5, grade7 := masterID(t, db, "grade", "GRADE_5"), masterID(t, db, "grade", "GRADE_7")

	// Detail siswa.
	var d userItem
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/users/"+ani, "", admin), &d)
	if d.ID != ani || d.Role != "student" || d.StudentCode == nil || d.Grade == nil || d.UpdatedBy != nil {
		t.Errorf("detail siswa: %+v", d)
	}
	for _, id := range []string{uuid.NewString(), "bukan-uuid"} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/users/"+id, "", admin); res.Code != http.StatusNotFound {
			t.Errorf("detail %s: status %d", id, res.Code)
		}
	}

	// Ubah siswa: pindah ke SMP Kelas 7, data siswa, nonaktif → sesi login dicabut.
	db.Exec(`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, NOW() + INTERVAL '1 day')`, ani, strings.Repeat("a", 64))
	body := fmt.Sprintf(`{"join_date":"2026-01-01","username":" Siswa.Ani ","name":"Ani Lestari","email":"ani@example.com","parent_phone":"081234",
		"branch_id":%q,"education_level_id":%q,"grade_id":%q,"status":"inactive",
		"date_of_birth":"2013-04-02","target_school":"SMAN 5 Surabaya"}`, branch, smp, grade7)
	res := sendRequest(r, http.MethodPut, "/api/v1/users/"+ani, body, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah siswa: status %d. body: %s", res.Code, res.Body)
	}
	decodeData(t, res, &d)
	if d.Username != "siswa.ani" || d.Name != "Ani Lestari" || d.Status != "inactive" || d.Grade == nil || d.Grade.ID != grade7 ||
		d.Branch == nil || d.Branch.ID != branch || d.DateOfBirth == nil || *d.DateOfBirth != "2013-04-02" ||
		d.TargetSchool == nil || d.UpdatedBy == nil {
		t.Errorf("setelah diubah: %+v", d)
	}
	var revoked bool
	db.Raw(`SELECT revoked_at IS NOT NULL FROM refresh_tokens WHERE user_id = ?`, ani).Scan(&revoked)
	if !revoked {
		t.Error("sesi login siswa yang dinonaktifkan seharusnya dicabut")
	}

	// Validasi: format username, kelas bukan milik jenjang, cabang kosong, tanggal lahir di masa depan.
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+ani, fmt.Sprintf(
		`{"join_date":"2026-01-01","username":"ab","name":"Ani","education_level_id":%q,"grade_id":%q,"status":"active","date_of_birth":"2999-01-01"}`, sd, grade7), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["username"] == "" || f["grade_id"] != "Kelas 7 bukan bagian dari jenjang SD" ||
		f["branch_id"] != "wajib diisi untuk siswa" || f["date_of_birth"] == "" {
		t.Errorf("validasi siswa: status %d, errors %v", res.Code, f)
	}
	// Tutor tidak boleh punya jenjang/kelas; username tidak boleh dipakai akun lain.
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+tutor, fmt.Sprintf(
		`{"join_date":"2026-01-01","username":"siswa.ani","name":"Rina","branch_id":%q,"grade_id":%q,"status":"active"}`, branch, grade5), admin)
	if f := errorFields(t, res.Body.String()); f["grade_id"] != "hanya untuk siswa, kosongkan untuk tutor" {
		t.Errorf("tutor dengan kelas: %v", f)
	}
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+tutor, fmt.Sprintf(
		`{"join_date":"2026-01-01","username":"siswa.ani","name":"Rina","branch_id":%q,"status":"active"}`, branch), admin)
	if f := errorFields(t, res.Body.String()); f["username"] != "sudah dipakai akun lain" {
		t.Errorf("username kembar: status %d, %v", res.Code, f)
	}

	// Reset password tutor → login dengan password baru berhasil, permintaan lupa kata sandi ikut selesai.
	db.Exec(`INSERT INTO password_reset_requests (user_id, identifier) VALUES (?, 'tutor.uji')`, tutor)
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+tutor, fmt.Sprintf(
		`{"join_date":"2026-01-01","username":"tutor.uji","name":"Rina","branch_id":%q,"status":"active","password":"PasswordBaru123"}`, branch), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("reset password: status %d. body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.uji","password":"PasswordBaru123"}`, ""); res.Code != http.StatusOK {
		t.Errorf("login dengan password baru: status %d", res.Code)
	}
	var resetStatus string
	db.Raw(`SELECT status FROM password_reset_requests WHERE user_id = ?`, tutor).Scan(&resetStatus)
	if resetStatus != "resolved" {
		t.Errorf("permintaan lupa kata sandi setelah reset lewat ubah user: %q", resetStatus)
	}

	// Hak akses: admin tidak boleh mengubah admin lain / super admin; super admin boleh.
	adminBody := fmt.Sprintf(`{"join_date":"2026-01-01","username":"admin.lain","name":"Admin Lain Baru","branch_id":%q,"status":"active"}`, branch)
	for _, id := range []string{otherAdmin, superID} {
		if res := sendRequest(r, http.MethodPut, "/api/v1/users/"+id, adminBody, admin); res.Code != http.StatusForbidden {
			t.Errorf("admin mengubah %s: status %d, seharusnya 403", id, res.Code)
		}
	}
	if res := sendRequest(r, http.MethodPut, "/api/v1/users/"+otherAdmin, adminBody, superAdmin); res.Code != http.StatusOK {
		t.Errorf("super admin mengubah admin: status %d. body: %s", res.Code, res.Body)
	}
	// Super admin: tanpa cabang, tidak bisa menonaktifkan diri sendiri.
	res = sendRequest(r, http.MethodPut, "/api/v1/users/"+superID, fmt.Sprintf(
		`{"join_date":"2026-01-01","username":"super.utama","name":"Super","branch_id":%q,"status":"inactive"}`, branch), superAdmin)
	f = errorFields(t, res.Body.String())
	if f["branch_id"] != "super admin berlaku untuk semua cabang, kosongkan" || f["status"] != "tidak bisa menonaktifkan akun sendiri" {
		t.Errorf("ubah diri sendiri: %v", f)
	}
}

func TestUsers_HapusSoftDelete(t *testing.T) {
	r, db := setupRouter(t)
	adminID, admin := staffUser(t, db, loginmodel.RoleAdmin, "admin.utama", "Admin Utama")
	superID, _ := staffUser(t, db, loginmodel.RoleSuperAdmin, "super.utama", "Super Utama")
	tutor, _ := staffUser(t, db, loginmodel.RoleTutor, "tutor.uji", "Rina Tutor")
	ani, _ := namedStudent(t, db, "Ani", "GRADE_5")
	db.Exec(`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, NOW() + INTERVAL '1 day')`, ani, strings.Repeat("b", 64))

	// Ada yang tidak boleh → tidak ada yang terhapus.
	res := sendRequest(r, http.MethodDelete, "/api/v1/users",
		fmt.Sprintf(`{"ids":[%q,%q,%q,%q]}`, ani, adminID, superID, uuid.NewString()), admin)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["ids[1]"] != "tidak bisa menghapus akun sendiri" ||
		f["ids[2]"] != "admin tidak boleh menghapus akun super admin (Super Utama)" || f["ids[3]"] != "user tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus tidak valid: status %d, errors %v", res.Code, f)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/users/"+ani, "", admin); res.Code != http.StatusOK {
		t.Errorf("siswa seharusnya belum terhapus: status %d", res.Code)
	}

	res = sendRequest(r, http.MethodDelete, "/api/v1/users", fmt.Sprintf(`{"ids":[%q,%q]}`, ani, tutor), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("hapus: status %d. body: %s", res.Code, res.Body)
	}
	// Soft delete: baris tetap ada (deleted_at & deleted_by), hilang dari list & detail, sesi dicabut,
	// tidak bisa login lagi.
	var row struct {
		Deleted   bool
		DeletedBy *string
	}
	db.Raw(`SELECT deleted_at IS NOT NULL AS deleted, deleted_by FROM users WHERE id = ?`, ani).Scan(&row)
	if !row.Deleted || row.DeletedBy == nil || *row.DeletedBy != adminID {
		t.Errorf("baris user terhapus: %+v", row)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/users/"+ani, "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail user terhapus: status %d, seharusnya 404", res.Code)
	}
	if items, _ := listUsers(t, r, admin, ""); len(items) != 2 {
		t.Errorf("list setelah hapus: %s", userNames(items))
	}
	var revoked bool
	db.Raw(`SELECT revoked_at IS NOT NULL FROM refresh_tokens WHERE user_id = ?`, ani).Scan(&revoked)
	if !revoked {
		t.Error("sesi user terhapus seharusnya dicabut")
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tutor.uji","password":"`+testPassword+`"}`, ""); res.Code == http.StatusOK {
		t.Error("user terhapus seharusnya tidak bisa login")
	}
}
