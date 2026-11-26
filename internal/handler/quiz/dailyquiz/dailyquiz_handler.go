// Package dailyquiz (handler) menangani endpoint Daily Quiz (khusus siswa).
//
// Di package lain, import dengan alias dailyquizhandler:
//
//	dailyquizhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/quiz/dailyquiz"
package dailyquiz

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	dailyquizdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/dailyquiz"
	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/dailyquiz"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/dailyquiz"
)

// DailyQuizHandler menangani endpoint /daily-quizzes.
type DailyQuizHandler struct {
	service *dailyquizservice.DailyQuizService
}

// NewDailyQuizHandler membuat DailyQuizHandler. Dipanggil sekali di main.go.
func NewDailyQuizHandler(service *dailyquizservice.DailyQuizService) *DailyQuizHandler {
	return &DailyQuizHandler{service: service}
}

// GradeOptions menangani GET /api/v1/daily-quizzes/grade-options: dropdown kelas Daily Quiz.
// Hanya kelas di jenjang siswa (SD: Kelas 1-3 atau Kelas 4-6); kelas siswa sendiri is_default = true.
func (h *DailyQuizHandler) GradeOptions(c *gin.Context) {
	options, _, err := h.service.GradeOptions(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pilihan kelas Daily Quiz", dailyquizdto.ToGradeOptionResponses(options))
}

// Generate menangani POST /api/v1/daily-quizzes: buat Daily Quiz dari soal acak sesuai pilihan
// siswa. data = Daily Quiz + soal untuk dikerjakan (tanpa kunci jawaban, tanpa kode akses & timer).
func (h *DailyQuizHandler) Generate(c *gin.Context) {
	var req dailyquizdto.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	detail, err := h.service.Generate(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Daily Quiz berhasil dibuat", dailyquizdto.ToDailyQuizResponse(detail))
}

// Open menangani POST /api/v1/daily-quizzes/:id/open: buka Daily Quiz milik siswa.
// Jatah gratis per hari habis → potong coin; coin kurang → 402. data = soal + kunci jawaban
// (tanpa pembahasan) & pemakaian jatah/coin.
func (h *DailyQuizHandler) Open(c *gin.Context) {
	opened, review, err := h.service.Open(c.Request.Context(), c.Param("id"), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	if review != nil { // sudah pernah submit: langsung review (state = "submitted")
		response.Success(c, http.StatusOK, "Daily Quiz sudah dikerjakan, menampilkan review", dailyquizdto.ToDailyQuizReviewResponse(*review))
		return
	}
	response.Success(c, http.StatusOK, "Daily Quiz dibuka", dailyquizdto.ToOpenedDailyQuizResponse(opened))
}

// Submit menangani POST /api/v1/daily-quizzes/:id/submit: kumpulkan jawaban Daily Quiz yang sudah
// dibuka. Body: {"answers": [{"question_id", "answer", "is_correct"}]} (dinilai FE). Hanya sekali.
// data = nilai + review (soal, kunci jawaban, pembahasan, jawaban siswa).
func (h *DailyQuizHandler) Submit(c *gin.Context) {
	var req questiondto.SubmitAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	review, err := h.service.Submit(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), questiondto.ToSubmittedAnswers(req))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jawaban berhasil dikumpulkan", dailyquizdto.ToDailyQuizReviewResponse(review))
}

// Review menangani GET /api/v1/daily-quizzes/:id/review: hasil & review Daily Quiz yang sudah di-submit.
func (h *DailyQuizHandler) Review(c *gin.Context) {
	review, err := h.service.Review(c.Request.Context(), c.Param("id"), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Review Daily Quiz", dailyquizdto.ToDailyQuizReviewResponse(review))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *DailyQuizHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *dailyquizmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.As(err, new(*dailyquizmodel.StateError)):
		response.Error(c, http.StatusConflict, err.Error()) // belum dibuka / sudah submit / belum submit
	case errors.Is(err, dailyquizmodel.ErrDailyQuizNotFound):
		response.Error(c, http.StatusNotFound, "Daily Quiz tidak ditemukan")
	case errors.As(err, new(*dailyquizmodel.InsufficientCoinError)):
		response.Error(c, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, dailyquizmodel.ErrStudentGradeMissing):
		response.Error(c, http.StatusUnprocessableEntity, "Kelas kamu belum diisi, hubungi admin")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// GetSettings menangani GET /api/v1/daily-quiz-settings: pengaturan global Daily Quiz
// (point per level & hadiah juara bulanan).
func (h *DailyQuizHandler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengaturan Daily Quiz", dailyquizdto.ToSettingsResponse(settings))
}

// UpdateSettings menangani PUT /api/v1/daily-quiz-settings (admin & super admin). Berlaku untuk
// seluruh Daily Quiz.
func (h *DailyQuizHandler) UpdateSettings(c *gin.Context) {
	var req dailyquizdto.SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	settings, err := h.service.UpdateSettings(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengaturan Daily Quiz berhasil disimpan", dailyquizdto.ToSettingsResponse(settings))
}

// MonthlyPrizes menangani GET /api/v1/daily-quiz-prizes: penerima hadiah juara Daily Quiz 1 bulan
// (per kelas), untuk admin & super admin.
func (h *DailyQuizHandler) MonthlyPrizes(c *gin.Context) {
	var q dailyquizdto.MonthlyPrizesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	month, period, awards, err := h.service.MonthlyPrizes(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Hadiah juara Daily Quiz", dailyquizdto.ToMonthlyPrizesResponse(month, period, awards))
}
