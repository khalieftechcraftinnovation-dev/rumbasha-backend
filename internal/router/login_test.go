// Test endpoint /auth (fitur login): login, refresh, logout, dan me (termasuk cek paket harian).
package router

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// sessionData adalah isi "data" dari response login/refresh.
type sessionData struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	User                  struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

// meData adalah isi "data" dari response GET /auth/me.
type meData struct {
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	Access *struct {
		Subscription *struct {
			PlanName string `json:"plan_name"`
			Channel  string `json:"channel"`
		} `json:"subscription"`
		Features map[string]bool `json:"features"`
	} `json:"access"`
}

// login memanggil POST /auth/login dan mengembalikan isi sesinya.
func login(t *testing.T, r *gin.Engine, username, password string) sessionData {
	t.Helper()
	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login",
		`{"username":"`+username+`","password":"`+password+`"}`, "")
	if res.Code != http.StatusOK {
		t.Fatalf("login gagal: status %d, body: %s", res.Code, res.Body)
	}
	var session sessionData
	decodeData(t, res, &session)
	return session
}

// createPlanAndSubscription membuat paket dan memberikannya ke siswa
// dengan rentang tanggal relatif dari hari ini (WIB). Contoh: -10, 20.
func createPlanAndSubscription(t *testing.T, db *gorm.DB, studentID, channel string, startOffset, endOffset int) {
	t.Helper()
	err := db.Exec(`
		INSERT INTO subscription_plans (name, price_monthly, has_module, has_daily_quiz)
		VALUES ('Reguler SD', 350000, TRUE, TRUE)`).Error
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`
		INSERT INTO student_subscriptions (student_id, plan_id, channel, period, price_paid, start_date, end_date)
		SELECT ?, id, ?, 'monthly', 350000,
		       (NOW() AT TIME ZONE 'Asia/Jakarta')::date + ?::int,
		       (NOW() AT TIME ZONE 'Asia/Jakarta')::date + ?::int
		FROM subscription_plans`, studentID, channel, startOffset, endOffset).Error
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogin_Berhasil(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)

	session := login(t, r, "admin.test", testPassword)

	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("access_token dan refresh_token seharusnya terisi")
	}
	if session.User.Role != loginmodel.RoleAdmin {
		t.Errorf("role = %q, seharusnya admin", session.User.Role)
	}

	// Sesi harus berakhir tepat tanggal 1 bulan berikutnya, jam 00:00 WIB.
	wib := time.FixedZone("WIB", 7*60*60)
	now := time.Now().In(wib)
	expected := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, wib)
	if !session.RefreshTokenExpiresAt.Equal(expected) {
		t.Errorf("akhir sesi = %v, seharusnya %v", session.RefreshTokenExpiresAt.In(wib), expected)
	}
	// Access token tidak boleh berlaku melewati akhir sesi.
	if session.AccessTokenExpiresAt.After(session.RefreshTokenExpiresAt) {
		t.Error("access token tidak boleh berlaku melewati akhir sesi")
	}
}

func TestLogin_UsernameTidakPekaHurufBesarDanSpasi(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleTutor, "tutor.test", loginmodel.UserStatusActive)

	session := login(t, r, " Tutor.Test ", testPassword)

	if session.User.Username != "tutor.test" {
		t.Errorf("username = %q, seharusnya tutor.test", session.User.Username)
	}
}

func TestLogin_PasswordSalahDanUsernameTidakAdaPesannyaSama(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)

	wrongPassword := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"admin.test","password":"salah"}`, "")
	unknownUser := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"tidak.ada","password":"salah"}`, "")

	if wrongPassword.Code != http.StatusUnauthorized || unknownUser.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d dan %d, seharusnya 401 dan 401", wrongPassword.Code, unknownUser.Code)
	}
	// Pesan harus sama, supaya tidak bisa dipakai menebak username yang terdaftar.
	if wrongPassword.Body.String() != unknownUser.Body.String() {
		t.Errorf("pesan seharusnya sama:\n%s\n%s", wrongPassword.Body, unknownUser.Body)
	}
}

