// Package middleware berisi fungsi yang dijalankan SEBELUM handler,
// misalnya untuk mengecek apakah user sudah login.
package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// Kunci untuk menyimpan data user yang login di gin.Context.
const (
	contextUserID = "auth_user_id"
	contextRole   = "auth_role"
)

// RequireAuth memastikan request membawa access token yang valid di header:
//
//	Authorization: Bearer <access_token>
//
// Kalau valid, user ID dan role disimpan di context supaya bisa dibaca handler
// lewat GetUserID(c) dan GetRole(c). Kalau tidak valid, request dihentikan (401).
func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found || accessToken == "" {
			response.Error(c, http.StatusUnauthorized, "Silakan login terlebih dahulu")
			return
		}

		claims, err := token.ParseAccessToken(accessToken, jwtSecret)
		if err != nil {
			// Alasan aslinya (kedaluwarsa, tanda tangan salah, dsb.) dicatat di log API.
			_ = c.Error(err)
			// Termasuk token kedaluwarsa: client sebaiknya memanggil /auth/refresh.
			response.Error(c, http.StatusUnauthorized, "Sesi tidak valid atau sudah berakhir")
			return
		}

		c.Set(contextUserID, claims.Subject)
		c.Set(contextRole, claims.Role)
		c.Next() // lanjut ke middleware/handler berikutnya
	}
}

// RequireRole hanya mengizinkan role tertentu. Wajib dipasang SETELAH RequireAuth.
//
//	api.POST("/quizzes", middleware.RequireRole(model.RoleAdmin, model.RoleSuperAdmin), h.Create)
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(allowedRoles, GetRole(c)) {
			response.Error(c, http.StatusForbidden, "Anda tidak memiliki akses ke fitur ini")
			return
		}
		c.Next()
	}
}

// GetUserID mengembalikan ID user yang sedang login.
func GetUserID(c *gin.Context) string {
	return c.GetString(contextUserID)
}

// GetRole mengembalikan role user yang sedang login.
func GetRole(c *gin.Context) string {
	return c.GetString(contextRole)
}
