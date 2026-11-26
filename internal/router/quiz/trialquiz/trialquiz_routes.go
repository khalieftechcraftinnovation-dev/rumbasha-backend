// Package trialquiz (router) berisi daftar URL fitur Quiz Trial.
//
// Di package lain, import dengan alias trialquizrouter:
//
//	trialquizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quiz/trialquiz"
package trialquiz

import (
	"time"

	"github.com/gin-gonic/gin"

	trialquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/trialquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Batas submit trial per IP (endpoint publik yang menulis ke database).
const (
	submitLimit  = 10
	submitWindow = time.Minute
)

// Register mendaftarkan route Quiz Trial.
//   - public : /public/trial-quiz/... (tombol "Coba quiz gratis", tanpa login)
//   - private: /trial-quiz/... (tab Quiz Trial di Landing Page + report, hanya admin & super admin)
func Register(public, private *gin.RouterGroup, h *trialquizhandler.TrialQuizHandler) {
	public.GET("/public/trial-quiz/grades", h.Grades)
	public.GET("/public/trial-quiz/questions", h.Questions)
	public.POST("/public/trial-quiz/submit", middleware.RateLimit(submitLimit, submitWindow), h.Submit)

	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	private.GET("/trial-quiz/questions", adminOnly, h.List)
	private.POST("/trial-quiz/questions", adminOnly, h.Create)
	private.GET("/trial-quiz/questions/:id", adminOnly, h.GetDetail)
	private.PUT("/trial-quiz/questions/:id", adminOnly, h.Update)
	private.DELETE("/trial-quiz/questions/:id", adminOnly, h.Delete)
	private.GET("/trial-quiz/attempts", adminOnly, h.ListAttempts)
	private.GET("/trial-quiz/attempts/summary", adminOnly, h.Summary)
}
