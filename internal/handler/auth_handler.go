package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service"
)

// AuthHandler menangani semua endpoint /auth.
type AuthHandler struct {
	authService         *service.AuthService
	subscriptionService *service.SubscriptionService
}

// NewAuthHandler membuat AuthHandler. Dipanggil sekali di main.go.
func NewAuthHandler(authService *service.AuthService, subscriptionService *service.SubscriptionService) *AuthHandler {
	return &AuthHandler{authService: authService, subscriptionService: subscriptionService}
}

// Login menangani POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	session, err := h.authService.Login(c.Request.Context(), req.Username, req.Password, c.Request.UserAgent())
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Login berhasil", toSessionResponse(session))
}

// Refresh menangani POST /api/v1/auth/refresh: menukar refresh token dengan token baru.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	session, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent())
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Sesi berhasil diperbarui", toSessionResponse(session))
}

// Logout menangani POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// Me menangani GET /api/v1/auth/me: data user yang sedang login.
// Untuk siswa, ikut dikirim paket yang aktif HARI INI dan fitur yang terbuka.
func (h *AuthHandler) Me(c *gin.Context) {
	ctx := c.Request.Context()

	// GetUserID diisi oleh middleware.RequireAuth dari access token.
	user, err := h.authService.GetUser(ctx, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	result := dto.MeResponse{User: dto.ToUserResponse(user)}

	if user.Role == model.RoleStudent {
		access, err := h.subscriptionService.GetTodayAccess(ctx, user.ID)
		if err != nil {
			h.handleError(c, err)
			return
		}
		result.Access = &dto.AccessResponse{
			Subscription: dto.ToSubscriptionResponse(access.Subscription),
			Features:     access.Features,
		}
	}

	response.Success(c, http.StatusOK, "Data user berhasil diambil", result)
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "Username atau kata sandi salah")
	case errors.Is(err, model.ErrUserInactive):
		response.Error(c, http.StatusForbidden, "Akun Anda tidak aktif. Silakan hubungi admin")
	case errors.Is(err, model.ErrSessionExpired), errors.Is(err, model.ErrUserNotFound):
		response.Error(c, http.StatusUnauthorized, "Sesi sudah berakhir. Silakan login kembali")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// toSessionResponse mengubah service.Session menjadi bentuk JSON.
func toSessionResponse(session service.Session) dto.SessionResponse {
	return dto.ToSessionResponse(
		session.AccessToken,
		session.AccessTokenExpiresAt,
		session.RefreshToken,
		session.RefreshTokenExpiresAt,
		session.User,
	)
}
