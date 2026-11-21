package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
)

// UserRepository mengakses tabel users.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat UserRepository. Dipanggil sekali di main.go.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByUsername mencari user (yang belum dihapus) berdasarkan username. [GORM]
//
//	SELECT * FROM users WHERE username = ? AND deleted_at IS NULL LIMIT 1
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ? AND deleted_at IS NULL", username).
		Take(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, model.ErrUserNotFound
	}
	return user, err
}

// FindByID mencari user (yang belum dihapus) berdasarkan ID. [GORM]
func (r *UserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Take(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, model.ErrUserNotFound
	}
	return user, err
}

// UpdateLastLogin mencatat waktu login terakhir. [GORM]
//
//	UPDATE users SET last_login_at = ? WHERE id = ?
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("last_login_at", at).Error
}
