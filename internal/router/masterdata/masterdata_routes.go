// Package masterdata (router) berisi daftar URL menu Master Data & Cabang (panel admin).
//
// Di package lain, import dengan alias masterdatarouter:
//
//	masterdatarouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/masterdata"
package masterdata

import (
	"github.com/gin-gonic/gin"

	masterdatahandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/masterdata"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan route /master-data & /branches. Wajib login, hanya admin & super admin.
// Pilihan dropdown untuk semua user yang login tetap lewat /dropdowns/:type.
func Register(private *gin.RouterGroup, h *masterdatahandler.MasterDataHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.GET("/master-data", adminOnly, h.List)          // tab, filter letak, cari, status
	private.GET("/master-data/counts", adminOnly, h.Counts) // angka tab
	private.GET("/master-data/:id", adminOnly, h.Get)
	private.POST("/master-data", adminOnly, h.Create)
	private.PUT("/master-data/:id", adminOnly, h.Update) // nama, kode, urutan, status
	private.DELETE("/master-data", adminOnly, h.Delete)  // body {"ids": [...]}, soft delete

	private.GET("/branches", adminOnly, h.ListBranches)
	private.GET("/branches/:id", adminOnly, h.GetBranch)
	private.POST("/branches", adminOnly, h.CreateBranch)
	private.PUT("/branches/:id", adminOnly, h.UpdateBranch)
	private.DELETE("/branches", adminOnly, h.DeleteBranches) // body {"ids": [...]}, soft delete
}
