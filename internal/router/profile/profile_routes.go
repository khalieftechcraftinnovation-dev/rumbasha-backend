// Package profile (router) berisi daftar URL halaman Profil siswa.
//
// Di package lain, import dengan alias profilerouter:
//
//	profilerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/profile"
package profile

import (
	"github.com/gin-gonic/gin"

	profilehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/profile"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /student/profile. Wajib login sebagai siswa (profil milik sendiri).
// Ubah kata sandi: POST /auth/change-password (fitur login).
func Register(private *gin.RouterGroup, h *profilehandler.ProfileHandler) {
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)

	private.GET("/student/profile", studentOnly, h.Get)                                // data diri + paket + statistik
	private.PUT("/student/profile", studentOnly, h.Update)                             // data diri
	private.PUT("/student/profile/parent-contact", studentOnly, h.UpdateParentContact) // no HP orang tua / wali
	private.POST("/student/profile/photo", studentOnly, h.UploadPhoto)                 // foto profil (multipart "photo")
	private.DELETE("/student/profile/photo", studentOnly, h.DeletePhoto)
}
