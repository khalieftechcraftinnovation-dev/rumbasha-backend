package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/apilog"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

// APILog mencatat setiap request ke file log harian (lihat package apilog):
// waktu, endpoint, payload, proses inti (error validasi/service), dan response.
//
// Wajib dipasang PALING AWAL di router, supaya semua request tercatat,
// termasuk yang ditolak middleware lain (belum login, panic, dsb.).
//
// Untuk menambah catatan proses dari handler/service, cukup panggil:
//
//	_ = c.Error(err)
//
// Semua error yang ditambahkan lewat c.Error ikut tercatat di field "process".
func APILog(writer *apilog.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// /health dipanggil berkali-kali oleh monitoring, dan file gambar (/storage/...) dibuka
		// sangat sering oleh aplikasi siswa. Keduanya tidak perlu dicatat.
		if c.Request.URL.Path == "/health" || strings.HasPrefix(c.Request.URL.Path, "/storage/") {
			c.Next()
			return
		}

		start := time.Now()

		// request_id dikirim juga di header response. Kalau ada laporan error dari
		// frontend, minta X-Request-ID-nya, lalu cari di file log.
		requestID := uuid.NewString()
		c.Header("X-Request-ID", requestID)

		payload := readPayload(c)

		// Ganti writer response dengan versi yang ikut menyimpan salinan body-nya.
		recorder := &responseRecorder{ResponseWriter: c.Writer}
		c.Writer = recorder

		c.Next() // jalankan middleware & handler berikutnya

		entry := apilog.Entry{
			Time:       start.In(clock.Jakarta).Format("2006-01-02 15:04:05.000"),
			RequestID:  requestID,
			Method:     c.Request.Method,
			Endpoint:   c.Request.URL.RequestURI(),
			Route:      c.FullPath(),
			Status:     c.Writer.Status(),
			DurationMs: float64(time.Since(start).Microseconds()) / 1000,
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			UserID:     GetUserID(c),
			Role:       GetRole(c),
			Payload:    payload,
			Response:   apilog.BodyForLog(recorder.body.Bytes(), recorder.size),
		}
		for _, e := range c.Errors {
			entry.Process = append(entry.Process, e.Error())
		}

		line, err := entry.Line()
		if err != nil {
			log.Printf("gagal menyusun log API: %v", err)
			return
		}
		writer.Write(line)
	}
}

// readPayload membaca body request untuk dicatat, lalu "mengembalikannya"
// supaya handler tetap bisa membaca body yang sama (body hanya bisa dibaca sekali).
func readPayload(c *gin.Context) json.RawMessage {
	if c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}

	// Upload file tidak dicatat isinya (bisa berukuran besar).
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		return apilog.Text("(upload file, isi tidak dicatat)")
	}

	// Baca maksimal MaxBodySize+1 byte saja. Lebih dari itu berarti body terlalu besar untuk dicatat.
	original := c.Request.Body
	head, err := io.ReadAll(io.LimitReader(original, apilog.MaxBodySize+1))
	if err != nil {
		return apilog.Text("(gagal membaca body: " + err.Error() + ")")
	}

	// Pasang ulang body: bagian yang sudah dibaca + sisa yang belum dibaca.
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(head), original), original}

	totalSize := len(head)
	if totalSize > apilog.MaxBodySize && c.Request.ContentLength > 0 {
		totalSize = int(c.Request.ContentLength)
	}
	return apilog.BodyForLog(head, totalSize)
}

// responseRecorder meneruskan response ke client seperti biasa,
// sambil menyimpan salinannya (maksimal MaxBodySize) untuk dicatat.
//
// Catatan Go: gin.ResponseWriter ditulis tanpa nama field (embedding), jadi semua
// method aslinya tetap ada. Kita hanya "menimpa" Write dan WriteString.
type responseRecorder struct {
	gin.ResponseWriter
	body bytes.Buffer
	size int // ukuran response sebenarnya
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	w.record(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseRecorder) WriteString(data string) (int, error) {
	w.record([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

// record menyimpan salinan response selama ukurannya belum melewati MaxBodySize.
func (w *responseRecorder) record(data []byte) {
	w.size += len(data)
	if w.size <= apilog.MaxBodySize {
		w.body.Write(data)
	}
}
