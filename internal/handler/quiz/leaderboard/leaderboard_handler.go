// Package leaderboard (handler) menangani endpoint leaderboard.
//
// Di package lain, import dengan alias leaderboardhandler:
//
//	leaderboardhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/leaderboard"
package leaderboard

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	leaderboarddto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/leaderboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/leaderboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	leaderboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/leaderboard"
)

// LeaderboardHandler menangani endpoint /leaderboards.
type LeaderboardHandler struct {
	service *leaderboardservice.LeaderboardService
}

// NewLeaderboardHandler membuat LeaderboardHandler. Dipanggil sekali di main.go.
func NewLeaderboardHandler(service *leaderboardservice.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{service: service}
}

// DailyQuiz menangani GET /api/v1/leaderboards/daily-quiz: total point Daily Quiz 1 bulan (WIB).
// Query: month (YYYY-MM, default bulan ini), education_level_id, grade_id, page, per_page.
func (h *LeaderboardHandler) DailyQuiz(c *gin.Context) {
	var q leaderboarddto.DailyQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	month, filter, board, err := h.service.DailyQuiz(c.Request.Context(), middleware.GetUserID(c), middleware.GetRole(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Leaderboard Daily Quiz", leaderboarddto.ToDailyBoardResponse(month, filter, board),
		response.Meta{Page: filter.Page, PerPage: filter.PerPage, Total: board.Total})
}

// Quiz menangani GET /api/v1/leaderboards/quizzes/:id: leaderboard 1 quiz Uji Kom / Tryout.
func (h *LeaderboardHandler) Quiz(c *gin.Context) {
	var q leaderboarddto.QuizQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	quiz, board, page, perPage, err := h.service.Quiz(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), middleware.GetRole(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Leaderboard quiz", leaderboarddto.ToQuizBoardResponse(quiz, board),
		response.Meta{Page: page, PerPage: perPage, Total: board.Total})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *LeaderboardHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *leaderboardmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, leaderboardmodel.ErrQuizNotFound):
		response.Error(c, http.StatusNotFound, "Quiz tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
