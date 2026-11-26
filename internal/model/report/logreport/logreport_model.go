// Package logreport (model) berisi daftar error untuk fitur kirim log ke email (laporan bug).
// Fitur ini tidak punya tabel database.
//
// Di package lain, import dengan alias logreportmodel:
//
//	logreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/logreport"
package logreport

import "errors"

// Daftar error fitur kirim log.
var (
	ErrNotConfigured   = errors.New("pengiriman email belum dikonfigurasi (SMTP / LOG_REPORT_EMAIL)")
	ErrLogFileNotFound = errors.New("file log hari ini belum ada")
	ErrLogTooLarge     = errors.New("file log hari ini terlalu besar untuk dikirim lewat email")
	ErrTooSoon         = errors.New("log baru saja dikirim, tunggu sebentar")
	ErrSendFailed      = errors.New("gagal mengirim email")
)
