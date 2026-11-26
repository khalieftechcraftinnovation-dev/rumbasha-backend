// Package profile (handler) menangani endpoint halaman Profil siswa.
//
// Di package lain, import dengan alias profilehandler:
//
//	profilehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/profile"
package profile

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	profiledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/profile"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	profilemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/profile"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	profileservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/profile"
)

// ProfileHandler menangani endpoint Profil siswa.
type ProfileHandler struct {
	service *profileservice.ProfileService
}

// NewProfileHandler membuat ProfileHandler. Dipanggil sekali di main.go.
func NewProfileHandler(service *profileservice.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// Get menangani GET /api/v1/student/profile: data diri, paket aktif, coin, dan statistik.
func (h *ProfileHandler) Get(c *gin.Context) {
	p, access, stats, err := h.service.Summary(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil siswa", profiledto.ToSummaryResponse(p, access, stats))
}

// Update menangani PUT /api/v1/student/profile: ubah data diri (ganti penuh).
func (h *ProfileHandler) Update(c *gin.Context) {
	var req profiledto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.Update(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Data diri berhasil diubah", profiledto.ToProfileResponse(p))
}

// UpdateParentContact menangani PUT /api/v1/student/profile/parent-contact.
func (h *ProfileHandler) UpdateParentContact(c *gin.Context) {
	var req profiledto.ParentContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.UpdateParentContact(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Kontak orang tua / wali berhasil disimpan", profiledto.ToProfileResponse(p))
}

// UploadPhoto menangani POST /api/v1/student/profile/photo (multipart, field "photo").
func (h *ProfileHandler) UploadPhoto(c *gin.Context) {
	// Batasi ukuran request sebelum dibaca: foto 2 MB + sedikit untuk header multipart.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, profileservice.MaxAvatarSize+(1<<20))
	fh, err := c.FormFile("photo")
	if err != nil {
		_ = c.Error(err)
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal",
			[]response.FieldError{{Field: "photo", Message: "wajib diisi (multipart/form-data, maks 2 MB)"}})
		return
	}
	var data []byte
	if fh.Size <= profileservice.MaxAvatarSize { // file terlalu besar tidak dibaca (ditolak service)
		f, err := fh.Open()
		if err != nil {
			h.handleError(c, err)
			return
		}
		defer f.Close()
		if data, err = io.ReadAll(f); err != nil {
			h.handleError(c, err)
			return
		}
	}
	p, err := h.service.UploadAvatar(c.Request.Context(), middleware.GetUserID(c), fh.Size, data)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Foto profil berhasil diganti", profiledto.ToProfileResponse(p))
}

// DeletePhoto menangani DELETE /api/v1/student/profile/photo.
func (h *ProfileHandler) DeletePhoto(c *gin.Context) {
	p, err := h.service.DeleteAvatar(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Foto profil dihapus", profiledto.ToProfileResponse(p))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *ProfileHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")
	var validationErr *profilemodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, profilemodel.ErrNotFound):
		response.Error(c, http.StatusNotFound, "Profil tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
