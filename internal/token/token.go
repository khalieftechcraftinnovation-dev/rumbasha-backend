// Package token membuat dan membaca token login.
//
// Ada 2 jenis token:
//   - Access token  → JWT berumur pendek (contoh 15 menit). Dikirim di setiap
//     request lewat header "Authorization: Bearer <token>". Tidak disimpan di database.
//   - Refresh token → string acak berumur panjang (sampai awal bulan berikutnya).
//     Dipakai untuk meminta access token baru. Yang disimpan di database hanya hash-nya.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidAccessToken dikembalikan kalau access token rusak, palsu, atau kedaluwarsa.
var ErrInvalidAccessToken = errors.New("access token tidak valid")

// AccessClaims adalah isi (payload) access token.
type AccessClaims struct {
	Role string `json:"role"`
	// RegisteredClaims berisi field standar JWT: sub (user id), exp, iat.
	jwt.RegisteredClaims
}

// NewAccessToken membuat access token (JWT) untuk user, berlaku sampai expiresAt.
func NewAccessToken(userID, role, secret string, expiresAt time.Time) (string, error) {
	claims := AccessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	// HS256 = ditandatangani dengan secret. Siapa pun yang tidak tahu secret
	// tidak bisa membuat atau mengubah token tanpa ketahuan.
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken membaca dan memvalidasi access token, lalu mengembalikan isinya.
func ParseAccessToken(tokenString, secret string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return []byte(secret), nil },
		// Hanya terima HS256, supaya token dengan algoritma lain (misalnya "none") ditolak.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAccessToken, err)
	}
	return claims, nil
}

// NewRefreshToken membuat refresh token acak (64 karakter hex).
// Yang dikirim ke client adalah token ini; yang disimpan di database adalah HashRefreshToken(token).
func NewRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashRefreshToken mengubah refresh token menjadi hash SHA-256.
// Kalau database bocor, hash ini tidak bisa dipakai untuk login.
func HashRefreshToken(refreshToken string) string {
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}
