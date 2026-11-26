// Package login (handler) menangani endpoint /auth (login, refresh, logout, me).
// Tugas handler: baca request → panggil service → kirim response.
// Aturan bisnis TIDAK boleh ditulis di sini, tempatnya di service.
//
// Di package lain, import dengan alias loginhandler:
//
//	loginhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/auth/login"
package login

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	logindto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/auth/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/login"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"
)

// LoginHandler menangani semua endpoint /auth.
type LoginHandler struct {
	loginService        *loginservice.LoginService
	subscriptionService *subscriptionservice.SubscriptionService
}

// NewLoginHandler membuat LoginHandler. Dipanggil sekali di main.go.
func NewLoginHandler(loginService *loginservice.LoginService, subscriptionService *subscriptionservice.SubscriptionService) *LoginHandler {
	return &LoginHandler{loginService: loginService, subscriptionService: subscriptionService}
}

// Login menangani POST /api/v1/auth/login.
//
// Catatan Go: (h *LoginHandler) di depan nama fungsi artinya Login adalah
// method milik LoginHandler, mirip $this di PHP.
func (h *LoginHandler) Login(c *gin.Context) {
	var req logindto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	session, err := h.loginService.Login(c.Request.Context(), req.Username, req.Password, c.Request.UserAgent(), req.RememberMe)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Login berhasil", toSessionResponse(session))
}

// Refresh menangani POST /api/v1/auth/refresh: menukar refresh token dengan token baru.
func (h *LoginHandler) Refresh(c *gin.Context) {
	var req logindto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	session, err := h.loginService.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent())
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Sesi berhasil diperbarui", toSessionResponse(session))
}

// Logout menangani POST /api/v1/auth/logout.
func (h *LoginHandler) Logout(c *gin.Context) {
	var req logindto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	if err := h.loginService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// Me menangani GET /api/v1/auth/me: data user yang sedang login.
// Untuk siswa, ikut dikirim paket yang aktif HARI INI dan fitur yang terbuka.
func (h *LoginHandler) Me(c *gin.Context) {
	ctx := c.Request.Context()

	// GetUserID diisi oleh middleware.RequireAuth dari access token.
	user, err := h.loginService.GetUser(ctx, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}

	result := logindto.MeResponse{User: logindto.ToUserResponse(user)}

	if user.Role == loginmodel.RoleStudent {
		access, err := h.subscriptionService.GetTodayAccess(ctx, user.ID)
		if err != nil {
			h.handleError(c, err)
			return
		}
		result.Access = &logindto.AccessResponse{
			Subscription: logindto.ToSubscriptionResponse(access.Subscription),
			Features:     access.Features,
		}
	}

	response.Success(c, http.StatusOK, "Data user berhasil diambil", result)
}

// ChangePassword menangani POST /api/v1/auth/change-password: ubah kata sandi sendiri (semua role).
// Semua sesi lama dicabut; respons berisi sesi baru untuk perangkat ini (simpan token barunya).
func (h *LoginHandler) ChangePassword(c *gin.Context) {
	var req logindto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	session, err := h.loginService.ChangePassword(c.Request.Context(), middleware.GetUserID(c),
		req.CurrentPassword, req.NewPassword, c.Request.UserAgent(), req.RememberMe)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Kata sandi berhasil diubah", toSessionResponse(session))
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
//
// Catatan Go: Go tidak punya try/catch. Error dikembalikan sebagai nilai biasa,
// lalu dicek dengan errors.Is untuk tahu jenis error-nya.
func (h *LoginHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	switch {
	case errors.Is(err, loginmodel.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "Username atau kata sandi salah")
	case errors.Is(err, loginmodel.ErrAccountEnded):
		response.Error(c, http.StatusForbidden, "Masa aktif akun Anda sudah berakhir. Silakan hubungi admin")
	case errors.Is(err, loginmodel.ErrUserInactive):
		response.Error(c, http.StatusForbidden, "Akun Anda tidak aktif. Silakan hubungi admin")
	case errors.Is(err, loginmodel.ErrWrongPassword):
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{{Field: "current_password", Message: "kata sandi lama salah"}})
	case errors.Is(err, loginmodel.ErrSamePassword):
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{{Field: "new_password", Message: "harus berbeda dengan kata sandi lama"}})
	case errors.Is(err, loginmodel.ErrSessionExpired), errors.Is(err, loginmodel.ErrUserNotFound):
		response.Error(c, http.StatusUnauthorized, "Sesi sudah berakhir. Silakan login kembali")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// toSessionResponse mengubah loginservice.Session menjadi bentuk JSON.
func toSessionResponse(session loginservice.Session) logindto.SessionResponse {
	return logindto.ToSessionResponse(
		session.AccessToken,
		session.AccessTokenExpiresAt,
		session.RefreshToken,
		session.RefreshTokenExpiresAt,
		session.RememberMe,
		session.User,
	)
}
