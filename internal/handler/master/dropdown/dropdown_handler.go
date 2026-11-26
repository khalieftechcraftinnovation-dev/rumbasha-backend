// Package dropdown (handler) menangani endpoint dropdown (pilihan dari master data).
//
// Di package lain, import dengan alias dropdownhandler:
//
//	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/dropdown"
package dropdown

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	dropdowndto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/dropdown"
	dropdownmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/dropdown"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/dropdown"
)

// DropdownHandler menangani endpoint /dropdowns.
type DropdownHandler struct {
	dropdownService *dropdownservice.DropdownService
}

// NewDropdownHandler membuat DropdownHandler. Dipanggil sekali di main.go.
func NewDropdownHandler(dropdownService *dropdownservice.DropdownService) *DropdownHandler {
	return &DropdownHandler{dropdownService: dropdownService}
}

// List menangani GET /api/v1/dropdowns/:type, contoh:
//
//	/api/v1/dropdowns/difficulty
//	/api/v1/dropdowns/subject_field?subject_id=...&grade_id=...
//
// data = daftar pilihan aktif, urut sesuai sort_order. Lihat dropdowndto.DropdownQuery untuk filter.
func (h *DropdownHandler) List(c *gin.Context) {
	var query dropdowndto.DropdownQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	options, err := h.dropdownService.List(c.Request.Context(), c.Param("type"), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Data dropdown", dropdowndto.ToOptionResponses(options))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *DropdownHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	if errors.Is(err, dropdownmodel.ErrUnknownType) {
		response.Error(c, http.StatusNotFound,
			"Dropdown tidak ditemukan. Pilihan: "+strings.Join(dropdownservice.Types(), ", "))
		return
	}

	var validationErr *dropdownmodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
		return
	}

	log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
}
