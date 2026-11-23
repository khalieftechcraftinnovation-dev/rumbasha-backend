// Package apilog mencatat setiap request API ke file .txt, satu file per hari:
//
//	logs/api-2026-11-23.txt
//
// Setiap baris = satu request dalam format JSON, jadi mudah dicari dengan
// Ctrl+F / grep, misalnya berdasarkan request_id. File yang lebih lama dari
// batas hari (LOG_RETENTION_DAYS, default 15) dihapus otomatis.
package apilog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

// queueSize = jumlah baris log yang boleh mengantre sebelum ditulis ke file.
const queueSize = 1024

// Writer menulis log ke file harian.
//
// Supaya API tetap cepat, request TIDAK menunggu file selesai ditulis:
// baris log dimasukkan ke antrean (channel), lalu satu goroutine di background
// yang menulisnya ke file satu per satu.
//
// Catatan Go: goroutine = fungsi yang berjalan bersamaan di background (mirip thread ringan).
// Channel = antrean yang aman dipakai bersama oleh banyak goroutine.
type Writer struct {
	dir           string
	retentionDays int
	queue         chan []byte
	done          chan struct{} // ditutup saat goroutine penulis sudah selesai
	closeOnce     sync.Once     // memastikan Close() aman dipanggil lebih dari sekali

	// Dua field ini hanya dipakai oleh goroutine penulis, jadi tidak perlu mutex.
	file     *os.File
	fileDate string // tanggal file yang sedang terbuka, contoh "2026-11-23"
}

// NewWriter membuat folder log (kalau belum ada), menghapus file lama,
// lalu menjalankan goroutine penulis. Panggil Close() saat aplikasi berhenti.
func NewWriter(dir string, retentionDays int) (*Writer, error) {
	if retentionDays < 1 {
		return nil, fmt.Errorf("LOG_RETENTION_DAYS minimal 1, tapi isinya %d", retentionDays)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("gagal membuat folder log %s: %w", dir, err)
	}

	w := &Writer{
		dir:           dir,
		retentionDays: retentionDays,
		queue:         make(chan []byte, queueSize),
		done:          make(chan struct{}),
	}
	w.deleteOldFiles(time.Now())

	go w.run() // "go" = jalankan di background
	return w, nil
}

// Write memasukkan satu baris log ke antrean, lalu langsung kembali (tidak menunggu disk).
//
// Kalau antrean penuh (server sangat sibuk atau disk lambat), baris ini dibuang
// dan dicetak peringatan di terminal. Lebih baik kehilangan log daripada API ikut lambat.
func (w *Writer) Write(line []byte) {
	select {
	case w.queue <- line:
	default:
		log.Printf("peringatan: antrean log API penuh, 1 log dibuang")
	}
}

// Close menunggu semua log di antrean selesai ditulis, lalu menutup file.
// Dipanggil saat aplikasi berhenti (setelah server tidak menerima request lagi).
// Write tidak boleh dipanggil lagi setelah Close.
func (w *Writer) Close() {
	w.closeOnce.Do(func() {
		close(w.queue)
	})
	<-w.done // tunggu goroutine penulis selesai
}

// run adalah goroutine penulis: mengambil baris dari antrean dan menulisnya ke file.
// Loop berhenti saat antrean ditutup oleh Close().
func (w *Writer) run() {
	defer close(w.done)

	for line := range w.queue {
		if err := w.writeLine(line); err != nil {
			log.Printf("gagal menulis log API: %v", err)
		}
	}

	if w.file != nil {
		_ = w.file.Close()
	}
}

// writeLine menulis satu baris ke file hari ini. Kalau tanggal sudah berganti,
// file kemarin ditutup, file hari ini dibuka, dan file lama dihapus.
func (w *Writer) writeLine(line []byte) error {
	now := time.Now()
	today := clock.TodayInJakarta(now)

	if w.file == nil || today != w.fileDate {
		if w.file != nil {
			_ = w.file.Close()
		}

		// O_APPEND: tulis di akhir file, jadi restart server tidak menimpa log hari ini.
		file, err := os.OpenFile(w.filePath(today), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			w.file = nil
			return err
		}
		w.file = file
		w.fileDate = today
		w.deleteOldFiles(now)
	}

	_, err := w.file.Write(line)
	return err
}

// filePath menghasilkan lokasi file log untuk tanggal tertentu di folder writer ini.
func (w *Writer) filePath(date string) string {
	return FilePath(w.dir, date)
}

// FilePath menghasilkan lokasi file log untuk tanggal tertentu, contoh logs/api-2026-11-23.txt.
// Dipakai juga oleh fitur lain yang perlu membaca file log (contoh: kirim log ke email).
func FilePath(dir, date string) string {
	return filepath.Join(dir, "api-"+date+".txt")
}

// deleteOldFiles menghapus file log yang lebih lama dari retentionDays.
// Contoh retentionDays = 15 dan hari ini 23 Nov: file 9-23 Nov disimpan, 8 Nov ke bawah dihapus.
func (w *Writer) deleteOldFiles(now time.Time) {
	today, _ := time.ParseInLocation(time.DateOnly, clock.TodayInJakarta(now), clock.Jakarta)
	oldestKept := today.AddDate(0, 0, -(w.retentionDays - 1))

	files, err := os.ReadDir(w.dir)
	if err != nil {
		log.Printf("gagal membaca folder log %s: %v", w.dir, err)
		return
	}

	for _, f := range files {
		// Hanya file dengan pola api-YYYY-MM-DD.txt yang disentuh. File lain dibiarkan.
		name := f.Name()
		if f.IsDir() || !strings.HasPrefix(name, "api-") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		dateText := strings.TrimSuffix(strings.TrimPrefix(name, "api-"), ".txt")
		date, err := time.ParseInLocation(time.DateOnly, dateText, clock.Jakarta)
		if err != nil {
			continue
		}

		if date.Before(oldestKept) {
			if err := os.Remove(filepath.Join(w.dir, name)); err != nil {
				log.Printf("gagal menghapus log lama %s: %v", name, err)
			}
		}
	}
}
