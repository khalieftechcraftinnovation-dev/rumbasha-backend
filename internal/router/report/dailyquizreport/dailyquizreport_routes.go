// Package dailyquizreport (router) berisi daftar URL report Daily Quiz per siswa.
//
// Di package lain, import dengan alias dailyquizreportrouter:
//
//	dailyquizreportrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/report/dailyquizreport"
package dailyquizreport

import (
	"github.com/gin-gonic/gin"

	dailyquizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/dailyquizreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route /daily-quiz-reports. Wajib login: super admin, admin, tutor (hanya membaca).
func Register(private *gin.RouterGroup, h *dailyquizreporthandler.DailyQuizReportHandler) {
	staff := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleTutor)

	private.GET("/daily-quiz-reports", staff, h.List)                       // siswa urut peringkat + rekap
	private.GET("/daily-quiz-reports/students/:id", staff, h.StudentDetail) // 1 siswa: total + per mapel
}