func TestLogin_AkunNonaktifDitolak(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleStudent, "siswa.nonaktif", loginmodel.UserStatusInactive)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"siswa.nonaktif","password":"`+testPassword+`"}`, "")

	if res.Code != http.StatusForbidden {
		t.Errorf("status = %d, seharusnya 403. body: %s", res.Code, res.Body)
	}
}

func TestLogin_ValidasiGagal(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login", `{"username":"admin"}`, "")

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	if !strings.Contains(res.Body.String(), `"field":"password"`) {
		t.Errorf("seharusnya ada error untuk field password. body: %s", res.Body)
	}
}

func TestRefresh_TokenLamaTidakBisaDipakaiLagi(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	first := login(t, r, "admin.test", testPassword)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+first.RefreshToken+`"}`, "")
	if res.Code != http.StatusOK {
		t.Fatalf("refresh: status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	var second sessionData
	decodeData(t, res, &second)

	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token baru seharusnya berbeda dari yang lama")
	}
	// Refresh tidak boleh memperpanjang sesi melewati awal bulan.
	if !second.RefreshTokenExpiresAt.Equal(first.RefreshTokenExpiresAt) {
		t.Errorf("akhir sesi berubah dari %v menjadi %v", first.RefreshTokenExpiresAt, second.RefreshTokenExpiresAt)
	}

	// Token lama sudah ditukar, jadi tidak boleh dipakai lagi.
	res = sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+first.RefreshToken+`"}`, "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("refresh token lama: status = %d, seharusnya 401. body: %s", res.Code, res.Body)
	}
}

func TestRefresh_AkunDinonaktifkanSetelahLogin(t *testing.T) {
	r, db := setupRouter(t)
	userID := createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, r, "admin.test", testPassword)

	db.Exec(`UPDATE users SET status = 'inactive' WHERE id = ?`, userID)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, "")
	if res.Code != http.StatusForbidden {
		t.Errorf("status = %d, seharusnya 403. body: %s", res.Code, res.Body)
	}
}

func TestRefresh_SesiSudahLewatAwalBulan(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, r, "admin.test", testPassword)

	// Simulasikan awal bulan sudah lewat: akhir sesi dimundurkan ke masa lalu.
	db.Exec(`UPDATE refresh_tokens SET expires_at = NOW() - INTERVAL '1 second'`)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, seharusnya 401 (wajib login ulang). body: %s", res.Code, res.Body)
	}
}

func TestLogout_RefreshTokenTidakBisaDipakaiLagi(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, r, "admin.test", testPassword)
	body := `{"refresh_token":"` + session.RefreshToken + `"}`

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/logout", body, "")
	if res.Code != http.StatusOK {
		t.Fatalf("logout: status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodPost, "/api/v1/auth/refresh", body, "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("refresh setelah logout: status = %d, seharusnya 401", res.Code)
	}

	// Logout dua kali tetap aman.
	res = sendRequest(r, http.MethodPost, "/api/v1/auth/logout", body, "")
	if res.Code != http.StatusOK {
		t.Errorf("logout kedua: status = %d, seharusnya 200", res.Code)
	}
}

func TestMe_TanpaTokenAtauTokenPalsuDitolak(t *testing.T) {
	r, _ := setupRouter(t)

	res := sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", "")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("tanpa token: status = %d, seharusnya 401", res.Code)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", "token.palsu.sekali")
	if res.Code != http.StatusUnauthorized {
		t.Errorf("token palsu: status = %d, seharusnya 401", res.Code)
	}
}

func TestMe_AdminTidakPunyaDataAkses(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)
	session := login(t, r, "admin.test", testPassword)

	res := sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", session.AccessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200. body: %s", res.Code, res.Body)
	}
	var me meData
	decodeData(t, res, &me)
	if me.User.Username != "admin.test" {
		t.Errorf("username = %q, seharusnya admin.test", me.User.Username)
	}
	if me.Access != nil {
		t.Error("admin seharusnya tidak punya data akses paket")
	}
}

