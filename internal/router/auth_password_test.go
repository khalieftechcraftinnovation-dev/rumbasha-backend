// Test "Ingat saya", ubah kata sandi sendiri, dan lupa kata sandi lewat admin.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

type sessionWithRemember struct {
	sessionData
	RememberMe bool `json:"remember_me"`
}

func loginRemember(t *testing.T, r *gin.Engine, username, password string, remember bool) sessionWithRemember {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login",
		fmt.Sprintf(`{"username":%q,"password":%q,"remember_me":%v}`, username, password, remember), "")
	if res.Code != http.StatusOK {
		t.Fatalf("login gagal: status %d, body: %s", res.Code, res.Body)
	}
	var s sessionWithRemember
	decodeData(t, res, &s)
	return s
}

func refreshCode(r *gin.Engine, refreshToken string) int {
	return sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, refreshToken), "").Code
}

func TestLogin_IngatSaya(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleStudent, "siswa.ingat", loginmodel.UserStatusActive)

	// Tanpa centang: sampai tanggal 1 bulan berikutnya 00:00 WIB (aturan lama).
	normal := loginRemember(t, r, "siswa.ingat", testPassword, false)
	if normal.RememberMe || !normal.RefreshTokenExpiresAt.Equal(clock.StartOfNextMonth(time.Now())) {
		t.Errorf("sesi biasa: remember %v, berakhir %s", normal.RememberMe, normal.RefreshTokenExpiresAt)
	}

	// Dicentang: 30 hari sejak login (boleh melewati tanggal 1), refresh tidak memperpanjang.
	before := time.Now()
	remembered := loginRemember(t, r, "siswa.ingat", testPassword, true)
	want := before.Add(30 * 24 * time.Hour)
	if !remembered.RememberMe || remembered.RefreshTokenExpiresAt.Before(want.Add(-time.Minute)) || remembered.RefreshTokenExpiresAt.After(want.Add(time.Minute)) {
		t.Errorf("sesi ingat saya: remember %v, berakhir %s (seharusnya ±%s)", remembered.RememberMe, remembered.RefreshTokenExpiresAt, want)
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token":%q}`, remembered.RefreshToken), "")
	var refreshed sessionWithRemember
	decodeData(t, res, &refreshed)
	if res.Code != http.StatusOK || !refreshed.RememberMe || !refreshed.RefreshTokenExpiresAt.Equal(remembered.RefreshTokenExpiresAt) {
		t.Errorf("refresh sesi ingat saya: %d %+v", res.Code, refreshed)
	}
	var stored bool
	db.Raw(`SELECT remember_me FROM refresh_tokens WHERE token_hash = ?`, token.HashRefreshToken(refreshed.RefreshToken)).Scan(&stored)
	if !stored {
		t.Error("remember_me tidak tersimpan di refresh_tokens")
	}
	// Akun berakhir tetap memutus sesi "Ingat saya".
	db.Exec(`UPDATE users SET join_date = CURRENT_DATE - 10, end_date = CURRENT_DATE - 2 WHERE username = 'siswa.ingat'`)
	if code := refreshCode(r, refreshed.RefreshToken); code != http.StatusForbidden {
		t.Errorf("refresh akun berakhir: %d", code)
	}
}

