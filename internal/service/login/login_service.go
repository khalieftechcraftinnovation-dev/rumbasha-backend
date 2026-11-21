// Package login (service) berisi logika login, refresh sesi, dan logout.
// Service tidak tahu soal HTTP (Gin) maupun SQL. Data dibaca/ditulis lewat repository.
//
// Di package lain, import dengan alias loginservice:
//
//	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
package login

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

// LoginService berisi logika login, refresh sesi, dan logout.
type LoginService struct {
	userRepo         *loginrepo.UserRepository
	refreshTokenRepo *loginrepo.RefreshTokenRepository
	jwtSecret        string
	accessTokenTTL   time.Duration
}

// NewLoginService membuat LoginService. Dipanggil sekali di main.go.
func NewLoginService(
	userRepo *loginrepo.UserRepository,
	refreshTokenRepo *loginrepo.RefreshTokenRepository,
	jwtSecret string,
	accessTokenTTL time.Duration,
) *LoginService {
	return &LoginService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
		accessTokenTTL:   accessTokenTTL,
	}
}

// Session adalah hasil login atau refresh: pasangan token + data user.
type Session struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time // = akhir sesi (tanggal 1 bulan berikutnya, 00:00 WIB)
	User                  loginmodel.User
}

// Login mengecek username & password, lalu membuat sesi baru.
func (s *LoginService) Login(ctx context.Context, username, password, userAgent string) (Session, error) {
	user, err := s.userRepo.FindByUsername(ctx, normalizeUsername(username))
	if errors.Is(err, loginmodel.ErrUserNotFound) {
		return Session{}, loginmodel.ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}

	// bcrypt membandingkan password dengan hash-nya. Password asli tidak pernah disimpan.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return Session{}, loginmodel.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return Session{}, loginmodel.ErrUserInactive
	}

	now := time.Now()
	sessionEnd := clock.StartOfNextMonth(now) // semua sesi berakhir tiap awal bulan

	refreshToken, err := token.NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	err = s.refreshTokenRepo.Create(ctx, loginmodel.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: token.HashRefreshToken(refreshToken),
		UserAgent: optionalString(userAgent),
		ExpiresAt: sessionEnd,
		CreatedAt: now,
	})
	if err != nil {
		return Session{}, err
	}

	// Gagal mencatat waktu login bukan alasan untuk menggagalkan login.
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, now)

	return s.newSession(user, refreshToken, sessionEnd, now)
}

// Refresh menukar refresh token lama dengan pasangan token baru.
// Akhir sesi TIDAK diperpanjang: tetap tanggal 1 bulan berikutnya seperti saat login.
func (s *LoginService) Refresh(ctx context.Context, refreshToken, userAgent string) (Session, error) {
	now := time.Now()

	oldToken, err := s.refreshTokenRepo.FindValidByHash(ctx, token.HashRefreshToken(refreshToken), now)
	if err != nil {
		return Session{}, err
	}

	// Cek ulang user: mungkin sudah dinonaktifkan admin sejak login.
	user, err := s.userRepo.FindByID(ctx, oldToken.UserID)
	if errors.Is(err, loginmodel.ErrUserNotFound) {
		return Session{}, loginmodel.ErrSessionExpired
	}
	if err != nil {
		return Session{}, err
	}
	if !user.IsActive() {
		return Session{}, loginmodel.ErrUserInactive
	}

	newRefreshToken, err := token.NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	err = s.refreshTokenRepo.Rotate(ctx, oldToken.ID, loginmodel.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: token.HashRefreshToken(newRefreshToken),
		UserAgent: optionalString(userAgent),
		ExpiresAt: oldToken.ExpiresAt, // akhir sesi tetap sama
		CreatedAt: now,
	}, now)
	if err != nil {
		return Session{}, err
	}

	return s.newSession(user, newRefreshToken, oldToken.ExpiresAt, now)
}

// Logout menutup sesi milik refresh token ini.
func (s *LoginService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokenRepo.RevokeByHash(ctx, token.HashRefreshToken(refreshToken), time.Now())
}

// GetUser mengambil data user yang sedang login.
func (s *LoginService) GetUser(ctx context.Context, userID string) (loginmodel.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// newSession membuat access token, lalu menyusun Session.
func (s *LoginService) newSession(user loginmodel.User, refreshToken string, sessionEnd, now time.Time) (Session, error) {
	// Access token tidak boleh berlaku melewati akhir sesi. Contoh: login
	// 31 Jan 23:55 → access token berakhir 1 Feb 00:00, bukan 00:10.
	accessExpiresAt := clock.Earliest(now.Add(s.accessTokenTTL), sessionEnd)

	accessToken, err := token.NewAccessToken(user.ID, user.Role, s.jwtSecret, accessExpiresAt)
	if err != nil {
		return Session{}, err
	}

	return Session{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: sessionEnd,
		User:                  user,
	}, nil
}

// HashPassword membuat hash bcrypt dari password. Dipakai saat membuat/mengubah user.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// normalizeUsername menyamakan format username dengan aturan di tabel users:
// huruf kecil tanpa spasi di ujung. Jadi login "Budi " tetap cocok dengan "budi".
func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// optionalString mengubah string kosong menjadi nil (NULL di database).
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
