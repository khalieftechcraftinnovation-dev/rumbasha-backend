// Package subscription (router) berisi daftar URL paket akses dan akses siswa.
//
// Di package lain, import dengan alias subscriptionrouter:
//
//	subscriptionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/billing/subscription"
package subscription

import (
	"github.com/gin-gonic/gin"

	subscriptionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route paket akses. Semua wajib login.
func Register(private *gin.RouterGroup, h *subscriptionhandler.SubscriptionHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	adminOrStudent := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleStudent)

	// Paket akses: siswa melihat paket aktif untuk jenjangnya (Paket & Coin), admin mengelola (CRUD).
	private.GET("/subscription-plans", adminOrStudent, h.ListPlans)
	private.GET("/subscription-plans/:id", adminOnly, h.GetPlan)
	private.POST("/subscription-plans", adminOnly, h.CreatePlan)
	private.PUT("/subscription-plans/:id", adminOnly, h.UpdatePlan)
	private.DELETE("/subscription-plans", adminOnly, h.DeletePlans) // body {"ids": [...]}, soft delete

	// Akses siswa (menu User / Coin & Akses).
	private.GET("/students/:id/access", adminOnly, h.StudentAccess)        // akses hari ini + riwayat paket
	private.POST("/student-subscriptions", adminOnly, h.Assign)            // beri paket ke 1-100 siswa
	private.POST("/student-subscriptions/:id/cancel", adminOnly, h.Cancel) // batalkan 1 paket siswa
	private.PUT("/student-access-overrides", adminOnly, h.SetOverrides)    // akses khusus 1-100 siswa
}
