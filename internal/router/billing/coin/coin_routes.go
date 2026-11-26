// Package coin (router) berisi daftar URL untuk fitur coin.
//
// Di package lain, import dengan alias coinrouter:
//
//	coinrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/billing/coin"
package coin

import (
	"github.com/gin-gonic/gin"

	coinhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/coin"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route coin. Semua wajib login.
func Register(private *gin.RouterGroup, h *coinhandler.CoinHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	adminOrStudent := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleStudent)
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)

	// Paket isi coin: siswa melihat paket aktif (tab "Isi coin"), admin mengelola (CRUD).
	private.GET("/coin-packages", adminOrStudent, h.ListPackages)
	private.GET("/coin-packages/:id", adminOnly, h.GetPackage)
	private.POST("/coin-packages", adminOnly, h.CreatePackage)
	private.PUT("/coin-packages/:id", adminOnly, h.UpdatePackage)
	private.DELETE("/coin-packages", adminOnly, h.DeletePackages) // body {"ids": [...]}, soft delete

	// Admin menambah coin banyak siswa sekaligus sesuai 1 paket (pembelian tunai di cabang).
	private.POST("/coin-topups", adminOnly, h.TopUp)

	// Riwayat coin: milik siswa sendiri, atau 1 siswa untuk admin.
	private.GET("/student/coin-transactions", studentOnly, h.MyHistory)
	private.GET("/students/:id/coin-transactions", adminOnly, h.StudentHistory)
}
