// Package response berisi helper untuk mengirim JSON dengan format yang sama
// di semua endpoint:
//
//	Sukses: {"status_code": 200, "message": "...", "data": {...}, "meta": {...}}
//	Gagal:  {"status_code": 422, "message": "...", "data": null, "errors": [{"field": "...", "message": "..."}]}
//
// status_code selalu sama dengan HTTP status code response-nya.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body adalah bentuk JSON semua response.
// "data" selalu ada (null kalau tidak ada isinya) supaya frontend bisa membacanya tanpa cek dulu.
// omitempty = field tidak ditampilkan di JSON kalau nilainya kosong (dipakai untuk meta & errors).
type Body struct {
	StatusCode int          `json:"status_code"`
	Message    string       `json:"message"`
	Data       any          `json:"data"`
	Meta       *Meta        `json:"meta,omitempty"`
	Errors     []FieldError `json:"errors,omitempty"`
}

// Meta berisi info pagination untuk endpoint list.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// FieldError menjelaskan kesalahan pada satu field request.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Success mengirim response sukses.
func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Body{StatusCode: status, Message: message, Data: data})
}

// SuccessWithMeta mengirim response sukses beserta info pagination.
func SuccessWithMeta(c *gin.Context, message string, data any, meta Meta) {
	c.JSON(http.StatusOK, Body{StatusCode: http.StatusOK, Message: message, Data: data, Meta: &meta})
}

// Error mengirim response gagal.
// AbortWithStatusJSON juga menghentikan middleware/handler berikutnya.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Body{StatusCode: status, Message: message})
}

// ErrorWithFields mengirim response gagal beserta daftar error per field.
// Dipakai untuk error validasi, supaya frontend bisa menampilkan pesan di bawah input yang salah.
func ErrorWithFields(c *gin.Context, status int, message string, fields []FieldError) {
	c.AbortWithStatusJSON(status, Body{StatusCode: status, Message: message, Errors: fields})
}
