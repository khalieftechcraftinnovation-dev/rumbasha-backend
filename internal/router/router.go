// Package router adalah layer ROUTE.
//
// File ini adalah pintu utamanya: membuat Gin, memasang middleware global,
// lalu memanggil route setiap fitur. Daftar URL per fitur ada di subfolder:
//
//	router/login/login_routes.go      → /auth/...
//	router/subject/subject_routes.go  → /subjects/...
//	router/question/question_routes.go → /questions/...
//	router/image/image_routes.go       → /images/...
package router

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dailyquiz"
	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dropdown"
	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/image"
	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/logreport"
	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/question"
	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	dailyquizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/dailyquiz"
	dropdownrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/dropdown"
	imagerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/image"
	loginrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/login"
	logreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/logreport"
	questionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/question"
	quizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
	// subjectrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/subject"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada fitur baru, tambahkan field-nya di sini.
type Handlers struct {
	Login     *loginhandler.LoginHandler
	Subject   *subjecthandler.SubjectHandler
	LogReport *logreporthandler.LogReportHandler
	Question  *questionhandler.QuestionHandler
	Image     *imagehandler.ImageHandler
	Dropdown  *dropdownhandler.DropdownHandler
	Quiz      *quizhandler.QuizHandler
	DailyQuiz *dailyquizhandler.DailyQuizHandler
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
	loginrouter.Register(public, private, h.Login)
	logreportrouter.Register(private, h.LogReport)
	questionrouter.Register(private, h.Question)
	imagerouter.Register(private, h.Image)
	dropdownrouter.Register(private, h.Dropdown)
	quizrouter.Register(private, h.Quiz)
	dailyquizrouter.Register(private, h.DailyQuiz)
	// subjectrouter.Register(private, h.Subject)

	return r
}
