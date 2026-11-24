// Package quiz (router) berisi daftar URL untuk fitur quiz.
//
// Di package lain, import dengan alias quizrouter:
//
//	quizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz"
package quiz

import (
	"github.com/gin-gonic/gin"

	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /quizzes. Wajib login, hanya admin & super admin.
func Register(private *gin.RouterGroup, h *quizhandler.QuizHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/quizzes", adminOnly, h.Create)          // buat quiz Uji Kom / Tryout + soal yang dipilih
	private.GET("/quiz-bundles", adminOnly, h.ListBundles) // pilihan bundle (soal quiz lama) untuk quiz baru
}
