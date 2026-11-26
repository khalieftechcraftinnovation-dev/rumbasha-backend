// Package image (handler) menangani endpoint Image Library: upload & hapus gambar.
//
// Di package lain, import dengan alias imagehandler:
//
//	imagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/image"
package image

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	imagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/image"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	imagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/image"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	imageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/image"
)

// ImageHandler menangani endpoint /images.
type ImageHandler struct {
	imageService *imageservice.ImageService
}

// NewImageHandler membuat ImageHandler. Dipanggil sekali di main.go.
func NewImageHandler(imageService *imageservice.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// Upload menangani POST /api/v1/images (multipart/form-data).
// Field: files (1-10 gambar), category, name, subject_id, subject_field_id, chapter_id,
// subchapter_id, topic_id. Response = daftar gambar yang tersimpan (urut sesuai file).
func (h *ImageHandler) Upload(c *gin.Context) {
	// Batasi ukuran total request sebelum dibaca: 10 file x 5 MB + 1 MB untuk field lain.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body,
		imageservice.MaxFilesPerUpload*imageservice.MaxFileSize+(1<<20))

	var req imagedto.UploadImagesRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		_ = c.Error(err)
		response.Error(c, http.StatusBadRequest, "Request harus multipart/form-data dan ukuran total maksimal 51 MB")
		return
	}

	// Baca isi file ke memori. File yang terlalu besar tidak dibaca (Data nil, ditolak service).
	fileHeaders := form.File["files"]
	files := make([]imageservice.UploadFile, 0, len(fileHeaders))
	for _, fh := range fileHeaders {
		file := imageservice.UploadFile{FileName: fh.Filename, Size: fh.Size}
		if fh.Size <= imageservice.MaxFileSize {
			if file.Data, err = readFile(fh); err != nil {
				h.handleError(c, fmt.Errorf("gagal membaca file %s: %w", fh.Filename, err))
				return
			}
		}
		files = append(files, file)
	}

	images, err := h.imageService.UploadMany(c.Request.Context(), req, files, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, fmt.Sprintf("%d gambar berhasil diupload", len(images)),
		imagedto.ToImageResponses(images))
}

// readFile membuka lalu membaca seluruh isi 1 file upload.
func readFile(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Delete menangani DELETE /api/v1/images: hapus permanen 1 sampai 100 gambar.
// Body: {"ids": ["...", "..."]}.
func (h *ImageHandler) Delete(c *gin.Context) {
	var req imagedto.DeleteImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	deletedIDs, err := h.imageService.DeleteMany(c.Request.Context(), req.IDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, fmt.Sprintf("%d gambar berhasil dihapus", len(deletedIDs)),
		imagedto.DeleteImagesResponse{DeletedIDs: deletedIDs})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *ImageHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *imagemodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
		return
	}
	if errors.Is(err, imagemodel.ErrImageNotFound) {
		response.Error(c, http.StatusNotFound, "Gambar tidak ditemukan")
		return
	}

	log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
}

// List menangani GET /api/v1/images: galeri gambar (filter kategori & hierarki mapel, cari nama /
// nama file, urutan, 15 per halaman).
func (h *ImageHandler) List(c *gin.Context) {
	var q imagedto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	items, f, total, err := h.imageService.List(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar gambar", imagedto.ToImageResponses(items),
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// CategoryCounts menangani GET /api/v1/images/category-counts: jumlah gambar per kategori.
func (h *ImageHandler) CategoryCounts(c *gin.Context) {
	var q imagedto.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	counts, err := h.imageService.CategoryCounts(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jumlah gambar per kategori", imagedto.ToCategoryCountsResponse(counts))
}

// Get menangani GET /api/v1/images/:id.
func (h *ImageHandler) Get(c *gin.Context) {
	item, err := h.imageService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail gambar", imagedto.ToImageResponse(item))
}
