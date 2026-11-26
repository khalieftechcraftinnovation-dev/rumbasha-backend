// Package landingpage (router) berisi daftar URL fitur Landing Page.
//
// Di package lain, import dengan alias landingpagerouter:
//
//	landingpagerouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/site/landingpage"
package landingpage

import (
	"github.com/gin-gonic/gin"

	landingpagehandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/site/landingpage"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

// Register mendaftarkan route Landing Page.
//   - public : GET /public/landing-pages/:page (situs publik, tanpa login)
//   - private: /landing-pages/... (panel admin, hanya admin & super admin)
//
// :page = home (Beranda), pricing (Paket & Biaya), achievers (Siswa Berprestasi).
func Register(public, private *gin.RouterGroup, h *landingpagehandler.LandingPageHandler) {
	public.GET("/public/landing-pages/:page", h.GetPublic)

	adminOnly := middleware.RequireRole(loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin)
	private.GET("/landing-pages", adminOnly, h.ListPages)
	private.GET("/landing-pages/:page", adminOnly, h.GetDraft)
	// "order" didaftarkan sebelum ":key" bukan masalah di Gin: path statis selalu didahulukan.
	private.PUT("/landing-pages/:page/sections/order", adminOnly, h.Reorder)
	private.PUT("/landing-pages/:page/sections/:key", adminOnly, h.UpdateContent)
	private.PATCH("/landing-pages/:page/sections/:key/visibility", adminOnly, h.SetVisible)
	private.POST("/landing-pages/:page/publish", adminOnly, h.Publish)
}
