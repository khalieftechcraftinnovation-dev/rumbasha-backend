// Package rumath handles RUMATH student endpoints.
package rumath

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	rumathdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/rumath"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	rumathmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	rumathservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/rumath"
)

type RumathHandler struct {
	service *rumathservice.RumathService
}

func NewRumathHandler(service *rumathservice.RumathService) *RumathHandler {
	return &RumathHandler{service: service}
}

func (h *RumathHandler) GenerateSession(c *gin.Context) {
	var req rumathdto.GenerateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	generated, err := h.service.GenerateSession(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Sesi RUMATH berhasil dibuat", rumathdto.ToGenerateSessionResponse(generated))
}

func (h *RumathHandler) handleError(c *gin.Context, err error) {
	var validationErr *rumathmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, field := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: field.Field, Message: field.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, rumathmodel.ErrAccessRequired):
		response.Error(c, http.StatusForbidden, "Akses RUMATH belum tersedia untuk akun kamu")
	default:
		_ = c.Error(err)
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
