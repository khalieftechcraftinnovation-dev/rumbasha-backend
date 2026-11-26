// Package masterdata (handler) menangani endpoint menu Master Data & Cabang (panel admin).
//
// Di package lain, import dengan alias masterdatahandler:
//
//	masterdatahandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/masterdata"
package masterdata

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	masterdatadto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/masterdata"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/masterdata"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	masterdataservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/masterdata"
)

// MasterDataHandler menangani endpoint Master Data & Cabang.
type MasterDataHandler struct {
	service *masterdataservice.MasterDataService
}

// NewMasterDataHandler membuat MasterDataHandler. Dipanggil sekali di main.go.
func NewMasterDataHandler(service *masterdataservice.MasterDataService) *MasterDataHandler {
	return &MasterDataHandler{service: service}
}

// List menangani GET /api/v1/master-data: 1 halaman data master (tab, filter letak, cari, status).
func (h *MasterDataHandler) List(c *gin.Context) {
	var q masterdatadto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar master data", masterdatadto.ToItemResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// Counts menangani GET /api/v1/master-data/counts: angka tab (filter search & status ikut).
func (h *MasterDataHandler) Counts(c *gin.Context) {
	var q masterdatadto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	counts, branches, err := h.service.Counts(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jumlah master data per tab", masterdatadto.ToCountsResponse(counts, branches))
}

// Get menangani GET /api/v1/master-data/:id.
func (h *MasterDataHandler) Get(c *gin.Context) {
	item, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail master data", masterdatadto.ToItemResponse(item))
}

// Create menangani POST /api/v1/master-data.
func (h *MasterDataHandler) Create(c *gin.Context) {
	var req masterdatadto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	item, err := h.service.Create(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Master data berhasil ditambahkan", masterdatadto.ToItemResponse(item))
}

// Update menangani PUT /api/v1/master-data/:id (ganti penuh: nama, kode, urutan, status).
func (h *MasterDataHandler) Update(c *gin.Context) {
	var req masterdatadto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Master data berhasil diubah", masterdatadto.ToItemResponse(item))
}

// Delete menangani DELETE /api/v1/master-data, body {"ids": [...]} (soft delete).
func (h *MasterDataHandler) Delete(c *gin.Context) {
	var req masterdatadto.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.Delete(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Master data berhasil dihapus", masterdatadto.DeleteResponse{DeletedIDs: ids})
}

// ListBranches menangani GET /api/v1/branches.
func (h *MasterDataHandler) ListBranches(c *gin.Context) {
	var q masterdatadto.BranchListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, err := h.service.ListBranches(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar cabang", masterdatadto.ToBranchResponses(items))
}

// GetBranch menangani GET /api/v1/branches/:id.
func (h *MasterDataHandler) GetBranch(c *gin.Context) {
	b, err := h.service.GetBranch(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail cabang", masterdatadto.ToBranchResponse(b))
}

// CreateBranch menangani POST /api/v1/branches.
func (h *MasterDataHandler) CreateBranch(c *gin.Context) {
	var req masterdatadto.BranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	b, err := h.service.CreateBranch(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Cabang berhasil ditambahkan", masterdatadto.ToBranchResponse(b))
}

// UpdateBranch menangani PUT /api/v1/branches/:id (ganti penuh).
func (h *MasterDataHandler) UpdateBranch(c *gin.Context) {
	var req masterdatadto.BranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	b, err := h.service.UpdateBranch(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Cabang berhasil diubah", masterdatadto.ToBranchResponse(b))
}

// DeleteBranches menangani DELETE /api/v1/branches, body {"ids": [...]} (soft delete).
func (h *MasterDataHandler) DeleteBranches(c *gin.Context) {
	var req masterdatadto.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.DeleteBranches(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Cabang berhasil dihapus", masterdatadto.DeleteResponse{DeletedIDs: ids})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *MasterDataHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *masterdatamodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, masterdatamodel.ErrNotFound):
		response.Error(c, http.StatusNotFound, "Master data tidak ditemukan")
	case errors.Is(err, masterdatamodel.ErrBranchNotFound):
		response.Error(c, http.StatusNotFound, "Cabang tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
