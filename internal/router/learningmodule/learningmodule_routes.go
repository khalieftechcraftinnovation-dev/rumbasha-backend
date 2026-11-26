// Package learningmodule (router) berisi daftar URL modul pembelajaran (materi PDF).
//
// Di package lain, import dengan alias learningmodulerouter:
//
//	learningmodulerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/learningmodule"
package learningmodule

import (
	"github.com/gin-gonic/gin"

	learningmodulehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/learningmodule"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route modul. Wajib login: /learning-modules untuk admin & super admin,
// /student/learning-modules untuk siswa.
// File PDF TIDAK disajikan publik: diunduh lewat GET /learning-modules/:id/file.
func Register(private *gin.RouterGroup, h *learningmodulehandler.LearningModuleHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/learning-modules", adminOnly, h.Create)       // upload 1 PDF + data modul (multipart)
	private.GET("/learning-modules", adminOnly, h.List)          // list: filter letak/paket/status, cari nama
	private.GET("/learning-modules/:id", adminOnly, h.Get)       // detail 1 modul
	private.GET("/learning-modules/:id/file", adminOnly, h.File) // unduh / lihat PDF
	private.PUT("/learning-modules/:id", adminOnly, h.Update)    // ubah (ganti penuh), file opsional = ganti PDF
	private.DELETE("/learning-modules", adminOnly, h.Delete)     // hapus banyak (soft delete), body {"ids": [...]}

	// Siswa: modul di jenjangnya (yang di luar paket tampil terkunci), buka modul, unduh PDF.
	studentOnly := middleware.RequireRole(loginmodel.RoleStudent)
	private.GET("/student/learning-modules", studentOnly, h.StudentList)
	private.POST("/student/learning-modules/:id/open", studentOnly, h.Open)
	private.GET("/student/learning-modules/:id/file", studentOnly, h.StudentFile)
}
