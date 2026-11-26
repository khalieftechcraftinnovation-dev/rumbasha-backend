// Package dropdown (router) berisi daftar URL untuk fitur dropdown.
//
// Di package lain, import dengan alias dropdownrouter:
//
//	dropdownrouter "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/router/master/dropdown"
package dropdown

import (
	"github.com/gin-gonic/gin"

	dropdownhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/master/dropdown"
)

// Register mendaftarkan route /dropdowns. Wajib login, semua role boleh
// (siswa juga butuh dropdown, contoh memilih mapel & level Daily Quiz).
func Register(private *gin.RouterGroup, h *dropdownhandler.DropdownHandler) {
	private.GET("/dropdowns/:type", h.List)
}
