// Package login (service) berisi logika login, refresh sesi, dan logout.
// Service tidak tahu soal HTTP (Gin) maupun SQL. Data dibaca/ditulis lewat repository.
//
// Di package lain, import dengan alias loginservice:
//
//	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/auth/login"
package login

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/auth/login"
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
	RefreshTokenExpiresAt time.Time // = akhir sesi (tanggal 1 bulan berikutnya 00:00 WIB, atau +30 hari kalau "Ingat saya")
	RememberMe            bool
	User                  loginmodel.UserDetail // user + jenjang & kelas
}

// Login mengecek username & password, lalu membuat sesi baru.
//
// Akhir sesi: tanpa "Ingat saya" = tanggal 1 bulan berikutnya 00:00 WIB (aturan paket offline
// bulanan); dengan "Ingat saya" = 30 hari sejak login (boleh melewati tanggal 1). Refresh tidak
// pernah memperpanjang akhir sesi. Akun nonaktif / masa aktif akun berakhir tetap dicek saat refresh.
func (s *LoginService) Login(ctx context.Context, username, password, userAgent string, rememberMe bool) (Session, error) {
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
	if user.IsEnded(now) {
		return Session{}, loginmodel.ErrAccountEnded
	}
	refreshToken, stored, err := newRefreshToken(user.ID, userAgent, rememberMe, now)
	if err != nil {
		return Session{}, err
	}
	if err := s.refreshTokenRepo.Create(ctx, stored); err != nil {
		return Session{}, err
	}
	sessionEnd := stored.ExpiresAt

	// Gagal mencatat waktu login bukan alasan untuk menggagalkan login.
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, now)

	return s.newSession(user, refreshToken, sessionEnd, rememberMe, now)
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
	if user.IsEnded(now) {
		return Session{}, loginmodel.ErrAccountEnded
	}

	newRefreshToken, err := token.NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	err = s.refreshTokenRepo.Rotate(ctx, oldToken.ID, loginmodel.RefreshToken{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  token.HashRefreshToken(newRefreshToken),
		UserAgent:  optionalString(userAgent),
		ExpiresAt:  oldToken.ExpiresAt, // akhir sesi tetap sama
		RememberMe: oldToken.RememberMe,
		CreatedAt:  now,
	}, now)
	if err != nil {
		return Session{}, err
	}

	return s.newSession(user, newRefreshToken, oldToken.ExpiresAt, oldToken.RememberMe, now)
}

// Logout menutup sesi milik refresh token ini.
func (s *LoginService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokenRepo.RevokeByHash(ctx, token.HashRefreshToken(refreshToken), time.Now())
}

// GetUser mengambil data user yang sedang login, beserta jenjang & kelasnya.
func (s *LoginService) GetUser(ctx context.Context, userID string) (loginmodel.UserDetail, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// newSession membuat access token, lalu menyusun Session.
func (s *LoginService) newSession(user loginmodel.UserDetail, refreshToken string, sessionEnd time.Time, rememberMe bool, now time.Time) (Session, error) {
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
		RememberMe:            rememberMe,
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

// sessionEndFor menghitung akhir sesi baru: "Ingat saya" = 30 hari, tanpa = tanggal 1 bulan berikutnya.
func sessionEndFor(rememberMe bool, now time.Time) time.Time {
	if rememberMe {
		// Dibulatkan ke mikrodetik (presisi kolom TIMESTAMPTZ), supaya nilai di respons login sama
		// persis dengan yang tersimpan & dikirim ulang saat refresh.
		return now.Add(loginmodel.RememberMeDuration).Truncate(time.Microsecond)
	}
	return clock.StartOfNextMonth(now)
}

// newRefreshToken membuat refresh token acak + baris refresh_tokens-nya (yang disimpan hanya hash-nya).
func newRefreshToken(userID, userAgent string, rememberMe bool, now time.Time) (string, loginmodel.RefreshToken, error) {
	raw, err := token.NewRefreshToken()
	if err != nil {
		return "", loginmodel.RefreshToken{}, err
	}
	return raw, loginmodel.RefreshToken{
		ID:         uuid.NewString(),
		UserID:     userID,
		TokenHash:  token.HashRefreshToken(raw),
		UserAgent:  optionalString(userAgent),
		ExpiresAt:  sessionEndFor(rememberMe, now),
		RememberMe: rememberMe,
		CreatedAt:  now,
	}, nil
}

// ChangePassword mengganti kata sandi user yang sedang login (semua role, menu Profil).
//
//  1. Kata sandi lama harus benar (ErrWrongPassword) dan kata sandi baru harus berbeda (ErrSamePassword).
//  2. Kata sandi disimpan sebagai hash bcrypt; permintaan lupa kata sandi yang masih menunggu ikut selesai.
//  3. SEMUA sesi lama dicabut (perangkat lain logout), lalu sesi baru dibuat untuk perangkat ini
//     ("Ingat saya" mengikuti pilihan rememberMe), jadi user tidak perlu login ulang.
func (s *LoginService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword, userAgent string, rememberMe bool) (Session, error) {
	hash, err := s.userRepo.FindPasswordHash(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return Session{}, loginmodel.ErrWrongPassword
	}
	if currentPassword == newPassword {
		return Session{}, loginmodel.ErrSamePassword
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	if !user.IsActive() {
		return Session{}, loginmodel.ErrUserInactive
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	if err := s.userRepo.UpdateOwnPassword(ctx, userID, newHash, now); err != nil {
		return Session{}, err
	}

	refreshToken, stored, err := newRefreshToken(userID, userAgent, rememberMe, now)
	if err != nil {
		return Session{}, err
	}
	if err := s.refreshTokenRepo.ReplaceAllWithNew(ctx, userID, stored, now); err != nil {
		return Session{}, err
	}
	return s.newSession(user, refreshToken, stored.ExpiresAt, rememberMe, now)
}
