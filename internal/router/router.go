// Package router adalah layer ROUTE.
//
// File ini adalah pintu utamanya: membuat Gin, memasang middleware global,
// lalu memanggil route setiap fitur. Daftar URL per fitur ada di subfolder:
//
//	router/login/login_routes.go      → /auth/...
//	router/subject/subject_routes.go  → /subjects/...
package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/logreport"
	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	loginrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/login"
	logreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/logreport"
	// subjectrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/subject"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada fitur baru, tambahkan field-nya di sini.
type Handlers struct {
	Login     *loginhandler.LoginHandler
	Subject   *subjecthandler.SubjectHandler
	LogReport *logreporthandler.LogReportHandler
}

// New membuat router Gin lengkap dengan semua route.
// jwtSecret dipakai middleware untuk memeriksa access token.
// apiLog dipakai untuk mencatat setiap request ke file log harian.
func New(h Handlers, jwtSecret string, apiLog *apilog.Writer) *gin.Engine {
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
	// subjectrouter.Register(private, h.Subject)

	return r
}
