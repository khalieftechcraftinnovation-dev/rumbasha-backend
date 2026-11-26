// Package trialquiz (handler) menangani endpoint Quiz Trial (admin & publik).
//
// Di package lain, import dengan alias trialquizhandler:
//
//	trialquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/trialquiz"
package trialquiz

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	trialquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/trialquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	trialquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/trialquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
	trialquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/trialquiz"
)

// TrialQuizHandler menangani endpoint /trial-quiz dan /public/trial-quiz.
type TrialQuizHandler struct {
	service *trialquizservice.TrialQuizService
}

// NewTrialQuizHandler membuat TrialQuizHandler. Dipanggil sekali di main.go.
func NewTrialQuizHandler(service *trialquizservice.TrialQuizService) *TrialQuizHandler {
	return &TrialQuizHandler{service: service}
}

// ============================================================================
// Admin: kelola soal
// ============================================================================

// List menangani GET /api/v1/trial-quiz/questions?grade_group=...: soal trial + jumlah per golongan.
func (h *TrialQuizHandler) List(c *gin.Context) {
	var query trialquizdto.ListQuestionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	questions, counts, err := h.service.List(c.Request.Context(), query.GradeGroup)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Data soal quiz trial", trialquizdto.ToListQuestionsResponse(questions, counts))
}

// Create menangani POST /api/v1/trial-quiz/questions: tambah 1 soal trial.
func (h *TrialQuizHandler) Create(c *gin.Context) {
	var req trialquizdto.QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	q, err := h.service.Create(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Soal quiz trial berhasil dibuat", trialquizdto.ToQuestionResponse(q))
}

// GetDetail menangani GET /api/v1/trial-quiz/questions/:id.
func (h *TrialQuizHandler) GetDetail(c *gin.Context) {
	q, err := h.service.GetDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail soal quiz trial", trialquizdto.ToQuestionResponse(q))
}

// Update menangani PUT /api/v1/trial-quiz/questions/:id: ganti seluruh isi 1 soal trial.
func (h *TrialQuizHandler) Update(c *gin.Context) {
	var req trialquizdto.QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	q, err := h.service.Update(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal quiz trial berhasil diubah", trialquizdto.ToQuestionResponse(q))
}

// Delete menangani DELETE /api/v1/trial-quiz/questions/:id (soft delete).
func (h *TrialQuizHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id"), middleware.GetUserID(c)); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal quiz trial berhasil dihapus", nil)
}

// ============================================================================
// Admin: report
// ============================================================================

// ListAttempts menangani GET /api/v1/trial-quiz/attempts: daftar peserta trial + nilainya.
func (h *TrialQuizHandler) ListAttempts(c *gin.Context) {
	var query trialquizdto.AttemptsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.ListAttempts(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Data peserta quiz trial", trialquizdto.ToAttemptResponses(items), response.Meta{
		Page: f.Page, PerPage: f.PerPage, Total: total,
	})
}

// Summary menangani GET /api/v1/trial-quiz/attempts/summary: ringkasan & soal paling sering salah.
func (h *TrialQuizHandler) Summary(c *gin.Context) {
	var query trialquizdto.SummaryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	summary, err := h.service.Summary(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Ringkasan quiz trial", trialquizdto.ToSummaryResponse(summary))
}

// ============================================================================
// Publik (tanpa login)
// ============================================================================

// Grades menangani GET /api/v1/public/trial-quiz/grades: jenjang & kelas untuk form trial.
func (h *TrialQuizHandler) Grades(c *gin.Context) {
	grades, err := h.service.Grades(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	response.Success(c, http.StatusOK, "Data jenjang & kelas", trialquizdto.ToLevelResponses(grades))
}

// Questions menangani GET /api/v1/public/trial-quiz/questions?grade_id=...: soal trial kelas itu.
func (h *TrialQuizHandler) Questions(c *gin.Context) {
	var query trialquizdto.PublicQuestionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	grade, questions, err := h.service.PublicQuiz(c.Request.Context(), query.GradeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal quiz trial",
		trialquizdto.ToPublicQuizResponse(grade, questions, scoring.ScorePerQuestion(len(questions))))
}

// Submit menangani POST /api/v1/public/trial-quiz/submit: simpan data peserta + nilai.
func (h *TrialQuizHandler) Submit(c *gin.Context) {
	var req trialquizdto.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	attempt, err := h.service.Submit(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Hasil quiz trial berhasil disimpan", trialquizdto.ToSubmitResponse(attempt))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *TrialQuizHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *questionmodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		message := validationErr.Message
		if message == "" {
			message = "Validasi gagal"
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, message, fields)
		return
	}

	switch {
	case errors.Is(err, trialquizmodel.ErrQuestionNotFound):
		response.Error(c, http.StatusNotFound, "Soal quiz trial tidak ditemukan")
	case errors.Is(err, trialquizmodel.ErrNoQuestions):
		response.Error(c, http.StatusNotFound, "Quiz trial untuk kelas ini belum tersedia")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
