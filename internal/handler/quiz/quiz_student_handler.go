package quiz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// StudentUpcoming menangani GET /api/v1/student/quizzes/upcoming: bagian "Akan datang" (tab, cari, mapel).
func (h *QuizHandler) StudentUpcoming(c *gin.Context) {
	var q quizdto.StudentQuizQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, now, err := h.quizService.StudentUpcoming(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Uji Kom & Tryout akan datang", quizdto.ToUpcomingResponses(items, now),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// StudentHistory menangani GET /api/v1/student/quizzes/history: bagian "Sudah dikerjakan".
func (h *QuizHandler) StudentHistory(c *gin.Context) {
	var q quizdto.StudentQuizQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.quizService.StudentHistory(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Uji Kom & Tryout sudah dikerjakan", quizdto.ToDoneResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// StudentStats menangani GET /api/v1/student/quizzes/stats: rata-rata nilai, jumlah sesi, point,
// dan peringkat Tryout terakhir.
func (h *QuizHandler) StudentStats(c *gin.Context) {
	var q quizdto.StudentQuizQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	stats, err := h.quizService.StudentStats(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik Uji Kom & Tryout", quizdto.ToStudentStatsResponse(stats))
}
