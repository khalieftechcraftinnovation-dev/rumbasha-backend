// Package question (handler) menangani endpoint bank soal.
//
// Di package lain, import dengan alias questionhandler:
//
//	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/question"
package question

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	questionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/question"
)

// QuestionHandler menangani endpoint /questions.
type QuestionHandler struct {
	questionService *questionservice.QuestionService
}

// NewQuestionHandler membuat QuestionHandler. Dipanggil sekali di main.go.
func NewQuestionHandler(questionService *questionservice.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: questionService}
}

// Create menangani POST /api/v1/questions: membuat 1 atau banyak soal (maks 100) dengan 1 tipe.
// Hanya admin & super admin (diatur di router).
// data = daftar soal yang dibuat, urut sesuai urutan di request.
func (h *QuestionHandler) Create(c *gin.Context) {
	var req questiondto.CreateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	questions, err := h.questionService.CreateMany(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	data := make([]questiondto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		data = append(data, questiondto.ToQuestionResponse(q))
	}
	response.Success(c, http.StatusCreated, fmt.Sprintf("%d soal berhasil dibuat", len(questions)), data)
}

// List menangani GET /api/v1/questions: list Bank Soal dengan filter, pencarian & pagination.
// Urut terbaru dulu. Query string lihat questiondto.ListQuestionsQuery.
func (h *QuestionHandler) List(c *gin.Context) {
	var query questiondto.ListQuestionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	items, filter, total, err := h.questionService.List(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Data soal berhasil diambil", questiondto.ToQuestionListResponses(items), response.Meta{
		Page:    filter.Page,
		PerPage: filter.PerPage,
		Total:   total,
	})
}

// CountByType menangani GET /api/v1/questions/type-counts: jumlah soal per tipe untuk tab Bank Soal.
// Query string sama dengan List (search, difficulty, hierarki); type & page/per_page diabaikan,
// jadi FE bisa mengirim query string yang sama persis dengan request list.
func (h *QuestionHandler) CountByType(c *gin.Context) {
	var query questiondto.ListQuestionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	counts, err := h.questionService.CountByType(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Jumlah soal per tipe", questiondto.ToTypeCountsResponse(counts))
}

// GetDetail menangani GET /api/v1/questions/:id: detail 1 soal termasuk kunci jawaban & pembahasan.
// Hanya admin & super admin (diatur di router), karena berisi kunci jawaban.
func (h *QuestionHandler) GetDetail(c *gin.Context) {
	detail, err := h.questionService.GetDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Detail soal", questiondto.ToQuestionDetailResponse(detail))
}

// Update menangani PUT /api/v1/questions/:id: ganti seluruh isi 1 soal.
// Body = 1 soal (bentuk sama dengan item create) + "type" wajib. Response = detail terbaru.
func (h *QuestionHandler) Update(c *gin.Context) {
	var req questiondto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	detail, err := h.questionService.Update(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Soal berhasil diubah", questiondto.ToQuestionDetailResponse(detail))
}

// Delete menangani DELETE /api/v1/questions: soft delete 1 sampai 100 soal sekaligus.
// Body: {"ids": ["...", "..."]}.
func (h *QuestionHandler) Delete(c *gin.Context) {
	var req questiondto.DeleteQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	deletedIDs, err := h.questionService.DeleteMany(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, fmt.Sprintf("%d soal berhasil dihapus", len(deletedIDs)),
		questiondto.DeleteQuestionsResponse{DeletedIDs: deletedIDs})
}

// msgValidationFailed adalah pesan umum response 422.
const msgValidationFailed = "Validasi gagal"

// maxExcelFileSize adalah ukuran maksimal file Excel upload soal (5 MB).
const maxExcelFileSize = 5 << 20

// CheckExcel menangani POST /api/v1/questions/import/check (multipart, field "file").
// Hanya MENGECEK isi file Excel, tidak menyimpan. Kalau valid, data berisi soal-soal
// yang siap dikirim ke POST /api/v1/questions. Kalau ada error, 422 berisi SEMUA error
// lengkap dengan baris & kolom Excel-nya.
func (h *QuestionHandler) CheckExcel(c *gin.Context) {
	// Batasi ukuran body sebelum dibaca (sedikit di atas batas file untuk header multipart).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxExcelFileSize+(1<<20))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(err)
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, msgValidationFailed, []response.FieldError{
			{Field: "file", Message: "wajib diisi (file .xlsx, maksimal 5 MB)"},
		})
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xlsx") || fileHeader.Size > maxExcelFileSize {
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, msgValidationFailed, []response.FieldError{
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

	questionType, items, err := h.questionService.CheckExcel(c.Request.Context(), file)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, fmt.Sprintf("File valid: %d soal siap disimpan", len(items)),
		questiondto.ToImportCheckResponse(questionType, items))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *QuestionHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	// Kesalahan isi soal → 422 beserta daftar field yang salah.
	var validationErr *questionmodel.ValidationError
	if errors.As(err, &validationErr) {
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			// Row & Column hanya terisi untuk error dari file Excel.
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message, Row: f.Row, Column: f.Column})
		}
		message := validationErr.Message
		if message == "" {
			message = msgValidationFailed
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, message, fields)
		return
	}

	switch {
	case errors.Is(err, questionmodel.ErrQuestionNotFound):
		response.Error(c, http.StatusNotFound, "Soal tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
