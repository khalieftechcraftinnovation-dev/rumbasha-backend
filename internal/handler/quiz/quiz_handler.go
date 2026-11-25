// Package quiz (handler) menangani endpoint quiz.
//
// Di package lain, import dengan alias quizhandler:
//
//	quizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz"
package quiz

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
	quizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
)

// QuizHandler menangani endpoint /quizzes.
type QuizHandler struct {
	quizService *quizservice.QuizService
}

// NewQuizHandler membuat QuizHandler. Dipanggil sekali di main.go.
func NewQuizHandler(quizService *quizservice.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

// Create menangani POST /api/v1/quizzes: membuat 1 quiz (Uji Kom / Tryout) dengan soal yang dipilih.
// Quiz baru berstatus draft. data = detail quiz (letak dengan nama + daftar soal urut nomor).
func (h *QuizHandler) Create(c *gin.Context) {
	var req quizdto.CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	detail, err := h.quizService.Create(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "Quiz berhasil dibuat", quizdto.ToQuizDetailResponse(detail, time.Now()))
}

// Update menangani PUT /api/v1/quizzes/:id: ganti seluruh isi 1 quiz (body sama dengan create).
// Hanya quiz yang belum dimulai. Kode akses & status tidak berubah. data = detail terbaru.
func (h *QuizHandler) Update(c *gin.Context) {
	var req quizdto.CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	detail, err := h.quizService.Update(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Quiz berhasil diubah", quizdto.ToQuizDetailResponse(detail, time.Now()))
}

// GetDetail menangani GET /api/v1/quizzes/:id: detail 1 quiz (letak dengan nama, jadwal,
// schedule_status, kode akses, daftar soal urut nomor). Tidak ada / sudah dihapus → 404.
func (h *QuizHandler) GetDetail(c *gin.Context) {
	detail, err := h.quizService.GetDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Detail quiz", quizdto.ToQuizDetailResponse(detail, time.Now()))
}

// Delete menangani DELETE /api/v1/quizzes: hapus 1 sampai 100 quiz (soft delete).
// Body: {"ids": ["...", "..."]}. Semua atau tidak sama sekali; quiz yang sedang berjalan ditolak.
func (h *QuizHandler) Delete(c *gin.Context) {
	var req quizdto.DeleteQuizzesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	deletedIDs, err := h.quizService.DeleteMany(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, fmt.Sprintf("%d quiz berhasil dihapus", len(deletedIDs)),
		quizdto.DeleteQuizzesResponse{DeletedIDs: deletedIDs})
}

// Announcements menangani GET /api/v1/student/quiz-announcements (khusus siswa): pengumuman
// Uji Kom & Tryout yang sudah publish dan belum selesai untuk kelas siswa, urut paling dekat dimulai.
// Tanpa kode akses.
func (h *QuizHandler) Announcements(c *gin.Context) {
	var query quizdto.AnnouncementsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	items, total, page, perPage, now, err := h.quizService.ListAnnouncements(c.Request.Context(), middleware.GetUserID(c), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Pengumuman quiz", quizdto.ToAnnouncementResponses(items, now), response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

// Open menangani POST /api/v1/quizzes/:id/open (khusus siswa): buka Uji Kom / Tryout yang
// sedang berjalan. Body {"access_code": "K7Q2ZD"} wajib saat pertama kali; membuka ulang boleh
// tanpa body. data = quiz + waktu mengerjakan + soal lengkap dengan kunci jawaban (tanpa pembahasan).
func (h *QuizHandler) Open(c *gin.Context) {
	var req quizdto.OpenQuizRequest
	// Body boleh kosong (membuka ulang tanpa kode akses).
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.ValidationError(c, err)
		return
	}

	opened, review, err := h.quizService.Open(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), req.AccessCode)
	if err != nil {
		h.handleError(c, err)
		return
	}
	if review != nil { // sudah pernah submit: langsung review (state = "submitted")
		response.Success(c, http.StatusOK, "Quiz sudah dikerjakan, menampilkan review", quizdto.ToQuizReviewResponse(*review, time.Now()))
		return
	}
	response.Success(c, http.StatusOK, "Quiz dibuka", quizdto.ToOpenedQuizResponse(opened, time.Now()))
}

// Submit menangani POST /api/v1/quizzes/:id/submit (khusus siswa): kumpulkan jawaban Uji Kom /
// Tryout yang sudah dibuka. Body: {"answers": [{"question_id", "answer", "is_correct"}]} (dinilai FE).
// Hanya sekali. data = nilai + review (soal, kunci jawaban, pembahasan, jawaban siswa).
func (h *QuizHandler) Submit(c *gin.Context) {
	var req questiondto.SubmitAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	review, err := h.quizService.Submit(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), questiondto.ToSubmittedAnswers(req))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jawaban berhasil dikumpulkan", quizdto.ToQuizReviewResponse(review, time.Now()))
}

// Review menangani GET /api/v1/quizzes/:id/review (khusus siswa): hasil & review quiz yang sudah
// di-submit (bentuk sama dengan response submit). Belum submit → 409.
func (h *QuizHandler) Review(c *gin.Context) {
	review, err := h.quizService.Review(c.Request.Context(), c.Param("id"), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Review quiz", quizdto.ToQuizReviewResponse(review, time.Now()))
}

// AccessCode menangani GET /api/v1/quizzes/:id/access-code (tutor, admin, super admin):
// kode akses yang berlaku sekarang, untuk dibagikan ke siswa saat quiz dimulai.
func (h *QuizHandler) AccessCode(c *gin.Context) {
	detail, err := h.quizService.GetAccessCode(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Kode akses quiz", quizdto.ToAccessCodeResponse(detail, time.Now()))
}

// List menangani GET /api/v1/quizzes: list quiz admin dengan filter, search, pagination.
// Setiap quiz berisi schedule_status: upcoming (akan dimulai), ongoing (berjalan), finished (sudah lewat).
func (h *QuizHandler) List(c *gin.Context) {
	var query quizdto.ListQuizzesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	items, filter, total, now, err := h.quizService.List(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Data quiz berhasil diambil", quizdto.ToQuizResponses(items, now), response.Meta{
		Page:    filter.Page,
		PerPage: filter.PerPage,
		Total:   total,
	})
}

// ScheduleCounts menangani GET /api/v1/quizzes/schedule-counts: jumlah quiz per status jadwal
// (angka tab). Query string sama dengan List; schedule_status & pagination diabaikan.
func (h *QuizHandler) ScheduleCounts(c *gin.Context) {
	var query quizdto.ListQuizzesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	counts, err := h.quizService.CountBySchedule(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Jumlah quiz per status jadwal", quizdto.ToScheduleCountsResponse(counts))
}

// ListBundles menangani GET /api/v1/quiz-bundles: daftar bundle (quiz yang sudah dibuat) untuk
// dipilih saat membuat quiz baru. Isi soal 1 bundle: GET /api/v1/questions?quiz_id=<id>.
func (h *QuizHandler) ListBundles(c *gin.Context) {
	var query quizdto.ListBundlesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	bundles, filter, total, err := h.quizService.ListBundles(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.SuccessWithMeta(c, "Data bundle soal", quizdto.ToBundleResponses(bundles), response.Meta{
		Page:    filter.Page,
		PerPage: filter.PerPage,
		Total:   total,
	})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *QuizHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *quizmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, quizmodel.ErrQuizNotFound):
		response.Error(c, http.StatusNotFound, "Quiz tidak ditemukan")
	case errors.As(err, new(*quizmodel.ClosedError)):
		response.Error(c, http.StatusConflict, err.Error()) // quiz belum dimulai / sudah selesai
	case errors.Is(err, quizmodel.ErrQuizStarted):
		response.Error(c, http.StatusConflict, "Quiz sudah dimulai, tidak bisa diubah")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
