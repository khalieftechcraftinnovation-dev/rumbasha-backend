// Package router adalah layer ROUTE: daftar semua URL dan handler-nya.
// Semua route ada di file ini (seperti routes/api.php di Laravel), jadi
// kalau mau tahu sebuah URL ditangani oleh fungsi apa, cukup cari di sini.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// Handlers berisi semua handler aplikasi.
// Setiap ada domain baru, tambahkan field-nya di sini.
type Handlers struct {
	Subject *handler.SubjectHandler
}

// New membuat router Gin lengkap dengan semua route.
func New(h Handlers) *gin.Engine {
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

	api := r.Group("/api/v1")

	// Subject (mata pelajaran)
	api.GET("/subjects", h.Subject.List)
	api.POST("/subjects", h.Subject.Create)
	api.GET("/subjects/:id", h.Subject.GetByID)

	return r
}
