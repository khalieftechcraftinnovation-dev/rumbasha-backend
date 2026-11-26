// Package dashboard registers Admin dashboard endpoints.
package dashboard

import (
	"github.com/gin-gonic/gin"

	dashboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/admin/dashboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func Register(private *gin.RouterGroup, h *dashboardhandler.DashboardHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	private.GET("/admin/dashboard/summary", adminOnly, h.Summary)
	private.GET("/admin/dashboard/daily-quiz-trend", adminOnly, h.DailyQuizTrend)
	private.GET("/admin/dashboard/search", adminOnly, h.Search)
}
