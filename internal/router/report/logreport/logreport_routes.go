// Package logreport (router) berisi daftar URL untuk fitur kirim log ke email (laporan bug).
//
// Di package lain, import dengan alias logreportrouter:
//
//	logreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/report/logreport"
package logreport

import (
	"github.com/gin-gonic/gin"

	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route /log-reports. Wajib login, hanya admin & super admin.
func Register(private *gin.RouterGroup, h *logreporthandler.LogReportHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/log-reports", adminOnly, h.Send)
}
