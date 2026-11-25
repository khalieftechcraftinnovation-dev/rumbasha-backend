package apilog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// MaxBodySize = batas ukuran payload/response yang dicatat (10 KB).
// Body yang lebih besar hanya dicatat ukurannya, supaya file log tidak membengkak.
const MaxBodySize = 10 * 1024

// Entry adalah isi satu baris log (satu request).
// Urutan field di sini = urutan yang tampil di file log.
type Entry struct {
	Time       string          `json:"time"`       // waktu request masuk (WIB)
	RequestID  string          `json:"request_id"` // sama dengan header X-Request-ID di response
	Method     string          `json:"method"`
	Endpoint   string          `json:"endpoint"`        // URL yang dipanggil, termasuk query string
	Route      string          `json:"route,omitempty"` // pola route, contoh /api/v1/quizzes/:id
	Status     int             `json:"status"`          // HTTP status code response
	DurationMs float64         `json:"duration_ms"`     // lama proses dalam milidetik
	ClientIP   string          `json:"client_ip"`
	UserAgent  string          `json:"user_agent,omitempty"`
	UserID     string          `json:"user_id,omitempty"` // terisi kalau request membawa token yang valid
	Role       string          `json:"role,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`  // body request (password & token disamarkan)
	Process    []string        `json:"process,omitempty"`  // catatan proses inti: error validasi, error service, dsb.
	Response   json.RawMessage `json:"response,omitempty"` // body response (token disamarkan)
}

// Line mengubah Entry menjadi satu baris JSON yang diakhiri enter.
func (e Entry) Line() ([]byte, error) {
	line, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return append(line, '\n'), nil
}

// sensitiveKeys = field JSON yang isinya tidak boleh tersimpan di log.
// Tambahkan di sini kalau ada field rahasia baru (huruf kecil semua).
var sensitiveKeys = map[string]bool{
	"password":              true,
	"password_hash":         true,
	"old_password":          true,
	"new_password":          true,
	"password_confirmation": true,
	"access_token":          true,
	"refresh_token":         true,
	"token":                 true,
}

// BodyForLog menyiapkan body request/response untuk dicatat:
//   - kosong                  → tidak dicatat
//   - lebih dari MaxBodySize  → hanya dicatat ukurannya
//   - JSON                    → dicatat apa adanya, field rahasia diganti "***"
//   - bukan JSON (rusak, dsb) → dicatat sebagai teks biasa, KECUALI kalau
//     mungkin berisi password/token (tidak bisa disamarkan per field)
//
// totalSize = ukuran body sebenarnya (body yang diterima bisa sudah dipotong).
func BodyForLog(body []byte, totalSize int) json.RawMessage {
	if totalSize == 0 {
		return nil
	}
	if totalSize > MaxBodySize {
		return Text(fmt.Sprintf("(body %d byte, terlalu besar untuk dicatat)", totalSize))
	}

	// Cek cepat dulu: masking (parse ulang JSON) hanya dilakukan kalau
	// body mungkin berisi field rahasia. Kebanyakan request tidak, jadi lebih hemat.
	lower := bytes.ToLower(body)
	mayHaveSecret := bytes.Contains(lower, []byte("password")) || bytes.Contains(lower, []byte("token"))

	if !json.Valid(body) {
		if mayHaveSecret {
			return Text("(body bukan JSON valid dan mungkin berisi password/token, tidak dicatat)")
		}
		return Text(string(body))
	}
	if !mayHaveSecret {
		return body
	}
	return maskJSON(body)
}

// maskJSON mengganti nilai field rahasia di JSON (termasuk yang bersarang) dengan "***".
func maskJSON(body []byte) json.RawMessage {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber() // angka tetap apa adanya (tidak berubah jadi 1e+06 dsb.)

	var data any
	if err := decoder.Decode(&data); err != nil {
		return Text("(body tidak bisa dibaca untuk disamarkan)")
	}

	masked, err := json.Marshal(maskValue(data))
	if err != nil {
		return Text("(body tidak bisa dibaca untuk disamarkan)")
	}
	return masked
}

// maskValue menelusuri isi JSON: object (map) dan array diperiksa sampai ke dalam.
func maskValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if sensitiveKeys[strings.ToLower(key)] {
				v[key] = "***"
			} else {
				v[key] = maskValue(item)
			}
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = maskValue(item)
		}
		return v
	default:
		return v
	}
}

// Text mengubah teks biasa menjadi string JSON, contoh: abc → "abc".
func Text(text string) json.RawMessage {
	encoded, _ := json.Marshal(text)
	return encoded
}
