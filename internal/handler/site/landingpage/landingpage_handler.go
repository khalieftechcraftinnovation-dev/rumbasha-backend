// Package landingpage (handler) menangani endpoint Landing Page (admin & publik).
//
// Di package lain, import dengan alias landingpagehandler:
//
//	landingpagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/site/landingpage"
package landingpage

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	landingpagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/site/landingpage"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	landingpagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/site/landingpage"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	landingpageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/site/landingpage"
)

// LandingPageHandler menangani endpoint /landing-pages dan /public/landing-pages.
type LandingPageHandler struct {
	service *landingpageservice.LandingPageService
}

// NewLandingPageHandler membuat LandingPageHandler. Dipanggil sekali di main.go.
func NewLandingPageHandler(service *landingpageservice.LandingPageService) *LandingPageHandler {
	return &LandingPageHandler{service: service}
}

// ListPages menangani GET /api/v1/landing-pages: semua tab + info publish & perubahan tertunda.
func (h *LandingPageHandler) ListPages(c *gin.Context) {
	pages, err := h.service.ListPages(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Data halaman landing page", landingpagedto.ToPageResponses(pages))
}

// GetDraft menangani GET /api/v1/landing-pages/:page: section draft + panel "Belum dipublish".
func (h *LandingPageHandler) GetDraft(c *gin.Context) {
	page, sections, changes, err := h.service.GetDraft(c.Request.Context(), c.Param("page"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Data landing page", landingpagedto.ToDraftResponse(page, sections, changes))
}

// UpdateContent menangani PUT /api/v1/landing-pages/:page/sections/:key: ganti isi section (draft).
func (h *LandingPageHandler) UpdateContent(c *gin.Context) {
	var req landingpagedto.UpdateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	err := h.service.UpdateContent(c.Request.Context(), c.Param("page"), c.Param("key"), req.Content, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	h.respondDraft(c, "Isi section disimpan (belum dipublish)")
}

// SetVisible menangani PATCH /api/v1/landing-pages/:page/sections/:key/visibility (saklar "Tampil").
func (h *LandingPageHandler) SetVisible(c *gin.Context) {
	var req landingpagedto.SetVisibleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	err := h.service.SetVisible(c.Request.Context(), c.Param("page"), c.Param("key"), *req.IsVisible, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	h.respondDraft(c, "Saklar tampil disimpan (belum dipublish)")
}

// Reorder menangani PUT /api/v1/landing-pages/:page/sections/order: urutan section (draft).
func (h *LandingPageHandler) Reorder(c *gin.Context) {
	var req landingpagedto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	if err := h.service.Reorder(c.Request.Context(), c.Param("page"), req.Keys, middleware.GetUserID(c)); err != nil {
		h.handleError(c, err)
		return
	}
	h.respondDraft(c, "Urutan section disimpan (belum dipublish)")
}

// respondDraft membalas dengan data panel admin terbaru, supaya FE tidak perlu GET ulang.
func (h *LandingPageHandler) respondDraft(c *gin.Context, message string) {
	page, sections, changes, err := h.service.GetDraft(c.Request.Context(), c.Param("page"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, landingpagedto.ToDraftResponse(page, sections, changes))
}

// Publish menangani POST /api/v1/landing-pages/:page/publish: tayangkan draft ke situs publik.
func (h *LandingPageHandler) Publish(c *gin.Context) {
	result, err := h.service.Publish(c.Request.Context(), c.Param("page"), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, fmt.Sprintf("Landing page dipublish (%d perubahan)", len(result.Changes)),
		landingpagedto.ToPublishResponse(result))
}

// GetPublic menangani GET /api/v1/public/landing-pages/:page (tanpa login): isi halaman publik.
func (h *LandingPageHandler) GetPublic(c *gin.Context) {
	page, err := h.service.GetPublished(c.Request.Context(), c.Param("page"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	// Boleh disimpan browser/CDN sebentar (isi hanya berubah saat Publish).
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, http.StatusOK, "Data landing page", landingpagedto.ToPublicPageResponse(page))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *LandingPageHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *landingpagemodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
		return
	}

	switch {
	case errors.Is(err, landingpagemodel.ErrPageNotFound):
		response.Error(c, http.StatusNotFound, "Halaman landing page tidak ditemukan")
	case errors.Is(err, landingpagemodel.ErrSectionNotFound):
		response.Error(c, http.StatusNotFound, "Section landing page tidak ditemukan")
	case errors.Is(err, landingpagemodel.ErrNothingToPublish):
		response.Error(c, http.StatusConflict, "Tidak ada perubahan yang perlu dipublish")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
