// Package subject (router) berisi daftar URL untuk fitur subject (mata pelajaran).
//
// Di package lain, import dengan alias subjectrouter:
//
//	subjectrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/subject"
package subject

import (
	"github.com/gin-gonic/gin"

	subjecthandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subject"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// Register mendaftarkan semua route /subjects. Semuanya wajib login (grup private).
// Semua user boleh melihat, hanya admin & super admin yang boleh mengubah.
func Register(private *gin.RouterGroup, h *subjecthandler.SubjectHandler) {
	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)

	private.GET("/subjects", h.List)
	private.GET("/subjects/:id", h.GetByID)
	private.POST("/subjects", adminOnly, h.Create)
	private.PUT("/subjects/:id", adminOnly, h.Update)
	private.DELETE("/subjects/:id", adminOnly, h.Delete)
}
