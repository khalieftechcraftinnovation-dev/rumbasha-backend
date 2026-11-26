// Package router adalah layer ROUTE.
//
// File ini adalah pintu utamanya: membuat Gin, memasang middleware global,
// lalu memanggil route setiap fitur. Daftar URL per fitur ada di subfolder:
//
//	router/auth/login/...               → /auth/...
//	router/quiz/quiz/...                → /quizzes/...
//	router/quiz/question/...            → /questions/...
//	router/master/image/...             → /images/...
//	router/site/landingpage/...         → /landing-pages/..., /public/landing-pages/...
//	router/quiz/trialquiz/...            → /trial-quiz/..., /public/trial-quiz/...
//	router/notification/announcement/... → /announcements/..., /student/notifications/...
//	router/admin/task/...               → /tasks/...
package router

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

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
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	taskrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/admin/task"
	loginrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/auth/login"
	passwordresetrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/auth/passwordreset"
	coinrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/billing/coin"
	subscriptionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/billing/subscription"
	learningmodulerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/learning/learningmodule"
	dropdownrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/master/dropdown"
	imagerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/master/image"
	masterdatarouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/master/masterdata"
	announcementrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/notification/announcement"
	dailyquizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/dailyquiz"
	leaderboardrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/leaderboard"
	questionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/question"
	quizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/quiz"
	trialquizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/trialquiz"
	dailyquizreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/report/dailyquizreport"
	logreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/report/logreport"
	quizreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/report/quizreport"
	landingpagerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/site/landingpage"
	profilerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/user/profile"
	userrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/user/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada fitur baru, tambahkan field-nya di sini.
type Handlers struct {
	Announcement    *announcementhandler.AnnouncementHandler
	Task            *taskhandler.TaskHandler
	Login           *loginhandler.LoginHandler
	LogReport       *logreporthandler.LogReportHandler
	Question        *questionhandler.QuestionHandler
	Image           *imagehandler.ImageHandler
	Dropdown        *dropdownhandler.DropdownHandler
	Quiz            *quizhandler.QuizHandler
	DailyQuiz       *dailyquizhandler.DailyQuizHandler
	Leaderboard     *leaderboardhandler.LeaderboardHandler
	Coin            *coinhandler.CoinHandler
	User            *userhandler.UserHandler
	QuizReport      *quizreporthandler.QuizReportHandler
	DailyQuizReport *dailyquizreporthandler.DailyQuizReportHandler
	LearningModule  *learningmodulehandler.LearningModuleHandler
	Subscription    *subscriptionhandler.SubscriptionHandler
	MasterData      *masterdatahandler.MasterDataHandler
	PasswordReset   *passwordresethandler.PasswordResetHandler
	Profile         *profilehandler.ProfileHandler
	LandingPage     *landingpagehandler.LandingPageHandler
	TrialQuiz       *trialquizhandler.TrialQuizHandler
}

// New membuat router Gin lengkap dengan semua route.
// jwtSecret dipakai middleware untuk memeriksa access token.
// apiLog dipakai untuk mencatat setiap request ke file log harian.
// storageDir adalah folder file upload; gambar di storageDir/images disajikan publik di /storage/images.
func New(h Handlers, jwtSecret string, apiLog *apilog.Writer, storageDir string) *gin.Engine {
	// Supaya pesan error validasi memakai nama field JSON ("per_page"),
	// bukan nama field struct Go ("PerPage").
	response.UseJSONFieldNames()

	// gin.New() = Gin kosong tanpa middleware. Kita pasang middleware sendiri, urutannya penting:
	// 1. APILog: mencatat request ke file logs/api-YYYY-MM-DD.txt. Dipasang paling awal
	//    supaya semua request tercatat, termasuk yang panic atau ditolak middleware lain.
	// 2. Logger: mencetak ringkasan setiap request ke terminal.
	// 3. CustomRecovery: mencegah server mati kalau terjadi panic, lalu membalas
	//    500 dengan format response standar (Recovery bawaan Gin membalas body kosong).
	//    Detail panic tetap dicetak di terminal oleh Gin.
	r := gin.New()
	r.Use(middleware.APILog(apiLog))
	r.Use(gin.Logger())
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		_ = c.Error(fmt.Errorf("panic: %v", recovered)) // dicatat di log API
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}))

	// URL yang tidak terdaftar juga dibalas dengan format standar
	// (bawaannya Gin membalas teks biasa "404 page not found").
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "Endpoint tidak ditemukan")
	})

	// Jangan percaya header X-Forwarded-For dari siapa pun (menghilangkan warning Gin).
	// Atur ulang nanti kalau server sudah di belakang reverse proxy (nginx).
	_ = r.SetTrustedProxies(nil)

	// File gambar (Image Library) bisa dibuka publik tanpa login, contoh:
	// GET /storage/images/question/2026/11/<uuid>.png. Daftar isi folder tidak ditampilkan.
	r.Static(storage.PublicPrefix+"/images", filepath.Join(storageDir, "images"))

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "OK", nil)
	})

	// public  = boleh diakses tanpa login
	// private = wajib login (semua route di dalamnya melewati RequireAuth lebih dulu)
	public := r.Group("/api/v1")
	private := r.Group("/api/v1", middleware.RequireAuth(jwtSecret))

	// Daftarkan route setiap fitur. Tambahkan baris baru setiap ada fitur baru.
	announcementrouter.Register(private, h.Announcement)
	taskrouter.Register(private, h.Task)
	loginrouter.Register(public, private, h.Login)
	logreportrouter.Register(private, h.LogReport)
	questionrouter.Register(private, h.Question)
	imagerouter.Register(private, h.Image)
	dropdownrouter.Register(private, h.Dropdown)
	quizrouter.Register(private, h.Quiz)
	dailyquizrouter.Register(private, h.DailyQuiz)
	leaderboardrouter.Register(private, h.Leaderboard)
	coinrouter.Register(private, h.Coin)
	userrouter.Register(private, h.User)
	quizreportrouter.Register(private, h.QuizReport)
	dailyquizreportrouter.Register(private, h.DailyQuizReport)
	learningmodulerouter.Register(private, h.LearningModule)
	subscriptionrouter.Register(private, h.Subscription)
	masterdatarouter.Register(private, h.MasterData)
	passwordresetrouter.Register(public, private, h.PasswordReset)
	profilerouter.Register(private, h.Profile)
	landingpagerouter.Register(public, private, h.LandingPage)
	trialquizrouter.Register(public, private, h.TrialQuiz)

	return r
}
