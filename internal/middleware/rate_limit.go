package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// RateLimit membatasi jumlah request per IP: maksimal limit request setiap window
// (fixed window, disimpan di memori proses). Dipakai untuk endpoint publik yang MENULIS ke
// database tanpa login (contoh submit Quiz Trial), supaya tidak mudah di-spam.
//
//	public.POST("/public/trial-quiz/submit", middleware.RateLimit(10, time.Minute), h.Submit)
//
// Catatan: hitungan per proses API. Kalau nanti API dijalankan lebih dari 1 server atau di
// belakang reverse proxy, atur SetTrustedProxies supaya ClientIP berisi IP asli pengunjung.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	type counter struct {
		count   int
		resetAt time.Time
	}
	var (
		mu        sync.Mutex
		counters  = map[string]*counter{}
		nextSweep = time.Now().Add(window)
	)

	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()

		mu.Lock()
		// Bersihkan hitungan yang sudah lewat window, supaya map tidak terus membesar.
		if now.After(nextSweep) {
			for key, ct := range counters {
				if now.After(ct.resetAt) {
					delete(counters, key)
				}
			}
			nextSweep = now.Add(window)
		}
		ct, ok := counters[ip]
		if !ok || now.After(ct.resetAt) {
			ct = &counter{resetAt: now.Add(window)}
			counters[ip] = ct
		}
		ct.count++
		allowed := ct.count <= limit
		retryAfter := int(ct.resetAt.Sub(now).Seconds()) + 1
		mu.Unlock()

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			response.Error(c, http.StatusTooManyRequests, "Terlalu banyak percobaan, silakan coba lagi sebentar lagi")
			return
		}
		c.Next()
	}
}
