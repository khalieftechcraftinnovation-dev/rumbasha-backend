// Package quizreport (handler) menangani endpoint report quiz Uji Kom & Tryout.
//
// Di package lain, import dengan alias quizreporthandler:
//
//	quizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quizreport"
package quizreport

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	quizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quizreport"
	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quizreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	quizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quizreport"
)

// QuizReportHandler menangani endpoint /quiz-reports.
type QuizReportHandler struct {
	service *quizreportservice.QuizReportService
}

// NewQuizReportHandler membuat QuizReportHandler. Dipanggil sekali di main.go.
func NewQuizReportHandler(service *quizreportservice.QuizReportService) *QuizReportHandler {
	return &QuizReportHandler{service: service}
}

// List menangani GET /api/v1/quiz-reports: ringkasan per event (tanggal, nama quiz, total peserta,
// nilai rata-rata / tertinggi / terendah), event terbaru dulu.
func (h *QuizReportHandler) List(c *gin.Context) {
	var q quizreportdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Report quiz", quizreportdto.ToSummaryResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// Detail menangani GET /api/v1/quiz-reports/:id: ringkasan event + peserta urut peringkat.
func (h *QuizReportHandler) Detail(c *gin.Context) {
	var q quizreportdto.DetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	summary, participants, f, total, err := h.service.Detail(c.Request.Context(), c.Param("id"), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Detail report quiz", quizreportdto.ToDetailResponse(summary, participants),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// Questions menangani GET /api/v1/quiz-reports/:id/questions: soal urut dari yang paling sering salah.
func (h *QuizReportHandler) Questions(c *gin.Context) {
	summary, stats, err := h.service.Questions(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Analisis soal quiz", quizreportdto.ToQuestionsResponse(summary, stats))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *QuizReportHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *quizreportmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, quizreportmodel.ErrQuizNotFound):
		response.Error(c, http.StatusNotFound, "Quiz tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
