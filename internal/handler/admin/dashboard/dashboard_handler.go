// Package dashboard handles Admin dashboard endpoints.
package dashboard

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	dashboarddto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/admin/dashboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	dashboardservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/admin/dashboard"
)

type DashboardHandler struct {
	service *dashboardservice.DashboardService
}

func NewDashboardHandler(service *dashboardservice.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	summary, err := h.service.Summary(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Ringkasan dashboard Admin", dashboarddto.ToSummaryResponse(summary))
}

func (h *DashboardHandler) DailyQuizTrend(c *gin.Context) {
	var query dashboarddto.DailyQuizTrendQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	if query.Period == "" {
		query.Period = dashboardservice.TrendPeriod12Weeks
	}

	trend, err := h.service.DailyQuizTrend(c.Request.Context(), query.Period)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Tren nilai Daily Quiz", dashboarddto.ToDailyQuizTrendResponse(trend))
}

func (h *DashboardHandler) Search(c *gin.Context) {
	var query dashboarddto.SearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	query.Query = strings.TrimSpace(query.Query)
	queryLength := utf8.RuneCountInString(query.Query)
	if queryLength < 3 || queryLength > 100 {
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: "q", Message: "harus 3-100 karakter setelah spasi di awal/akhir dihapus"},
		})
		return
	}

	results, err := h.service.Search(c.Request.Context(), query.Query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Hasil pencarian dashboard", dashboarddto.ToSearchResponse(results))
}

func (h *DashboardHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, dashboardservice.ErrInvalidSearchQuery) {
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: "q", Message: "harus 3-100 karakter"},
		})
		return
	}
	_ = c.Error(err)
	log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
}
