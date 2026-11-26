// Test halaman Profil siswa: lihat profil + ringkasan, ubah data diri, kontak orang tua, foto profil.
package router

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

type profileData struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Email             *string  `json:"email"`
	Phone             *string  `json:"phone"`
	ParentPhone       *string  `json:"parent_phone"`
	ParentPhoneMasked *string  `json:"parent_phone_masked"`
	DateOfBirth       *string  `json:"date_of_birth"`
	SchoolOrigin      *string  `json:"school_origin"`
	Grade             *listRef `json:"grade"`
	AvatarURL         *string  `json:"avatar_url"`
	EditableFields    []string `json:"editable_fields"`
}

// sendPhoto mengirim multipart field "photo" ke POST /student/profile/photo.
func sendPhoto(r *gin.Engine, token, fileName string, data []byte) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, _ := w.CreateFormFile("photo", fileName)
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/student/profile/photo", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestProfile_LihatDanRingkasan(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	citra, token := namedStudent(t, db, "Citra", "GRADE_5")
	other, _ := namedStudent(t, db, "Dewi", "GRADE_5")
	smp, _ := namedStudent(t, db, "Eko", "GRADE_7")
	db.Exec(`UPDATE users SET parent_phone = '0812-3456-2210', school_origin = 'SDN 1 Surabaya' WHERE id = ?`, citra)
	db.Exec(`UPDATE students SET coin_balance = 75 WHERE user_id = ?`, citra)
	plan := newPlan(t, db, "Profil Uji", 1)
	subscribe(t, db, citra, plan)

	// Leaderboard Daily Quiz bulan ini: Dewi 80 point, Citra 50 → Citra #2 dari 2 di Kelas 5
	// (Eko kelas lain tidak dihitung). Bulan lalu Citra 20 point (ikut total point).
	for _, e := range []struct {
		id, period string
		points     int
	}{{citra, "this", 50}, {other, "this", 80}, {smp, "this", 999}, {citra, "last", 20}} {
		periodExpr := "TO_CHAR(NOW() AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')"
		if e.period == "last" {
			periodExpr = "TO_CHAR((NOW() AT TIME ZONE 'Asia/Jakarta') - INTERVAL '1 month', 'YYYY-MM')"
		}
		err := db.Exec(`
			INSERT INTO leaderboard_entries (board, period, student_id, education_level_id, grade_id, points, duration_seconds, quiz_count, last_submitted_at)
			SELECT 'daily_quiz', `+periodExpr+`, u.id, u.education_level_id, u.grade_id, ?, 100, 1, NOW() FROM users u WHERE u.id = ?`,
			e.points, e.id).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	submittedDailyQuiz(t, db, citra, p, "easy", 5, 4, 80, 50, 120, "NOW()")

	res := sendRequest(r, http.MethodGet, "/api/v1/student/profile", "", token)
	if res.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", res.Code, res.Body)
	}
	var data struct {
		Profile profileData `json:"profile"`
		Access  struct {
			Plan     *listRef `json:"plan"`
			DaysLeft *int     `json:"days_left"`
		} `json:"access"`
		Coin  int `json:"coin_balance"`
		Stats struct {
			DailyRank *struct {
				Rank, Total int
			} `json:"daily_quiz_rank"`
			PointsThisMonth  int `json:"points_this_month"`
			TotalPoints      int `json:"total_points"`
			QuizzesThisMonth int `json:"quizzes_this_month"`
		} `json:"stats"`
	}
	decodeData(t, res, &data)
	pr := data.Profile
	if pr.Name != "Citra" || pr.Grade == nil || pr.Grade.Name != "Kelas 5" || pr.ParentPhoneMasked == nil || *pr.ParentPhoneMasked != "0812-xxxx-2210" ||
		*pr.SchoolOrigin != "SDN 1 Surabaya" || len(pr.EditableFields) != 6 || pr.AvatarURL != nil {
		t.Errorf("data diri: %+v", pr)
	}
	if data.Access.Plan == nil || data.Access.Plan.Name != "Profil Uji" || data.Access.DaysLeft == nil || data.Coin != 75 {
		t.Errorf("paket & coin: %+v coin %d", data.Access, data.Coin)
	}
	st := data.Stats
	if st.DailyRank == nil || st.DailyRank.Rank != 2 || st.DailyRank.Total != 2 || st.PointsThisMonth != 50 || st.TotalPoints != 70 || st.QuizzesThisMonth != 1 {
		t.Errorf("statistik: %+v rank %+v", st, st.DailyRank)
	}

	// Siswa yang belum ikut Daily Quiz bulan ini: peringkat null.
	_, newToken := namedStudent(t, db, "Fani", "GRADE_5")
	res = sendRequest(r, http.MethodGet, "/api/v1/student/profile", "", newToken)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"daily_quiz_rank":null`) {
		t.Errorf("profil tanpa peringkat: %d %s", res.Code, res.Body)
	}
	for _, role := range []string{loginmodel.RoleAdmin, loginmodel.RoleTutor} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/student/profile", "", tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("profil siswa oleh %s: %d", role, res.Code)
		}
	}
}

func TestProfile_UbahDataDiriKontakDanFoto(t *testing.T) {
	app := setupApp(t)
	r, db := app.Router, app.DB
	citra, token := namedStudent(t, db, "Citra", "GRADE_5")
	db.Exec(`UPDATE users SET school_origin = 'SDN 1 Surabaya' WHERE id = ?`, citra)

	// Validasi dikumpulkan sekaligus.
	res := sendRequest(r, http.MethodPut, "/api/v1/student/profile", `{"name":" ","email":"bukan-email"}`, token)
	if res.Code != http.StatusUnprocessableEntity {
		t.Errorf("email salah format: %d", res.Code)
	}
	res = sendRequest(r, http.MethodPut, "/api/v1/student/profile", `{"name":" ","phone":"08abc","date_of_birth":"2999-01-01"}`, token)
	f := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || f["name"] == "" || f["phone"] == "" || f["date_of_birth"] == "" {
		t.Errorf("validasi data diri: %d %v", res.Code, f)
	}

	// Ubah data diri; field yang diatur admin (asal sekolah) diabaikan.
	res = sendRequest(r, http.MethodPut, "/api/v1/student/profile",
		`{"name":" Citra Lestari ","email":"Citra@Example.com","phone":"+62 812-3456-7890","date_of_birth":"2015-03-02","school_origin":"Diubah"}`, token)
	var pr profileData
	decodeData(t, res, &pr)
	if res.Code != http.StatusOK || pr.Name != "Citra Lestari" || *pr.Email != "citra@example.com" || *pr.Phone != "+62 812-3456-7890" ||
		*pr.DateOfBirth != "2015-03-02" || *pr.SchoolOrigin != "SDN 1 Surabaya" {
		t.Errorf("ubah data diri: %d %+v", res.Code, pr)
	}
	// Ganti penuh: field yang tidak dikirim dikosongkan.
	decodeData(t, sendRequest(r, http.MethodPut, "/api/v1/student/profile", `{"name":"Citra Lestari"}`, token), &pr)
	if pr.Email != nil || pr.Phone != nil || pr.DateOfBirth != nil {
		t.Errorf("ganti penuh: %+v", pr)
	}

	// Kontak orang tua / wali.
	if res := sendRequest(r, http.MethodPut, "/api/v1/student/profile/parent-contact", `{"parent_phone":"abc"}`, token); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("no HP orang tua salah: %d", res.Code)
	}
	decodeData(t, sendRequest(r, http.MethodPut, "/api/v1/student/profile/parent-contact", `{"parent_phone":"0813-9999-1234"}`, token), &pr)
	if pr.ParentPhone == nil || *pr.ParentPhoneMasked != "0813-xxxx-1234" {
		t.Errorf("simpan kontak orang tua: %+v", pr)
	}
	decodeData(t, sendRequest(r, http.MethodPut, "/api/v1/student/profile/parent-contact", `{"parent_phone":null}`, token), &pr)
	if pr.ParentPhone != nil || pr.ParentPhoneMasked != nil {
		t.Errorf("hapus kontak orang tua: %+v", pr)
	}

	// Foto profil: upload, ganti (file lama terhapus), tolak bukan gambar / terlalu besar, hapus.
	fileOf := func(url string) string {
		return filepath.Join(app.StorageDir, filepath.FromSlash(strings.TrimPrefix(url[strings.Index(url, "/storage/")+len("/storage/"):], "/")))
	}
	res = sendPhoto(r, token, "foto.png", makeImage(t, "png", 20, 20))
	decodeData(t, res, &pr)
	if res.Code != http.StatusOK || pr.AvatarURL == nil || !strings.Contains(*pr.AvatarURL, "/storage/images/avatar/") {
		t.Fatalf("upload foto: %d %s", res.Code, res.Body)
	}
	first := fileOf(*pr.AvatarURL)
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("file foto tidak ada: %v", err)
	}
	res = sendPhoto(r, token, "foto2.jpg", makeImage(t, "jpeg", 20, 20))
	decodeData(t, res, &pr)
	second := fileOf(*pr.AvatarURL)
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Error("foto lama seharusnya terhapus setelah diganti")
	}
	if !strings.HasSuffix(second, ".jpg") {
		t.Errorf("foto kedua: %s", second)
	}
	for name, data := range map[string][]byte{
		"bukan.png": []byte("ini teks biasa, bukan gambar"),
		"anim.gif":  makeImage(t, "gif", 5, 5),
		"besar.png": bytes.Repeat([]byte{0}, 2<<20+10),
	} {
		res := sendPhoto(r, token, name, data)
		if f := errorFields(t, res.Body.String()); res.Code != http.StatusUnprocessableEntity || f["photo"] == "" {
			t.Errorf("foto %s: %d %s", name, res.Code, res.Body)
		}
	}
	// /auth/me ikut menampilkan foto.
	if res := sendRequest(r, http.MethodGet, "/api/v1/auth/me", "", token); !strings.Contains(res.Body.String(), `"avatar_url":"`) {
		t.Errorf("/auth/me tanpa avatar: %s", res.Body)
	}
	decodeData(t, sendRequest(r, http.MethodDelete, "/api/v1/student/profile/photo", "", token), &pr)
	if pr.AvatarURL != nil {
		t.Errorf("hapus foto: %+v", pr)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Error("file foto seharusnya terhapus")
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/student/profile/photo", "", token); res.Code != http.StatusOK {
		t.Errorf("hapus foto 2x: %d", res.Code)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/student/profile/photo", "", token); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("tanpa file: %d %s", res.Code, res.Body)
	}
}
