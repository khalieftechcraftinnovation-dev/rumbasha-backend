// Package dailyquiz (router) berisi daftar URL untuk fitur Daily Quiz.
//
// Di package lain, import dengan alias dailyquizrouter:
//
//	dailyquizrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/dailyquiz"
package dailyquiz

import (
	"github.com/gin-gonic/gin"

	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/dailyquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /daily-quizzes. Wajib login, khusus siswa (kelas diambil dari akunnya).
func Register(private *gin.RouterGroup, h *dailyquizhandler.DailyQuizHandler) {
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)

	private.GET("/daily-quizzes/grade-options", studentOnly, h.GradeOptions) // dropdown kelas (sesuai jenjang/kelompok siswa)
	private.POST("/daily-quizzes", studentOnly, h.Generate)                  // generate Daily Quiz dari soal acak
	private.POST("/daily-quizzes/:id/open", studentOnly, h.Open)             // buka: jatah gratis / potong coin, soal + kunci
	private.POST("/daily-quizzes/:id/submit", studentOnly, h.Submit)         // kumpulkan jawaban (dinilai FE) → nilai + review
	private.GET("/daily-quizzes/:id/review", studentOnly, h.Review)          // hasil & review setelah submit
}
