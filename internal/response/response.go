// Package response berisi helper untuk mengirim JSON dengan format yang sama
// di semua endpoint:
//
//	Sukses: {"success": true,  "message": "...", "data": {...}, "meta": {...}}
//	Gagal:  {"success": false, "message": "...", "errors": [{"field": "...", "message": "..."}]}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body adalah bentuk JSON semua response.
// omitempty = field tidak ditampilkan di JSON kalau nilainya kosong.
type Body struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
	Meta    *Meta        `json:"meta,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`
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
	c.JSON(status, Body{Success: true, Message: message, Data: data})
}

// SuccessWithMeta mengirim response sukses beserta info pagination.
func SuccessWithMeta(c *gin.Context, message string, data any, meta Meta) {
	c.JSON(http.StatusOK, Body{Success: true, Message: message, Data: data, Meta: &meta})
}

// Error mengirim response gagal.
// AbortWithStatusJSON juga menghentikan middleware/handler berikutnya.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Body{Success: false, Message: message})
}
