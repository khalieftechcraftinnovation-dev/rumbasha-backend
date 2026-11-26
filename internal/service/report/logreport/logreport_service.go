// Package logreport (service) berisi logika kirim file log API hari ini ke email developer.
// Dipakai admin/super admin untuk melaporkan bug: deskripsi masalah + file log dikirim sekaligus.
//
// Di package lain, import dengan alias logreportservice:
//
//	logreportservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/report/logreport"
package logreport

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/mailer"
	logreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/logreport"
)

const (
	// cooldown = jeda minimal antar pengiriman, supaya admin yang menekan tombol
	// berkali-kali tidak membanjiri inbox (Gmail juga membatasi jumlah email per hari).
	cooldown = time.Minute

	// File di atas compressAbove dikompres (gzip) dulu. Log teks biasanya mengecil ~10x.
	compressAbove = 5 * 1024 * 1024 // 5 MB

	// Batas lampiran Gmail 25 MB, dan lampiran membesar ~33% setelah diubah ke base64.
	maxAttachment = 18 * 1024 * 1024 // 18 MB
)

// MailSender adalah kemampuan mengirim email yang dibutuhkan service ini.
// Di aplikasi diisi *mailer.Mailer; di test diisi pengirim palsu supaya tidak benar-benar kirim email.
//
// Catatan Go: interface = daftar method. Tipe apa pun yang punya method ini
// otomatis memenuhi interface-nya (tidak perlu tulis "implements" seperti di PHP).
type MailSender interface {
	IsConfigured() bool
	Send(msg mailer.Message) error
}

// LogReportService mengirim file log hari ini ke email developer.
type LogReportService struct {
	sender    MailSender
	logDir    string // folder log API (LOG_DIR)
	recipient string // email tujuan (LOG_REPORT_EMAIL)
	appEnv    string // development / staging / production, ditulis di email

	mu         sync.Mutex // menjaga lastSentAt kalau ada beberapa request bersamaan
	lastSentAt time.Time
}

// NewLogReportService membuat LogReportService. Dipanggil sekali di main.go.
func NewLogReportService(sender MailSender, logDir, recipient, appEnv string) *LogReportService {
	return &LogReportService{sender: sender, logDir: logDir, recipient: recipient, appEnv: appEnv}
}

// Reporter adalah admin yang mengirim laporan.
type Reporter struct {
	Username string
	RealName string
	Role     string
}

// Result adalah info lampiran yang terkirim.
type Result struct {
	FileName   string
	FileSize   int
	Compressed bool
}

// SendToday mengirim file log hari ini (WIB) beserta deskripsi bug ke email developer.
func (s *LogReportService) SendToday(reporter Reporter, description, requestID string) (Result, error) {
	if s.recipient == "" || !s.sender.IsConfigured() {
		return Result{}, logreportmodel.ErrNotConfigured
	}

	now := time.Now()

	// Cek & catat waktu kirim di awal, supaya 2 request bersamaan tidak sama-sama mengirim.
	s.mu.Lock()
	if !s.lastSentAt.IsZero() && now.Sub(s.lastSentAt) < cooldown {
		s.mu.Unlock()
		return Result{}, logreportmodel.ErrTooSoon
	}
	previousSentAt := s.lastSentAt
	s.lastSentAt = now
	s.mu.Unlock()

	result, err := s.send(reporter, description, requestID, now)
	if err != nil {
		// Gagal kirim → admin boleh langsung mencoba lagi tanpa menunggu jeda.
		s.mu.Lock()
		s.lastSentAt = previousSentAt
		s.mu.Unlock()
	}
	return result, err
}

// send membaca file log, menyiapkan lampiran, lalu mengirim emailnya.
func (s *LogReportService) send(reporter Reporter, description, requestID string, now time.Time) (Result, error) {
	today := clock.TodayInJakarta(now)
	fileName := "api-" + today + ".txt"

	// File log sedang terus ditulis; yang terkirim adalah isinya sampai detik ini.
	// Catatan: request laporan ini sendiri belum ada di file (baru ditulis setelah response terkirim).
	data, err := os.ReadFile(apilog.FilePath(s.logDir, today))
	if errors.Is(err, os.ErrNotExist) {
		return Result{}, logreportmodel.ErrLogFileNotFound
	}
	if err != nil {
		return Result{}, err
	}

	attachment := mailer.Attachment{
		FileName:    fileName,
		ContentType: "text/plain; charset=utf-8",
		Data:        data,
	}
	compressed := false
	if len(data) > compressAbove {
		gz, err := gzipBytes(data)
		if err != nil {
			return Result{}, err
		}
		attachment = mailer.Attachment{
			FileName:    fileName + ".gz",
			ContentType: "application/gzip",
			Data:        gz,
		}
		compressed = true
	}
	if len(attachment.Data) > maxAttachment {
		return Result{}, logreportmodel.ErrLogTooLarge
	}

	err = s.sender.Send(mailer.Message{
		To:          []string{s.recipient},
		Subject:     fmt.Sprintf("[Rumbasha][%s] Laporan bug dari %s (%s)", s.appEnv, reporter.Username, today),
		Body:        buildBody(reporter, description, requestID, attachment, s.appEnv, now),
		Attachments: []mailer.Attachment{attachment},
	})
	if err != nil {
		// %w ErrSendFailed → handler tahu jenis error-nya; detail asli (err) tetap tercatat di log API.
		return Result{}, fmt.Errorf("%w: %v", logreportmodel.ErrSendFailed, err)
	}

	return Result{
		FileName:   attachment.FileName,
		FileSize:   len(attachment.Data),
		Compressed: compressed,
	}, nil
}

// buildBody menyusun isi email yang mudah dibaca developer.
func buildBody(reporter Reporter, description, requestID string, attachment mailer.Attachment, appEnv string, now time.Time) string {
	if requestID == "" {
		requestID = "-"
	}

	howToRead := "Satu baris = satu request (JSON). Cari request_id di atas dengan Ctrl+F."
	if attachment.ContentType == "application/gzip" {
		howToRead = "File dikompres (gzip) karena besar; ekstrak dulu (7-Zip / gunzip). " + howToRead
	}

	return fmt.Sprintf(`Laporan bug dari admin Rumbasha.

Dilaporkan oleh : %s (%s, %s)
Waktu laporan   : %s WIB
Environment     : %s
Request ID      : %s

Deskripsi masalah:
%s

Lampiran: %s (%d byte)
%s
`,
		reporter.RealName, reporter.Username, reporter.Role,
		now.In(clock.Jakarta).Format("2006-01-02 15:04:05"),
		appEnv,
		requestID,
		description,
		attachment.FileName, len(attachment.Data),
		howToRead,
	)
}

// gzipBytes mengompres data dengan gzip.
func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
