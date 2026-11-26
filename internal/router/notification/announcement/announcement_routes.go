// Package announcement registers announcement management and student notification routes.
package announcement

import (
	"github.com/gin-gonic/gin"

	announcementhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/notification/announcement"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func Register(private *gin.RouterGroup, h *announcementhandler.AnnouncementHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)

	private.GET("/announcements", adminOnly, h.ListAdmin)
	private.GET("/announcements/:id", adminOnly, h.GetAdmin)
	private.POST("/announcements", adminOnly, h.Create)
	private.PUT("/announcements/:id", adminOnly, h.Update)
	private.DELETE("/announcements/:id", adminOnly, h.Delete)

	private.GET("/student/notifications", studentOnly, h.ListForStudent)
	private.GET("/student/notifications/unread-count", studentOnly, h.UnreadCount)
	private.POST("/student/notifications/read-all", studentOnly, h.MarkAllRead)
	private.POST("/student/notifications/:id/read", studentOnly, h.MarkRead)
}
