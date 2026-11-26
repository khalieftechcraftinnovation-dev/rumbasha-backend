// Package login (router) berisi daftar URL untuk fitur login.
//
// Di package lain, import dengan alias loginrouter:
//
//	loginrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/auth/login"
package login

import (
	"github.com/gin-gonic/gin"

	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/login"
)

// Register mendaftarkan semua route /auth.
//   - public  → boleh tanpa login
//   - private → wajib login
func Register(public, private *gin.RouterGroup, h *loginhandler.LoginHandler) {
	public.POST("/auth/login", h.Login)
	public.POST("/auth/refresh", h.Refresh)
	public.POST("/auth/logout", h.Logout)

	private.GET("/auth/me", h.Me)
	private.POST("/auth/change-password", h.ChangePassword) // ubah kata sandi sendiri (semua role)
}
