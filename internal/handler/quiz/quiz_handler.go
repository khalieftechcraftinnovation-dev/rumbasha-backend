// Package quiz (handler) menangani endpoint quiz.
//
// Di package lain, import dengan alias quizhandler:
//
//	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
package quiz

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
)

// QuizHandler menangani endpoint /quizzes.
type QuizHandler struct {
	quizService *quizservice.QuizService
}

// NewQuizHandler membuat QuizHandler. Dipanggil sekali di main.go.
func NewQuizHandler(quizService *quizservice.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

// Create menangani POST /api/v1/quizzes: membuat 1 quiz (Uji Kom / Tryout) dengan soal yang dipilih.
// Quiz baru berstatus draft. data = detail quiz (letak dengan nama + daftar soal urut nomor).
func (h *QuizHandler) Create(c *gin.Context) {
	var req quizdto.CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	detail, err := h.quizService.Create(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Quiz berhasil dibuat", quizdto.ToQuizDetailResponse(detail))
}

// ListBundles menangani GET /api/v1/quiz-bundles: daftar bundle (quiz yang sudah dibuat) untuk
// dipilih saat membuat quiz baru. Isi soal 1 bundle: GET /api/v1/questions?quiz_id=<id>.
func (h *QuizHandler) ListBundles(c *gin.Context) {
	var query quizdto.ListBundlesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	bundles, filter, total, err := h.quizService.ListBundles(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Data bundle soal", quizdto.ToBundleResponses(bundles), response.Meta{
		Page:    filter.Page,
		PerPage: filter.PerPage,
		Total:   total,
	})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *QuizHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *quizmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, quizmodel.ErrQuizNotFound):
		response.Error(c, http.StatusNotFound, "Quiz tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
