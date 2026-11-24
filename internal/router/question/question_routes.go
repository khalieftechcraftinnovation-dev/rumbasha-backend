// Package question (router) berisi daftar URL untuk fitur bank soal.
//
// Di package lain, import dengan alias questionrouter:
//
//	questionrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/question"
package question

import (
	"github.com/gin-gonic/gin"

	questionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/question"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /questions. Wajib login, hanya admin & super admin
// (detail soal berisi kunci jawaban, jadi siswa & tutor tidak boleh mengakses).
func Register(private *gin.RouterGroup, h *questionhandler.QuestionHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.POST("/questions", adminOnly, h.Create)
	private.GET("/questions", adminOnly, h.List)                    // list Bank Soal: filter, search, pagination
	private.GET("/questions/type-counts", adminOnly, h.CountByType) // jumlah soal per tipe (tab Bank Soal)
	private.GET("/questions/:id", adminOnly, h.GetDetail)
	private.PUT("/questions/:id", adminOnly, h.Update)               // ubah 1 soal (ganti penuh)
	private.DELETE("/questions", adminOnly, h.Delete)                // hapus banyak soal (soft delete), body {"ids": [...]}
	private.POST("/questions/import/check", adminOnly, h.CheckExcel) // cek file Excel (tidak menyimpan)
}
