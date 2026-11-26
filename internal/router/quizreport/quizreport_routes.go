// Package quizreport (router) berisi daftar URL report quiz Uji Kom & Tryout.
//
// Di package lain, import dengan alias quizreportrouter:
//
//	quizreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/quizreport"
package quizreport

import (
	"github.com/gin-gonic/gin"

	quizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quizreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /quiz-reports. Wajib login: super admin, admin, tutor (hanya membaca).
func Register(private *gin.RouterGroup, h *quizreporthandler.QuizReportHandler) {
	staff := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleTutor)

	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.GET("/quiz-reports/overview", staff, h.Overview) // kartu ringkasan + grafik per tab (Daily Quiz / Uji Kom / Tryout)
	private.GET("/quiz-report-settings", staff, h.GetSettings)
	private.PUT("/quiz-report-settings", adminOnly, h.UpdateSettings) // ambang soal sering salah & rentang default

	private.GET("/quiz-reports", staff, h.List)                    // ringkasan per event
	private.GET("/quiz-reports/:id", staff, h.Detail)              // peserta urut peringkat
	private.GET("/quiz-reports/:id/questions", staff, h.Questions) // soal paling sering salah dulu
}
