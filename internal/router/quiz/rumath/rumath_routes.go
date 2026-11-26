// Package rumath registers student RUMATH endpoints.
package rumath

import (
	"github.com/gin-gonic/gin"

	rumathhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/rumath"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func Register(private *gin.RouterGroup, h *rumathhandler.RumathHandler) {
	private.POST("/rumath/sessions", middleware.RequireRole(loginmodel.RoleStudent), h.GenerateSession)
}
