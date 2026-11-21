// Package router adalah layer ROUTE: daftar semua URL dan handler-nya.
// Semua route ada di file ini (seperti routes/api.php di Laravel), jadi
// kalau mau tahu sebuah URL ditangani oleh fungsi apa, cukup cari di sini.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada domain baru, tambahkan field-nya di sini.
type Handlers struct {
	Auth    *handler.AuthHandler
	Subject *handler.SubjectHandler
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

	// Middleware yang sering dipakai, disiapkan sekali di sini.
	requireAuth := middleware.RequireAuth(jwtSecret)
	adminOnly := middleware.RequireRole(model.RoleSuperAdmin, model.RoleAdmin)

	api := r.Group("/api/v1")

	// ----- Publik (tanpa login) -----
	api.POST("/auth/login", h.Auth.Login)
	api.POST("/auth/refresh", h.Auth.Refresh)
	api.POST("/auth/logout", h.Auth.Logout)

	// ----- Wajib login -----
	// Semua route di dalam grup ini melewati requireAuth lebih dulu.
	private := api.Group("", requireAuth)

	private.GET("/auth/me", h.Auth.Me)

	// Subject (mata pelajaran): semua user boleh melihat, hanya admin yang boleh mengubah.
	private.GET("/subjects", h.Subject.List)
	private.GET("/subjects/:id", h.Subject.GetByID)
	private.POST("/subjects", adminOnly, h.Subject.Create)
	private.PUT("/subjects/:id", adminOnly, h.Subject.Update)
	private.DELETE("/subjects/:id", adminOnly, h.Subject.Delete)

	return r
}
