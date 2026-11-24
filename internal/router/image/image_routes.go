// Package image (router) berisi daftar URL untuk fitur Image Library.
//
// Di package lain, import dengan alias imagerouter:
//
//	imagerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/image"
package image

import (
	"github.com/gin-gonic/gin"

	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/image"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /images. Wajib login, hanya admin & super admin.
// File gambarnya sendiri disajikan publik di /storage/images/... (lihat router.go).
func Register(private *gin.RouterGroup, h *imagehandler.ImageHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/images", adminOnly, h.Upload)   // upload 1-10 gambar (multipart)
	private.DELETE("/images", adminOnly, h.Delete) // hapus permanen 1-100 gambar, body {"ids": [...]}
}
