// Package user (handler) menangani endpoint menu User (panel admin).
//
// Di package lain, import dengan alias userhandler:
//
//	userhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user"
package user

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	userdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	userservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user"
)

// UserHandler menangani endpoint /users.
type UserHandler struct {
	service *userservice.UserService
}

// NewUserHandler membuat UserHandler. Dipanggil sekali di main.go.
func NewUserHandler(service *userservice.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// List menangani GET /api/v1/users: list user (terbaru dulu) dengan filter role, jenjang, kelas,
// cabang, status, dan cari nama/username/email.
func (h *UserHandler) List(c *gin.Context) {
	var q userdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar user", userdto.ToUserResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// RoleCounts menangani GET /api/v1/users/role-counts: jumlah user per role (angka di tab), memakai
// filter yang sama dengan list (role diabaikan).
func (h *UserHandler) RoleCounts(c *gin.Context) {
	var q userdto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	counts, err := h.service.RoleCounts(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jumlah user per role", userdto.ToRoleCountsResponse(counts))
}

// GetDetail menangani GET /api/v1/users/:id.
func (h *UserHandler) GetDetail(c *gin.Context) {
	detail, err := h.service.GetDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail user", userdto.ToUserDetailResponse(detail))
}

// Update menangani PUT /api/v1/users/:id (ganti penuh, role tetap).
func (h *UserHandler) Update(c *gin.Context) {
	var req userdto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	detail, err := h.service.Update(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User berhasil diubah", userdto.ToUserDetailResponse(detail))
}

// Delete menangani DELETE /api/v1/users, body {"ids": [...]} (soft delete).
func (h *UserHandler) Delete(c *gin.Context) {
	var req userdto.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.Delete(c.Request.Context(), req.IDs, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User berhasil dihapus", userdto.DeleteResponse{DeletedIDs: ids})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *UserHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *usermodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			// Row & Column hanya terisi untuk error dari file Excel.
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message, Row: f.Row, Column: f.Column})
		}
		message := validationErr.Message
		if message == "" {
			message = "Validasi gagal"
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, message, fields)
	case errors.Is(err, usermodel.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, "User tidak ditemukan")
	case errors.Is(err, usermodel.ErrForbidden):
		response.Error(c, http.StatusForbidden, "Admin hanya boleh mengubah akun tutor dan siswa")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// Create menangani POST /api/v1/users: tambah 1-100 akun sekaligus. data = akun yang dibuat (urut
// sesuai request, bentuk sama dengan baris list).
func (h *UserHandler) Create(c *gin.Context) {
	var req userdto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	users, err := h.service.Create(c.Request.Context(), req, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, fmt.Sprintf("%d user berhasil ditambahkan", len(users)), userdto.ToUserResponses(users))
}

// maxExcelFileSize = batas ukuran file Excel tambah user (5 MB, sama dengan import soal).
const maxExcelFileSize = 5 << 20

// CheckExcel menangani POST /api/v1/users/import/check (multipart, field "file").
// Hanya MENGECEK isi file Excel, tidak menyimpan. Valid → data berisi akun-akun yang siap dikirim
// ke POST /api/v1/users. Ada error → 422 berisi SEMUA error lengkap dengan baris & kolom Excel.
func (h *UserHandler) CheckExcel(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxExcelFileSize+(1<<20))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(err)
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: "file", Message: "wajib diisi (file .xlsx, maksimal 5 MB)"},
		})
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xlsx") || fileHeader.Size > maxExcelFileSize {
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: "file", Message: "harus file .xlsx dengan ukuran maksimal 5 MB"},
		})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer file.Close()

	users, err := h.service.CheckExcel(c.Request.Context(), file, middleware.GetRole(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, fmt.Sprintf("File valid: %d user siap disimpan", len(users)), userdto.ToImportCheckResponse(users))
}
