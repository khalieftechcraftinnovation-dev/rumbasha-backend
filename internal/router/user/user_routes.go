// Package user (router) berisi daftar URL menu User (panel admin).
//
// Di package lain, import dengan alias userrouter:
//
//	userrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/user"
package user

import (
	"github.com/gin-gonic/gin"

	userhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/user"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /users. Wajib login, hanya admin & super admin.
// Tambah, ubah & hapus: admin hanya akun tutor & siswa (dicek di service).
func Register(private *gin.RouterGroup, h *userhandler.UserHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.GET("/users", adminOnly, h.List)                     // list: filter role/jenjang/kelas/cabang/status, cari
	private.POST("/users", adminOnly, h.Create)                  // tambah 1-100 user sekaligus, body {"users": [...]}
	private.POST("/users/import/check", adminOnly, h.CheckExcel) // cek file Excel tambah user (tidak menyimpan)
	private.GET("/users/role-counts", adminOnly, h.RoleCounts)   // jumlah per role (angka tab)
	private.GET("/users/:id", adminOnly, h.GetDetail)            // detail 1 user
	private.PUT("/users/:id", adminOnly, h.Update)               // ubah 1 user (ganti penuh, role tetap)
	private.DELETE("/users", adminOnly, h.Delete)                // hapus banyak user (soft delete), body {"ids": [...]}
}
