// Package question (handler) menangani endpoint bank soal.
//
// Di package lain, import dengan alias questionhandler:
//
//	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/question"
package question

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/question"
)

// QuestionHandler menangani endpoint /questions.
type QuestionHandler struct {
	questionService *questionservice.QuestionService
}

// NewQuestionHandler membuat QuestionHandler. Dipanggil sekali di main.go.
func NewQuestionHandler(questionService *questionservice.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: questionService}
}

// Create menangani POST /api/v1/questions. Hanya admin & super admin (diatur di router).
func (h *QuestionHandler) Create(c *gin.Context) {
	var req questiondto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	question, err := h.questionService.Create(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Soal berhasil dibuat", questiondto.ToQuestionResponse(question))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *QuestionHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	// Kesalahan isi soal → 422 beserta daftar field yang salah.
	var validationErr *questionmodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
		return
	}

	switch {
	case errors.Is(err, questionmodel.ErrQuestionNotFound):
		response.Error(c, http.StatusNotFound, "Soal tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
