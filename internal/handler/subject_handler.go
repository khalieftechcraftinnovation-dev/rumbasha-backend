// Package handler adalah layer HANDLER (controller).
// Tugas handler: baca request → panggil service → kirim response.
// Aturan bisnis TIDAK boleh ditulis di sini, tempatnya di service.
package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service"
)

// SubjectHandler menangani semua endpoint /subjects.
type SubjectHandler struct {
	service *service.SubjectService
}

// NewSubjectHandler membuat SubjectHandler. Dipanggil sekali di main.go.
func NewSubjectHandler(service *service.SubjectService) *SubjectHandler {
	return &SubjectHandler{service: service}
}

// Create menangani POST /api/v1/subjects.
//
// Catatan Go: (h *SubjectHandler) di depan nama fungsi artinya Create adalah
// method milik SubjectHandler, mirip $this di PHP.
func (h *SubjectHandler) Create(c *gin.Context) {
	var req dto.CreateSubjectRequest

	// ShouldBindJSON mengisi req dari body JSON, sekaligus menjalankan
	// validasi sesuai tag `binding` di dto.CreateSubjectRequest.
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	// c.Request.Context() diteruskan ke service & repository. Kalau client
	// membatalkan request, query database nanti ikut dibatalkan.
	subject, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Mata pelajaran berhasil dibuat", dto.ToSubjectResponse(subject))
}

// GetByID menangani GET /api/v1/subjects/:id.
func (h *SubjectHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	subject, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Data mata pelajaran berhasil diambil", dto.ToSubjectResponse(subject))
}

// List menangani GET /api/v1/subjects?page=1&per_page=20.
func (h *SubjectHandler) List(c *gin.Context) {
	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	query.SetDefaults()

	subjects, total, err := h.service.List(c.Request.Context(), query.Page, query.PerPage)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Data mata pelajaran berhasil diambil", dto.ToSubjectResponses(subjects), response.Meta{
		Page:    query.Page,
		PerPage: query.PerPage,
		Total:   total,
	})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
//
// Catatan Go: Go tidak punya try/catch. Error dikembalikan sebagai nilai biasa,
// lalu dicek dengan errors.Is untuk tahu jenis error-nya.
func (h *SubjectHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrSubjectNotFound):
		response.Error(c, http.StatusNotFound, "Mata pelajaran tidak ditemukan")
	case errors.Is(err, model.ErrSubjectCodeTaken):
		response.Error(c, http.StatusConflict, "Kode mata pelajaran sudah digunakan")
	default:
		// Error yang tidak dikenal: detailnya hanya dicatat di log,
		// client cukup tahu ada kesalahan di server.
		log.Printf("error tidak terduga di %s: %v", c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
