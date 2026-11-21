// Package router adalah layer ROUTE.
//
// File ini adalah pintu utamanya: membuat Gin, memasang middleware global,
// lalu memanggil route setiap fitur. Daftar URL per fitur ada di subfolder:
//
//	router/login/login_routes.go      → /auth/...
//	router/subject/subject_routes.go  → /subjects/...
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/login"
	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	loginrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/login"
	// subjectrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/subject"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada fitur baru, tambahkan field-nya di sini.
type Handlers struct {
	Login   *loginhandler.LoginHandler
	Subject *subjecthandler.SubjectHandler
}

// New membuat router Gin lengkap dengan semua route.
// jwtSecret dipakai middleware untuk memeriksa access token.
func New(h Handlers, jwtSecret string) *gin.Engine {
	// Supaya pesan error validasi memakai nama field JSON ("per_page"),
	// bukan nama field struct Go ("PerPage").
	response.UseJSONFieldNames()

	// gin.Default() sudah memasang 2 middleware bawaan:
	// - Logger: mencetak log setiap request ke terminal
	// - Recovery: mencegah server mati kalau terjadi panic
	r := gin.Default()

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
	// subjectrouter.Register(private, h.Subject)

	return r
}
