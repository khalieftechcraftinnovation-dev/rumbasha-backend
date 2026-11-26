// Package task registers Admin task endpoints.
package task

import (
	"github.com/gin-gonic/gin"

	taskhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/admin/task"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func Register(private *gin.RouterGroup, h *taskhandler.TaskHandler) {
	adminOrSuperAdmin := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	superAdminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin)
	adminOnly := middleware.RequireRole(loginmodel.RoleAdmin)

	private.GET("/tasks", adminOrSuperAdmin, h.List)
	private.GET("/tasks/:id", adminOrSuperAdmin, h.Get)
	private.POST("/tasks", superAdminOnly, h.Create)
	private.PATCH("/tasks/:id/status", adminOnly, h.UpdateStatus)
}
