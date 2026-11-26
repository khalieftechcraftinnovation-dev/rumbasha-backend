// Package passwordreset (handler) menangani endpoint "Lupa kata sandi" (form publik) dan daftar
// permintaan untuk admin.
//
// Di package lain, import dengan alias passwordresethandler:
//
//	passwordresethandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/passwordreset"
package passwordreset

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	passwordresetdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/passwordreset"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	passwordresetmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/passwordreset"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	passwordresetservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/passwordreset"
)

// PasswordResetHandler menangani endpoint lupa kata sandi.
type PasswordResetHandler struct {
	service *passwordresetservice.PasswordResetService
}

// NewPasswordResetHandler membuat PasswordResetHandler. Dipanggil sekali di main.go.
func NewPasswordResetHandler(service *passwordresetservice.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{service: service}
}

// ForgotMessage = jawaban form lupa kata sandi (sama untuk semua isian).
const ForgotMessage = "Permintaan reset kata sandi sudah dikirim ke admin. Admin akan menghubungi kamu " +
	"lewat kontak yang terdaftar. Untuk bantuan lebih cepat, gunakan Chat admin."

// Forgot menangani POST /api/v1/auth/forgot-password (tanpa login). Selalu 200 dengan pesan yang sama
// (kecuali format request salah / server error), supaya tidak bisa dipakai menebak akun.
func (h *PasswordResetHandler) Forgot(c *gin.Context) {
	var req passwordresetdto.ForgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	if err := h.service.Forgot(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent()); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, ForgotMessage, nil)
}

// List menangani GET /api/v1/password-reset-requests (admin & super admin).
func (h *PasswordResetHandler) List(c *gin.Context) {
	var q passwordresetdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.List(c.Request.Context(), q, middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar permintaan reset kata sandi", passwordresetdto.ToRequestResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// Counts menangani GET /api/v1/password-reset-requests/counts (badge menu = pending).
func (h *PasswordResetHandler) Counts(c *gin.Context) {
	var q passwordresetdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	counts, err := h.service.Counts(c.Request.Context(), q, middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jumlah permintaan reset kata sandi",
		passwordresetdto.CountsResponse{Pending: counts.Pending, Resolved: counts.Resolved, Rejected: counts.Rejected})
}

// Get menangani GET /api/v1/password-reset-requests/:id.
func (h *PasswordResetHandler) Get(c *gin.Context) {
	r, err := h.service.Get(c.Request.Context(), c.Param("id"), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail permintaan reset kata sandi", passwordresetdto.ToRequestResponse(r))
}

// Resolve menangani POST /api/v1/password-reset-requests/:id/resolve: set kata sandi baru user.
func (h *PasswordResetHandler) Resolve(c *gin.Context) {
	var req passwordresetdto.ResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	r, err := h.service.Resolve(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Kata sandi user berhasil direset", passwordresetdto.ToRequestResponse(r))
}

// Reject menangani POST /api/v1/password-reset-requests/:id/reject (body opsional {"note": "..."}).
func (h *PasswordResetHandler) Reject(c *gin.Context) {
	var req passwordresetdto.RejectRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
	}
	r, err := h.service.Reject(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Permintaan reset kata sandi ditolak", passwordresetdto.ToRequestResponse(r))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *PasswordResetHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")
	switch {
	case errors.Is(err, passwordresetmodel.ErrNotFound):
		response.Error(c, http.StatusNotFound, "Permintaan reset kata sandi tidak ditemukan")
	case errors.Is(err, passwordresetmodel.ErrForbidden):
		response.Error(c, http.StatusForbidden, "Admin hanya boleh mereset kata sandi akun tutor dan siswa")
	case errors.Is(err, passwordresetmodel.ErrNotPending):
		response.Error(c, http.StatusConflict, "Permintaan ini sudah diproses sebelumnya")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
