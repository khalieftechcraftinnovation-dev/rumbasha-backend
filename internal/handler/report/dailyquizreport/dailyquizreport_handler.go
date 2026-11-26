// Package dailyquizreport (handler) menangani endpoint report Daily Quiz per siswa.
//
// Di package lain, import dengan alias dailyquizreporthandler:
//
//	dailyquizreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/report/dailyquizreport"
package dailyquizreport

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	dailyquizreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/report/dailyquizreport"
	dailyquizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/dailyquizreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	dailyquizreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/dailyquizreport"
)

// DailyQuizReportHandler menangani endpoint /daily-quiz-reports.
type DailyQuizReportHandler struct {
	service *dailyquizreportservice.DailyQuizReportService
}

// NewDailyQuizReportHandler membuat DailyQuizReportHandler. Dipanggil sekali di main.go.
func NewDailyQuizReportHandler(service *dailyquizreportservice.DailyQuizReportService) *DailyQuizReportHandler {
	return &DailyQuizReportHandler{service: service}
}

// List menangani GET /api/v1/daily-quiz-reports: siswa urut peringkat + rekap Daily Quiz dalam periode.
func (h *DailyQuizReportHandler) List(c *gin.Context) {
	var q dailyquizreportdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	rows, f, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Report Daily Quiz", dailyquizreportdto.ToListResponse(f.Period, rows),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// StudentDetail menangani GET /api/v1/daily-quiz-reports/students/:id: rekap 1 siswa + per mapel.
func (h *DailyQuizReportHandler) StudentDetail(c *gin.Context) {
	var q dailyquizreportdto.PeriodQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	student, period, total, subjects, err := h.service.StudentDetail(c.Request.Context(), c.Param("id"), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Report Daily Quiz siswa",
		dailyquizreportdto.ToStudentDetailResponse(student, period, total, subjects))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *DailyQuizReportHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *dailyquizreportmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, dailyquizreportmodel.ErrStudentNotFound):
		response.Error(c, http.StatusNotFound, "Siswa tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
