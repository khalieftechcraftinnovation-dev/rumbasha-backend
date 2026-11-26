// Package learningmodule (handler) menangani endpoint modul pembelajaran (materi PDF).
//
// Di package lain, import dengan alias learningmodulehandler:
//
//	learningmodulehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/learning/learningmodule"
package learningmodule

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	learningmoduledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/learning/learningmodule"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learning/learningmodule"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	learningmoduleservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/learning/learningmodule"
)

// LearningModuleHandler menangani endpoint /learning-modules.
type LearningModuleHandler struct {
	service *learningmoduleservice.LearningModuleService
}

// NewLearningModuleHandler membuat LearningModuleHandler. Dipanggil sekali di main.go.
func NewLearningModuleHandler(service *learningmoduleservice.LearningModuleService) *LearningModuleHandler {
	return &LearningModuleHandler{service: service}
}

// Create menangani POST /api/v1/learning-modules (multipart/form-data): upload 1 PDF + data modul.
func (h *LearningModuleHandler) Create(c *gin.Context) {
	req, file, ok := h.readForm(c)
	if !ok {
		return
	}
	module, err := h.service.Create(c.Request.Context(), req, file, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Modul berhasil dibuat", learningmoduledto.ToModuleResponse(module, h.service.FileURL(module.ID)))
}

// Update menangani PUT /api/v1/learning-modules/:id (multipart/form-data, ganti penuh).
// Field "file" opsional: diisi = PDF diganti.
func (h *LearningModuleHandler) Update(c *gin.Context) {
	req, file, ok := h.readForm(c)
	if !ok {
		return
	}
	module, err := h.service.Update(c.Request.Context(), c.Param("id"), req, file, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Modul berhasil diubah", learningmoduledto.ToModuleResponse(module, h.service.FileURL(module.ID)))
}

// readForm membaca field teks + file "file" (opsional) dari form multipart. ok = false kalau
// response error sudah dikirim.
func (h *LearningModuleHandler) readForm(c *gin.Context) (learningmoduledto.ModuleRequest, *learningmoduleservice.UploadFile, bool) {
	// Batasi ukuran total request sebelum dibaca: 1 PDF + 1 MB untuk field lain.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, learningmoduleservice.MaxFileSize+(1<<20))

	var req learningmoduledto.ModuleRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err)
		return req, nil, false
	}
	fh, err := c.FormFile("file")
	if errors.Is(err, http.ErrMissingFile) {
		return req, nil, true
	}
	if err != nil {
		_ = c.Error(err)
		response.Error(c, http.StatusBadRequest, "Request harus multipart/form-data dan ukuran file maksimal 20 MB")
		return req, nil, false
	}

	file := &learningmoduleservice.UploadFile{FileName: fh.Filename, Size: fh.Size}
	if fh.Size <= learningmoduleservice.MaxFileSize { // terlalu besar: tidak dibaca, ditolak service
		f, err := fh.Open()
		if err == nil {
			file.Data, err = io.ReadAll(f)
			f.Close()
		}
		if err != nil {
			h.handleError(c, err)
			return req, nil, false
		}
	}
	return req, file, true
}

// Get menangani GET /api/v1/learning-modules/:id.
func (h *LearningModuleHandler) Get(c *gin.Context) {
	module, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail modul", learningmoduledto.ToModuleResponse(module, h.service.FileURL(module.ID)))
}

// List menangani GET /api/v1/learning-modules: list modul terbaru dulu (filter & cari).
func (h *LearningModuleHandler) List(c *gin.Context) {
	var q learningmoduledto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	modules, f, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	items := make([]learningmoduledto.ModuleResponse, 0, len(modules))
	for _, m := range modules {
		items = append(items, learningmoduledto.ToModuleResponse(m, h.service.FileURL(m.ID)))
	}
	response.SuccessWithMeta(c, "Daftar modul", items, response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// File menangani GET /api/v1/learning-modules/:id/file: mengirim PDF (tampil langsung di browser).
func (h *LearningModuleHandler) File(c *gin.Context) {
	path, fileName, err := h.service.FilePath(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	sendPDF(c, path, fileName)
}

// Delete menangani DELETE /api/v1/learning-modules, body {"ids": [...]} (soft delete).
func (h *LearningModuleHandler) Delete(c *gin.Context) {
	var req learningmoduledto.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.DeleteMany(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Modul berhasil dihapus", learningmoduledto.DeleteResponse{DeletedIDs: ids})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *LearningModuleHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *learningmodulemodel.ValidationError
	var lockedErr *learningmodulemodel.LockedError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.As(err, &lockedErr):
		plans := make([]learningmoduledto.RefResponse, 0, len(lockedErr.Plans))
		for _, p := range lockedErr.Plans {
			plans = append(plans, learningmoduledto.RefResponse(p))
		}
		// 403 + data.plans = paket yang membuka modul (FE bisa menampilkan ajakan upgrade).
		c.JSON(http.StatusForbidden, response.Body{StatusCode: http.StatusForbidden, Message: lockedErr.Message, Data: gin.H{"plans": plans}})
	case errors.Is(err, learningmodulemodel.ErrModuleNotFound):
		response.Error(c, http.StatusNotFound, "Modul tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