func TestMe_SiswaPaketOnlineAktifDenganAksesKhusus(t *testing.T) {
	r, db := setupRouter(t)
	studentID := createUser(t, db, loginmodel.RoleStudent, "siswa.test", loginmodel.UserStatusActive)
	createPlanAndSubscription(t, db, studentID, "online", -10, 20) // aktif hari ini
	db.Exec(`INSERT INTO student_feature_overrides (student_id, feature, is_enabled) VALUES (?, 'rumath', TRUE)`, studentID)

	session := login(t, r, "siswa.test", testPassword)
	res := sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", session.AccessToken)

	var me meData
	decodeData(t, res, &me)
	if me.Access == nil || me.Access.Subscription == nil {
		t.Fatalf("seharusnya ada paket aktif. body: %s", res.Body)
	}
	if me.Access.Subscription.Channel != "online" {
		t.Errorf("channel = %q, seharusnya online", me.Access.Subscription.Channel)
	}

	features := me.Access.Features
	if !features["module"] || !features["daily_quiz"] {
		t.Errorf("module & daily_quiz seharusnya terbuka dari paket: %v", features)
	}
	if !features["rumath"] {
		t.Errorf("rumath seharusnya terbuka dari akses khusus admin: %v", features)
	}
	if features["tryout"] {
		t.Errorf("tryout seharusnya tertutup: %v", features)
	}
}

func TestMe_SiswaPaketExpiredKemarin(t *testing.T) {
	r, db := setupRouter(t)
	studentID := createUser(t, db, loginmodel.RoleStudent, "siswa.test", loginmodel.UserStatusActive)
	createPlanAndSubscription(t, db, studentID, "online", -31, -1) // berakhir kemarin

	session := login(t, r, "siswa.test", testPassword)
	res := sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", session.AccessToken)

	var me meData
	decodeData(t, res, &me)
	if me.Access == nil {
		t.Fatalf("siswa seharusnya punya data akses. body: %s", res.Body)
	}
	if me.Access.Subscription != nil {
		t.Errorf("paket yang berakhir kemarin tidak boleh dianggap aktif. body: %s", res.Body)
	}
	for feature, enabled := range me.Access.Features {
		if enabled {
			t.Errorf("fitur %s seharusnya tertutup karena tidak ada paket aktif", feature)
		}
	}
}

// masterData adalah bentuk JSON jenjang/kelas di data user.
type masterData struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	Code *string `json:"code"`
	Name string  `json:"name"`
}

func TestLogin_SiswaMendapatJenjangDanKelas(t *testing.T) {
	r, db := setupRouter(t)
	studentID := createUser(t, db, loginmodel.RoleStudent, "siswa.test", loginmodel.UserStatusActive)

	// Jenjang & kelas diambil dari data awal migration (master_data_items).
	err := db.Exec(`
		UPDATE users
		SET education_level_id = (SELECT id FROM master_data_items WHERE type = 'education_level' AND code = 'SMP'),
		    grade_id           = (SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_7')
		WHERE id = ?`, studentID).Error
	if err != nil {
		t.Fatal(err)
	}

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login",
		`{"username":"siswa.test","password":"`+testPassword+`"}`, "")
	if res.Code != http.StatusOK {
		t.Fatalf("login gagal: status %d, body: %s", res.Code, res.Body)
	}

	var session struct {
		User struct {
			EducationLevel *masterData `json:"education_level"`
			Grade          *masterData `json:"grade"`
		} `json:"user"`
	}
	decodeData(t, res, &session)

	level := session.User.EducationLevel
	if level == nil || level.Type != "education_level" || level.Code == nil || *level.Code != "SMP" || level.Name != "SMP" || level.ID == "" {
		t.Errorf("education_level salah: %+v. body: %s", level, res.Body)
	}
	grade := session.User.Grade
	if grade == nil || grade.Type != "grade" || grade.Code == nil || *grade.Code != "GRADE_7" || grade.Name != "Kelas 7" || grade.ID == "" {
		t.Errorf("grade salah: %+v. body: %s", grade, res.Body)
	}
}

func TestLogin_AdminTanpaJenjangDanKelas(t *testing.T) {
	r, db := setupRouter(t)
	createUser(t, db, loginmodel.RoleAdmin, "admin.test", loginmodel.UserStatusActive)

	res := sendRequest(r, http.MethodPost, "/api/v1/auth/login",
		`{"username":"admin.test","password":"`+testPassword+`"}`, "")
	if res.Code != http.StatusOK {
		t.Fatalf("login gagal: status %d, body: %s", res.Code, res.Body)
	}

	// LEFT JOIN: admin tetap bisa login, jenjang & kelasnya null.
	if !strings.Contains(res.Body.String(), `"education_level":null`) || !strings.Contains(res.Body.String(), `"grade":null`) {
		t.Errorf("education_level dan grade seharusnya null. body: %s", res.Body)
	}
}
