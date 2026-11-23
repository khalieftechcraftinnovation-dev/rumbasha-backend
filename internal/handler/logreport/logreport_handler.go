// Package logreport (handler) menangani endpoint kirim log ke email (laporan bug dari admin).
//
// Di package lain, import dengan alias logreporthandler:
//
//	logreporthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/logreport"
package logreport

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	logreportdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	logreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/logreport"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/logreport"
)

// LogReportHandler menangani endpoint /log-reports.
type LogReportHandler struct {
	logReportService *logreportservice.LogReportService
	loginService     *loginservice.LoginService // untuk mengambil nama & username pelapor
}

// NewLogReportHandler membuat LogReportHandler. Dipanggil sekali di main.go.
func NewLogReportHandler(logReportService *logreportservice.LogReportService, loginService *loginservice.LoginService) *LogReportHandler {
	return &LogReportHandler{logReportService: logReportService, loginService: loginService}
}

// Send menangani POST /api/v1/log-reports: kirim log hari ini + deskripsi bug ke email developer.
// Hanya admin & super admin (diatur di router).
func (h *LogReportHandler) Send(c *gin.Context) {
	var req logreportdto.SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.loginService.GetUser(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	result, err := h.logReportService.SendToday(logreportservice.Reporter{
		Username: user.Username,
		RealName: user.RealName,
		Role:     user.Role,
	}, req.Description, req.RequestID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Log hari ini berhasil dikirim ke tim developer", logreportdto.SendResponse{
		FileName:   result.FileName,
		FileSize:   result.FileSize,
		Compressed: result.Compressed,
	})
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *LogReportHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	switch {
	case errors.Is(err, loginmodel.ErrUserNotFound):
		response.Error(c, http.StatusUnauthorized, "Sesi sudah berakhir. Silakan login kembali")
	case errors.Is(err, logreportmodel.ErrNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, "Fitur kirim log belum diaktifkan. Hubungi tim developer")
	case errors.Is(err, logreportmodel.ErrLogFileNotFound):
		response.Error(c, http.StatusNotFound, "Belum ada log untuk hari ini")
	case errors.Is(err, logreportmodel.ErrLogTooLarge):
		response.Error(c, http.StatusUnprocessableEntity, "File log hari ini terlalu besar untuk dikirim lewat email. Hubungi tim developer")
	case errors.Is(err, logreportmodel.ErrTooSoon):
		response.Error(c, http.StatusTooManyRequests, "Log baru saja dikirim. Tunggu 1 menit sebelum mengirim lagi")
	case errors.Is(err, logreportmodel.ErrSendFailed):
		log.Printf("gagal kirim log ke email: %v", err)
		response.Error(c, http.StatusBadGateway, "Gagal mengirim email. Coba lagi beberapa saat lagi")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
