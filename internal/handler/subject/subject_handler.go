// Package subject (handler) menangani endpoint /subjects.
// Tugas handler: baca request → panggil service → kirim response.
// Aturan bisnis TIDAK boleh ditulis di sini, tempatnya di service.
//
// Di package lain, import dengan alias subjecthandler:
//
//	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
package subject

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto"
	subjectdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subject"
	subjectmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subject"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	subjectservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subject"
)

// SubjectHandler menangani semua endpoint /subjects.
type SubjectHandler struct {
	service *subjectservice.SubjectService
}

// NewSubjectHandler membuat SubjectHandler. Dipanggil sekali di main.go.
func NewSubjectHandler(service *subjectservice.SubjectService) *SubjectHandler {
	return &SubjectHandler{service: service}
}

// Create menangani POST /api/v1/subjects.
func (h *SubjectHandler) Create(c *gin.Context) {
	var req subjectdto.CreateSubjectRequest

	// ShouldBindJSON mengisi req dari body JSON, sekaligus menjalankan
	// validasi sesuai tag `binding` di subjectdto.CreateSubjectRequest.
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	// c.Request.Context() diteruskan ke service & repository. Kalau client
	// membatalkan request, query database ikut dibatalkan.
	subject, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Mata pelajaran berhasil dibuat", subjectdto.ToSubjectResponse(subject))
}

// GetByID menangani GET /api/v1/subjects/:id.
func (h *SubjectHandler) GetByID(c *gin.Context) {
	subject, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Data mata pelajaran berhasil diambil", subjectdto.ToSubjectResponse(subject))
}

// List menangani GET /api/v1/subjects?page=1&per_page=20.
func (h *SubjectHandler) List(c *gin.Context) {
	// dto.PaginationQuery ada di root folder dto karena dipakai semua fitur.
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

	response.SuccessWithMeta(c, "Data mata pelajaran berhasil diambil", subjectdto.ToSubjectResponses(subjects), response.Meta{
		Page:    query.Page,
		PerPage: query.PerPage,
		Total:   total,
	})
}

// Update menangani PUT /api/v1/subjects/:id.
func (h *SubjectHandler) Update(c *gin.Context) {
	var req subjectdto.UpdateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	subject, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Mata pelajaran berhasil diubah", subjectdto.ToSubjectResponse(subject))
}

// Delete menangani DELETE /api/v1/subjects/:id.
func (h *SubjectHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Mata pelajaran berhasil dihapus", nil)
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *SubjectHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subjectmodel.ErrSubjectNotFound):
		response.Error(c, http.StatusNotFound, "Mata pelajaran tidak ditemukan")
	case errors.Is(err, subjectmodel.ErrSubjectCodeTaken):
		response.Error(c, http.StatusConflict, "Kode mata pelajaran sudah digunakan")
	default:
		// Error yang tidak dikenal (misalnya database mati): detailnya hanya
		// dicatat di log, client cukup tahu ada kesalahan di server.
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
