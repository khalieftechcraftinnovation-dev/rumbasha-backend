// Package coin (handler) menangani endpoint coin: paket isi coin, top-up oleh admin, riwayat coin.
//
// Di package lain, import dengan alias coinhandler:
//
//	coinhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/billing/coin"
package coin

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	coindto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/billing/coin"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/coin"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	coinservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/coin"
)

// CoinHandler menangani endpoint coin.
type CoinHandler struct {
	service *coinservice.CoinService
}

// NewCoinHandler membuat CoinHandler. Dipanggil sekali di main.go.
func NewCoinHandler(service *coinservice.CoinService) *CoinHandler {
	return &CoinHandler{service: service}
}

// ListPackages menangani GET /api/v1/coin-packages. Siswa: hanya paket aktif (tab "Isi coin"), tanpa
// data admin. Admin & super admin: semua paket (filter status, search nama).
func (h *CoinHandler) ListPackages(c *gin.Context) {
	var q coindto.ListPackagesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	isStudent := middleware.GetRole(c) == loginmodel.RoleStudent
	packages, err := h.service.ListPackages(c.Request.Context(), q, isStudent)
	if err != nil {
		h.handleError(c, err)
		return
	}
	if isStudent {
		response.Success(c, http.StatusOK, "Daftar paket coin", coindto.ToStudentPackageResponses(packages))
		return
	}
	response.Success(c, http.StatusOK, "Daftar paket coin", coindto.ToPackageResponses(packages))
}

// GetPackage menangani GET /api/v1/coin-packages/:id (admin & super admin).
func (h *CoinHandler) GetPackage(c *gin.Context) {
	p, err := h.service.GetPackage(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail paket coin", coindto.ToPackageResponse(p))
}

// CreatePackage menangani POST /api/v1/coin-packages (admin & super admin).
func (h *CoinHandler) CreatePackage(c *gin.Context) {
	var req coindto.PackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.CreatePackage(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Paket coin berhasil dibuat", coindto.ToPackageResponse(p))
}

// UpdatePackage menangani PUT /api/v1/coin-packages/:id (admin & super admin, ganti penuh).
func (h *CoinHandler) UpdatePackage(c *gin.Context) {
	var req coindto.PackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.UpdatePackage(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Paket coin berhasil diubah", coindto.ToPackageResponse(p))
}

// DeletePackages menangani DELETE /api/v1/coin-packages, body {"ids": [...]} (soft delete).
func (h *CoinHandler) DeletePackages(c *gin.Context) {
	var req coindto.DeletePackagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.DeletePackages(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Paket coin berhasil dihapus", coindto.DeletePackagesResponse{DeletedIDs: ids})
}

// TopUp menangani POST /api/v1/coin-topups: admin menambah coin banyak siswa sekaligus sesuai 1 paket.
func (h *CoinHandler) TopUp(c *gin.Context) {
	var req coindto.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	result, err := h.service.TopUp(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Coin siswa berhasil ditambahkan", coindto.ToTopUpResponse(result))
}

// MyHistory menangani GET /api/v1/student/coin-transactions: saldo, jatah Daily Quiz hari ini, dan
// riwayat coin milik siswa yang login.
func (h *CoinHandler) MyHistory(c *gin.Context) {
	h.history(c, middleware.GetUserID(c), true)
}

// StudentHistory menangani GET /api/v1/students/:id/coin-transactions: riwayat coin 1 siswa untuk
// admin & super admin.
func (h *CoinHandler) StudentHistory(c *gin.Context) {
	h.history(c, c.Param("id"), false)
}

func (h *CoinHandler) history(c *gin.Context, studentID string, withQuota bool) {
	var q coindto.TransactionsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	student, items, quota, f, total, err := h.service.History(c.Request.Context(), studentID, q, withQuota)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Riwayat coin", coindto.ToHistoryResponse(student, items, quota),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *CoinHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *coinmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, coinmodel.ErrPackageNotFound):
		response.Error(c, http.StatusNotFound, "Paket coin tidak ditemukan")
	case errors.Is(err, coinmodel.ErrStudentNotFound):
		response.Error(c, http.StatusNotFound, "Siswa tidak ditemukan")
	case errors.Is(err, coinmodel.ErrTopUpDuplicate):
		response.Error(c, http.StatusConflict, "Top-up ini sudah diproses sebelumnya (request_id sama), coin tidak ditambah lagi")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
