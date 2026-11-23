// Package login (repository) mengakses tabel untuk fitur login: users dan refresh_tokens
// (ditambah master_data_items untuk nama jenjang & kelas user).
//
// Di package lain, import dengan alias loginrepo:
//
//	loginrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/login"
package login

import (
	"context"
	"time"

	"gorm.io/gorm"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// UserRepository mengakses tabel users.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat UserRepository. Dipanggil sekali di main.go.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// userDetailQuery mengambil user (yang belum dihapus) beserta jenjang & kelasnya.
// Tabel master_data_items di-JOIN 2 kali dengan alias berbeda: "level" untuk jenjang, "grade" untuk kelas.
//
// LEFT JOIN (bukan JOIN): user tanpa jenjang/kelas (admin, tutor) tetap ditemukan,
// kolom jenjang/kelasnya saja yang bernilai NULL.
//
// Kondisi pencarian (username / id) ditambahkan oleh masing-masing fungsi di bawah.
const userDetailQuery = `
	SELECT u.*,
	       level.type AS education_level_type,
	       level.code AS education_level_code,
	       level.name AS education_level_name,
	       grade.type AS grade_type,
	       grade.code AS grade_code,
	       grade.name AS grade_name
	FROM users u
	LEFT JOIN master_data_items AS level ON level.id = u.education_level_id
	LEFT JOIN master_data_items AS grade ON grade.id = u.grade_id
	WHERE u.deleted_at IS NULL`

// FindByUsername mencari user berdasarkan username, beserta jenjang & kelasnya. [Native SQL]
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (loginmodel.UserDetail, error) {
	var user loginmodel.UserDetail
	err := r.db.WithContext(ctx).
		Raw(userDetailQuery+` AND u.username = ? LIMIT 1`, username).
		Scan(&user).Error
	if err != nil {
		return loginmodel.UserDetail{}, err
	}

	// Raw + Scan tidak mengembalikan error kalau datanya tidak ada, jadi dicek dari ID-nya.
	if user.ID == "" {
		return loginmodel.UserDetail{}, loginmodel.ErrUserNotFound
	}
	return user, nil
}

// FindByID mencari user berdasarkan ID, beserta jenjang & kelasnya. [Native SQL]
func (r *UserRepository) FindByID(ctx context.Context, id string) (loginmodel.UserDetail, error) {
	var user loginmodel.UserDetail
	err := r.db.WithContext(ctx).
		Raw(userDetailQuery+` AND u.id = ? LIMIT 1`, id).
		Scan(&user).Error
	if err != nil {
		return loginmodel.UserDetail{}, err
	}

	if user.ID == "" {
		return loginmodel.UserDetail{}, loginmodel.ErrUserNotFound
	}
	return user, nil
}

// UpdateLastLogin mencatat waktu login terakhir. [GORM]
//
//	UPDATE users SET last_login_at = ? WHERE id = ?
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&loginmodel.User{}).
		Where("id = ?", userID).
		UpdateColumn("last_login_at", at).Error
}
