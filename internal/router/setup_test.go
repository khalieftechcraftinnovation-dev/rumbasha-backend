// Helper bersama untuk semua test endpoint di folder ini.
//
// Test di sini menguji dari ujung ke ujung: route → middleware → handler →
// service → repository → PostgreSQL. Memakai database DB_NAME (rumbasha); tabel data dikosongkan setiap test.
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

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	taskhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/admin/task"
	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/login"
	passwordresethandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/passwordreset"
	coinhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/coin"
	subscriptionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/subscription"
	learningmodulehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/learning/learningmodule"
	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/dropdown"
	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/image"
	masterdatahandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/masterdata"
	announcementhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/notification/announcement"
	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/dailyquiz"
	leaderboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/leaderboard"
	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/question"
	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/quiz"
	trialquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/trialquiz"
	dailyquizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/dailyquizreport"
	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/logreport"
	quizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/quizreport"
	landingpagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/site/landingpage"
	profilehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user/profile"
	userhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	taskrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/admin/task"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/login"
	passwordresetrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/passwordreset"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/coin"
	subscriptionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/subscription"
	learningmodulerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/learning/learningmodule"
	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/dropdown"
	imagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/image"
	masterdatarepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/masterdata"
	announcementrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/notification/announcement"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/dailyquiz"
	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/leaderboard"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/question"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/quiz"
	trialquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/trialquiz"
	dailyquizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/dailyquizreport"
	quizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/quizreport"
	landingpagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/site/landingpage"
	profilerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user/profile"
	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user/user"
	taskservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/admin/task"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/login"
	passwordresetservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/passwordreset"
	coinservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/coin"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"
	learningmoduleservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/learning/learningmodule"
	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/dropdown"
	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/image"
	masterdataservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/masterdata"
	announcementservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/notification/announcement"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/dailyquiz"
	leaderboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/leaderboard"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/question"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/quiz"
	trialquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/trialquiz"
	dailyquizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/dailyquizreport"
	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/logreport"
	quizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/quizreport"
	landingpageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/site/landingpage"
	profileservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user/profile"
	userservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/testutil"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

const (
	testJWTSecret = "secret-khusus-test-minimal-32-karakter!!"
	testPassword  = "password123"
	testAppURL    = "http://api.test"
)

// testPasswordHash dihitung sekali saja, karena bcrypt sengaja dibuat lambat.
var testPasswordHash = sync.OnceValue(func() string {
	hash, err := loginservice.HashPassword(testPassword)
	if err != nil {
		panic(err)
	}
	return hash
})

// fakeMailSender pura-pura mengirim email: pesan yang "dikirim" disimpan di sent.
type fakeMailSender struct {
	sent []mailer.Message
}

func (f *fakeMailSender) IsConfigured() bool { return true }

func (f *fakeMailSender) Send(msg mailer.Message) error {
	f.sent = append(f.sent, msg)
	return nil
}

// testApp berisi semua yang mungkin dibutuhkan test endpoint.
type testApp struct {
	Router     *gin.Engine
	DB         *gorm.DB
	APILog     *apilog.Writer  // writer log API (panggil Close() sebelum membaca file log)
	LogDir     string          // folder log API sementara
	Mail       *fakeMailSender // email yang "terkirim" selama test
	StorageDir string          // folder file upload sementara (gambar)
}

// setupRouter menyusun semua layer persis seperti di main.go, tapi memakai database DB_NAME (tabel data dikosongkan).
// db ikut dikembalikan supaya test bisa menyiapkan data langsung dengan SQL.
func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	app := setupApp(t)
	return app.Router, app.DB
}

