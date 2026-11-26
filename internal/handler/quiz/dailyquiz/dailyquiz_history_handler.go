package dailyquiz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/dailyquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// History menangani GET /api/v1/student/daily-quizzes: riwayat Daily Quiz siswa (materi, level,
// waktu, nilai, point), terbaru dulu.
func (h *DailyQuizHandler) History(c *gin.Context) {
	var q dailyquizdto.HistoryQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.History(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Riwayat Daily Quiz", dailyquizdto.ToHistoryResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// HistoryStats menangani GET /api/v1/student/daily-quizzes/stats: rata-rata nilai & jumlah quiz,
// point, dan peringkat (bulan ini + sepanjang waktu).
func (h *DailyQuizHandler) HistoryStats(c *gin.Context) {
	var q dailyquizdto.HistoryStatsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	stats, err := h.service.HistoryStats(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik Daily Quiz", dailyquizdto.ToHistoryStatsResponse(stats))
}
