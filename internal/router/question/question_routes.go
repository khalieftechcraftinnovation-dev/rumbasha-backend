// Package question (router) berisi daftar URL untuk fitur bank soal.
//
// Di package lain, import dengan alias questionrouter:
//
//	questionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/question"
package question

import (
	"github.com/gin-gonic/gin"

	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /questions. Wajib login, hanya admin & super admin.
func Register(private *gin.RouterGroup, h *questionhandler.QuestionHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/questions", adminOnly, h.Create)
}
