package login

import (
	"context"
	"time"

	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// RefreshTokenRepository mengakses tabel refresh_tokens (sesi login).
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository membuat RefreshTokenRepository. Dipanggil sekali di main.go.
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create menyimpan sesi login baru. [GORM]
func (r *RefreshTokenRepository) Create(ctx context.Context, refreshToken loginmodel.RefreshToken) error {
	return r.db.WithContext(ctx).Create(&refreshToken).Error
}

// FindValidByHash mencari sesi yang masih berlaku: belum di-logout/ditukar dan
// belum lewat expires_at. [Native SQL]
func (r *RefreshTokenRepository) FindValidByHash(ctx context.Context, tokenHash string, now time.Time) (loginmodel.RefreshToken, error) {
	var refreshToken loginmodel.RefreshToken
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, user_id, token_hash, user_agent, expires_at, revoked_at, created_at
			FROM refresh_tokens
			WHERE token_hash = ?
			  AND revoked_at IS NULL
			  AND expires_at > ?`, tokenHash, now).
		Scan(&refreshToken).Error
	if err != nil {
		return loginmodel.RefreshToken{}, err
	}

	// Scan tidak mengembalikan error kalau datanya tidak ada, jadi cek ID-nya.
	if refreshToken.ID == "" {
		return loginmodel.RefreshToken{}, loginmodel.ErrSessionExpired
	}
	return refreshToken, nil
}

// Rotate menukar sesi lama dengan sesi baru (dipakai saat refresh).
// Dua langkah ini dijalankan dalam satu database transaction:
// kalau salah satu gagal, keduanya dibatalkan.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldID string, newToken loginmodel.RefreshToken, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// "AND revoked_at IS NULL" mencegah 2 request refresh bersamaan dengan
		// token yang sama sama-sama berhasil: hanya yang pertama mengubah 1 baris.
		result := tx.Exec(`
			UPDATE refresh_tokens
			SET revoked_at = ?
			WHERE id = ? AND revoked_at IS NULL`, now, oldID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return loginmodel.ErrSessionExpired
		}

		return tx.Create(&newToken).Error
	})
}

// RevokeByHash menutup sesi (logout). [Native SQL]
// Tidak error kalau sesinya sudah tidak ada (UPDATE 0 baris tetap dianggap berhasil),
// jadi logout dua kali tetap aman.
func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string, now time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE refresh_tokens
		SET revoked_at = ?
		WHERE token_hash = ? AND revoked_at IS NULL`, now, tokenHash).Error
}