func TestChangePassword_UbahSendiri(t *testing.T) {
	r, db := setupRouter(t)
	userID := createUser(t, db, loginmodel.RoleStudent, "siswa.sandi", loginmodel.UserStatusActive)
	hp := loginRemember(t, r, "siswa.sandi", testPassword, false)
	laptop := loginRemember(t, r, "siswa.sandi", testPassword, false)
	db.Exec(`INSERT INTO password_reset_requests (user_id, identifier) VALUES (?, 'siswa.sandi')`, userID)

	change := func(body string) (int, string) {
		res := sendRequest(r, http.MethodPost, "/api/v1/auth/change-password", body, hp.AccessToken)
		return res.Code, res.Body.String()
	}
	for body, field := range map[string]string{
		`{"current_password":"salah123","new_password":"sandiBaru123"}`:                      "current_password",
		fmt.Sprintf(`{"current_password":%q,"new_password":%q}`, testPassword, testPassword): "new_password",
		fmt.Sprintf(`{"current_password":%q,"new_password":"pendek"}`, testPassword):         "new_password",
	} {
		if code, b := change(body); code != http.StatusUnprocessableEntity || errorFields(t, b)[field] == "" {
			t.Errorf("validasi %s: %d %s", body, code, b)
		}
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/change-password", `{"current_password":"a","new_password":"sandiBaru123"}`, ""); res.Code != http.StatusUnauthorized {
		t.Errorf("tanpa login: %d", res.Code)
	}

	code, body := change(fmt.Sprintf(`{"current_password":%q,"new_password":"sandiBaru123","remember_me":true}`, testPassword))
	var fresh sessionWithRemember
	decodeData(t, sendRequestBody(code, body), &fresh)
	if code != http.StatusOK || fresh.AccessToken == "" || !fresh.RememberMe {
		t.Fatalf("ubah kata sandi: %d %s", code, body)
	}
	// Semua sesi lama dicabut, sesi baru tetap jalan.
	if refreshCode(r, hp.RefreshToken) != http.StatusUnauthorized || refreshCode(r, laptop.RefreshToken) != http.StatusUnauthorized {
		t.Error("sesi lama masih bisa dipakai setelah ubah kata sandi")
	}
	if code := refreshCode(r, fresh.RefreshToken); code != http.StatusOK {
		t.Errorf("sesi baru: %d", code)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"username":"siswa.sandi","password":%q}`, testPassword), ""); res.Code != http.StatusUnauthorized {
		t.Errorf("login sandi lama: %d", res.Code)
	}
	login(t, r, "siswa.sandi", "sandiBaru123")
	// Permintaan lupa kata sandi yang menunggu ikut selesai.
	var status string
	db.Raw(`SELECT status FROM password_reset_requests WHERE user_id = ?`, userID).Scan(&status)
	if status != "resolved" {
		t.Errorf("permintaan lupa kata sandi: %s", status)
	}
}

type resetRequestData struct {
	ID   string `json:"id"`
	User struct {
		ID       string  `json:"id"`
		Username string  `json:"username"`
		Role     string  `json:"role"`
		Phone    *string `json:"phone"`
	} `json:"user"`
	Identifier   string    `json:"identifier"`
	Contact      *string   `json:"contact"`
	RequestCount int       `json:"request_count"`
	Status       string    `json:"status"`
	ResolvedBy   *listRef  `json:"resolved_by"`
	RequestedAt  time.Time `json:"requested_at"`
}

func TestForgotPassword_LewatAdmin(t *testing.T) {
	r, db := setupRouter(t)
	student := createUser(t, db, loginmodel.RoleStudent, "siswa.lupa", loginmodel.UserStatusActive)
	db.Exec(`UPDATE users SET email = 'Lupa@Example.com', phone = '0812-3456-7890' WHERE id = ?`, student)
	adminID := createUser(t, db, loginmodel.RoleAdmin, "admin.lupa", loginmodel.UserStatusActive)
	db.Exec(`UPDATE users SET email = 'admin.lupa@example.com' WHERE id = ?`, adminID)
	tutor := createUser(t, db, loginmodel.RoleTutor, "tutor.lupa", loginmodel.UserStatusActive)
	session := login(t, r, "siswa.lupa", testPassword)

	forgot := func(body string) (int, string) {
		res := sendRequest(r, http.MethodPost, "/api/v1/auth/forgot-password", body, "")
		return res.Code, res.Body.String()
	}
	count := func() int {
		var n int
		db.Raw(`SELECT COUNT(*) FROM password_reset_requests`).Scan(&n)
		return n
	}

	// Akun tidak ada: jawaban sama persis, tidak ada yang dicatat.
	codeUnknown, bodyUnknown := forgot(`{"identifier":"tidak.ada"}`)
	codeKnown, bodyKnown := forgot(`{"identifier":"Siswa.Lupa","contact":"0899-1111"}`)
	if codeUnknown != http.StatusOK || codeKnown != http.StatusOK || bodyUnknown != bodyKnown || count() != 1 {
		t.Fatalf("jawaban form lupa kata sandi berbeda / jumlah salah: %d %s | %d %s | %d", codeUnknown, bodyUnknown, codeKnown, bodyKnown, count())
	}
	// Kirim lagi lewat no HP (+62) dan email: permintaan yang sama diperbarui, bukan baris baru.
	forgot(`{"identifier":"+62 812 3456 7890","note":"ganti HP"}`)
	forgot(`{"identifier":"lupa@example.com"}`)
	forgot(`{"identifier":"admin.lupa@example.com"}`)
	forgot(`{"identifier":"tutor.lupa"}`)
	if res := sendRequest(r, http.MethodPost, "/api/v1/auth/forgot-password", `{}`, ""); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("tanpa identifier: %d", res.Code)
	}

	admin := tokenOf(t, adminID, loginmodel.RoleAdmin)
	superID := createUser(t, db, loginmodel.RoleSuperAdmin, "super.lupa", loginmodel.UserStatusActive)
	super := tokenOf(t, superID, loginmodel.RoleSuperAdmin)

	list := func(token, query string) []resetRequestData {
		t.Helper()
		res := sendRequest(r, http.MethodGet, "/api/v1/password-reset-requests"+query, "", token)
		if res.Code != http.StatusOK {
			t.Fatalf("list permintaan: %d %s", res.Code, res.Body)
		}
		var items []resetRequestData
		decodeData(t, res, &items)
		return items
	}
	// Admin hanya melihat permintaan akun tutor & siswa; super admin semua.
	items := list(admin, "")
	byUser := map[string]resetRequestData{}
	for _, i := range items {
		byUser[i.User.Username] = i
	}
	s := byUser["siswa.lupa"]
	if len(items) != 2 || s.RequestCount != 3 || s.Contact == nil || *s.Contact != "0899-1111" || s.Identifier != "lupa@example.com" || s.Status != "pending" {
		t.Errorf("list admin: %+v", items)
	}
	if items := list(super, ""); len(items) != 3 {
		t.Errorf("list super admin: %d", len(items))
	}
	if items := list(admin, "?search=tutor"); len(items) != 1 || items[0].User.ID != tutor {
		t.Errorf("cari: %+v", items)
	}
	var counts map[string]int
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/password-reset-requests/counts", "", admin), &counts)
	if counts["pending"] != 2 {
		t.Errorf("jumlah: %v", counts)
	}

	// Admin tidak boleh mereset akun admin (permintaan milik admin.lupa).
	var adminReq string
	db.Raw(`SELECT id FROM password_reset_requests WHERE user_id = ?`, adminID).Scan(&adminReq)
	if res := sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+adminReq+"/resolve", `{"new_password":"sandiBaru123"}`, admin); res.Code != http.StatusForbidden {
		t.Errorf("admin reset akun admin: %d", res.Code)
	}

	// Reset kata sandi siswa: sesi lama dicabut, login dengan sandi baru.
	if res := sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+s.ID+"/resolve", `{"new_password":"pendek"}`, admin); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("sandi baru pendek: %d", res.Code)
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+s.ID+"/resolve", `{"new_password":"sandiReset123","note":"Sudah dikabari via WA"}`, admin)
	var resolved resetRequestData
	decodeData(t, res, &resolved)
	if res.Code != http.StatusOK || resolved.Status != "resolved" || resolved.ResolvedBy == nil || resolved.ResolvedBy.ID != adminID {
		t.Errorf("reset: %d %s", res.Code, res.Body)
	}
	if refreshCode(r, session.RefreshToken) != http.StatusUnauthorized {
		t.Error("sesi siswa masih berlaku setelah reset")
	}
	login(t, r, "siswa.lupa", "sandiReset123")
	if res := sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+s.ID+"/resolve", `{"new_password":"sandiReset456"}`, admin); res.Code != http.StatusConflict {
		t.Errorf("reset 2x: %d", res.Code)
	}
	// Permintaan baru setelah selesai = baris baru.
	forgot(`{"identifier":"siswa.lupa"}`)
	if count() != 4 {
		t.Errorf("permintaan baru setelah selesai: %d baris", count())
	}

	// Tolak permintaan tutor.
	tutorReq := byUser["tutor.lupa"].ID
	res = sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+tutorReq+"/reject", `{"note":"bukan pemilik akun"}`, admin)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"status":"rejected"`) {
		t.Errorf("tolak: %d %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/password-reset-requests/"+tutorReq+"/reject", "", admin); res.Code != http.StatusConflict {
		t.Errorf("tolak 2x: %d", res.Code)
	}
	login(t, r, "tutor.lupa", testPassword) // sandi tutor tidak berubah
	if items := list(admin, "?status=rejected"); len(items) != 1 {
		t.Errorf("filter ditolak: %d", len(items))
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/password-reset-requests/"+uuid.NewString(), "", admin); res.Code != http.StatusNotFound {
		t.Errorf("detail tidak ada: %d", res.Code)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/password-reset-requests", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor membuka daftar: %d", res.Code)
	}
}

// tokenOf membuat access token untuk user yang sudah ada.
func tokenOf(t *testing.T, userID, role string) string {
	t.Helper()
	accessToken, err := token.NewAccessToken(userID, role, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return accessToken
}
