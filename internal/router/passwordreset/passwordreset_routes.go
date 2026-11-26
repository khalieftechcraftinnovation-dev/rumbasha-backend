// Package passwordreset (router) berisi daftar URL "Lupa kata sandi" lewat admin.
//
// Di package lain, import dengan alias passwordresetrouter:
//
//	passwordresetrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/passwordreset"
package passwordreset

import (
	"github.com/gin-gonic/gin"

	passwordresethandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/passwordreset"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route lupa kata sandi:
//   - public  → form "Lupa kata sandi" di halaman login
//   - private → daftar & proses permintaan (admin & super admin)
func Register(public, private *gin.RouterGroup, h *passwordresethandler.PasswordResetHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	public.POST("/auth/forgot-password", h.Forgot)

	private.GET("/password-reset-requests", adminOnly, h.List)          // default status pending
	private.GET("/password-reset-requests/counts", adminOnly, h.Counts) // badge menu
	private.GET("/password-reset-requests/:id", adminOnly, h.Get)
	private.POST("/password-reset-requests/:id/resolve", adminOnly, h.Resolve) // set kata sandi baru
	private.POST("/password-reset-requests/:id/reject", adminOnly, h.Reject)
}
