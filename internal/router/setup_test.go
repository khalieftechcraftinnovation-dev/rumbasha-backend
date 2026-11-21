// Helper bersama untuk semua test endpoint di folder ini.
//
// Test di sini menguji dari ujung ke ujung: route → middleware → handler →
// service → repository → PostgreSQL. Memakai database test (rumbasha_test).
// Jalankan dengan: make test   (atau: go test -p 1 ./...)
//
// Catatan Go: file test selalu berakhiran _test.go, dan fungsi test
// selalu diawali "Test" dengan parameter (t *testing.T).
package router

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
	subjectrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subject"
	subscriptionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subscription"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
	subjectservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subject"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/testutil"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

const (
	testJWTSecret = "secret-khusus-test-minimal-32-karakter!!"
	testPassword  = "password123"
)

// testPasswordHash dihitung sekali saja, karena bcrypt sengaja dibuat lambat.
var testPasswordHash = sync.OnceValue(func() string {
	hash, err := loginservice.HashPassword(testPassword)
	if err != nil {
		panic(err)
	}
	return hash
})

// setupRouter menyusun semua layer persis seperti di main.go, tapi memakai database test.
// db ikut dikembalikan supaya test bisa menyiapkan data langsung dengan SQL.
func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB(t)

	loginService := loginservice.NewLoginService(
		loginrepo.NewUserRepository(db),
		loginrepo.NewRefreshTokenRepository(db),
		testJWTSecret,
		15*time.Minute,
	)
	subscriptionService := subscriptionservice.NewSubscriptionService(subscriptionrepo.NewSubscriptionRepository(db))
	subjectService := subjectservice.NewSubjectService(subjectrepo.NewSubjectRepository(db))

	r := New(Handlers{
		Login:   loginhandler.NewLoginHandler(loginService, subscriptionService),
		Subject: subjecthandler.NewSubjectHandler(subjectService),
	}, testJWTSecret)

	return r, db
}

// sendRequest mengirim request ke router tanpa perlu menjalankan server sungguhan.
// accessToken boleh kosong untuk request tanpa login.
func sendRequest(r *gin.Engine, method, url, body, accessToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	return recorder
}

// tokenFor membuat access token untuk role tertentu tanpa perlu login.
// Cocok untuk test yang tidak membutuhkan user sungguhan di database.
func tokenFor(t *testing.T, role string) string {
	t.Helper()
	accessToken, err := token.NewAccessToken(uuid.NewString(), role, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return accessToken
}

// createUser menyimpan user ke database test dengan password testPassword, lalu mengembalikan ID-nya.
func createUser(t *testing.T, db *gorm.DB, role, username, status string) string {
	t.Helper()
	var id string
	err := db.Raw(`
		INSERT INTO users (role, username, password_hash, real_name, status)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id`, role, username, testPasswordHash(), "User "+username, status).
		Scan(&id).Error
	if err != nil {
		t.Fatalf("gagal membuat user %s: %v", username, err)
	}

	if role == "student" {
		if err := db.Exec(`INSERT INTO students (user_id) VALUES (?)`, id).Error; err != nil {
			t.Fatalf("gagal membuat data siswa %s: %v", username, err)
		}
	}
	return id
}

// decodeData membaca field "data" dari response JSON ke dalam target.
func decodeData(t *testing.T, res *httptest.ResponseRecorder, target any) {
	t.Helper()
	body := struct {
		Data json.RawMessage `json:"data"`
	}{}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response bukan JSON yang valid: %v. body: %s", err, res.Body)
	}
	if err := json.Unmarshal(body.Data, target); err != nil {
		t.Fatalf("gagal membaca data: %v. body: %s", err, res.Body)
	}
}