// setupApp sama seperti setupRouter, tapi mengembalikan semuanya (folder log, email palsu, dsb.).
// Log API ditulis ke folder sementara yang otomatis dihapus setelah test selesai.
func setupApp(t *testing.T) testApp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB(t)

	logDir := t.TempDir()
	apiLog, err := apilog.NewWriter(logDir, 15)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiLog.Close)

	loginService := loginservice.NewLoginService(
		loginrepo.NewUserRepository(db),
		loginrepo.NewRefreshTokenRepository(db),
		testJWTSecret,
		15*time.Minute,
	)
	subscriptionService := subscriptionservice.NewSubscriptionService(subscriptionrepo.NewSubscriptionRepository(db))
	mail := &fakeMailSender{}
	logReportService := logreportservice.NewLogReportService(mail, logDir, "dev@example.com", "test")

	storageDir := t.TempDir() // file upload test otomatis dihapus setelah test selesai
	store := storage.NewLocal(storageDir, testAppURL)

	r := New(Handlers{
		Announcement:    announcementhandler.NewAnnouncementHandler(announcementservice.NewAnnouncementService(announcementrepo.NewAnnouncementRepository(db))),
		Task:            taskhandler.NewTaskHandler(taskservice.NewTaskService(taskrepo.NewTaskRepository(db))),
		Login:           loginhandler.NewLoginHandler(loginService, subscriptionService),
		LogReport:       logreporthandler.NewLogReportHandler(logReportService, loginService),
		Question:        questionhandler.NewQuestionHandler(questionservice.NewQuestionService(questionrepo.NewQuestionRepository(db))),
		Image:           imagehandler.NewImageHandler(imageservice.NewImageService(imagerepo.NewImageRepository(db), store)),
		Dropdown:        dropdownhandler.NewDropdownHandler(dropdownservice.NewDropdownService(dropdownrepo.NewDropdownRepository(db))),
		Quiz:            quizhandler.NewQuizHandler(quizservice.NewQuizService(quizrepo.NewQuizRepository(db), questionrepo.NewQuestionRepository(db))),
		DailyQuiz:       dailyquizhandler.NewDailyQuizHandler(dailyquizservice.NewDailyQuizService(dailyquizrepo.NewDailyQuizRepository(db), questionrepo.NewQuestionRepository(db))),
		Leaderboard:     leaderboardhandler.NewLeaderboardHandler(leaderboardservice.NewLeaderboardService(leaderboardrepo.NewLeaderboardRepository(db))),
		Coin:            coinhandler.NewCoinHandler(coinservice.NewCoinService(coinrepo.NewCoinRepository(db))),
		User:            userhandler.NewUserHandler(userservice.NewUserService(userrepo.NewUserRepository(db), subscriptionService)),
		QuizReport:      quizreporthandler.NewQuizReportHandler(quizreportservice.NewQuizReportService(quizreportrepo.NewQuizReportRepository(db))),
		DailyQuizReport: dailyquizreporthandler.NewDailyQuizReportHandler(dailyquizreportservice.NewDailyQuizReportService(dailyquizreportrepo.NewDailyQuizReportRepository(db))),
		LearningModule:  learningmodulehandler.NewLearningModuleHandler(learningmoduleservice.NewLearningModuleService(learningmodulerepo.NewLearningModuleRepository(db), store, subscriptionService)),
		Subscription:    subscriptionhandler.NewSubscriptionHandler(subscriptionService),
		MasterData:      masterdatahandler.NewMasterDataHandler(masterdataservice.NewMasterDataService(masterdatarepo.NewMasterDataRepository(db))),
		PasswordReset:   passwordresethandler.NewPasswordResetHandler(passwordresetservice.NewPasswordResetService(passwordresetrepo.NewPasswordResetRepository(db))),
		Profile:         profilehandler.NewProfileHandler(profileservice.NewProfileService(profilerepo.NewProfileRepository(db), subscriptionService, store)),
		LandingPage:     landingpagehandler.NewLandingPageHandler(landingpageservice.NewLandingPageService(landingpagerepo.NewLandingPageRepository(db))),
		TrialQuiz: trialquizhandler.NewTrialQuizHandler(trialquizservice.NewTrialQuizService(
			trialquizrepo.NewTrialQuizRepository(db), questionservice.NewQuestionService(questionrepo.NewQuestionRepository(db)))),
	}, testJWTSecret, apiLog, storageDir)

	return testApp{Router: r, DB: db, APILog: apiLog, LogDir: logDir, Mail: mail, StorageDir: storageDir}
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

// createUser menyimpan user ke database dengan password testPassword, lalu mengembalikan ID-nya.
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
