// Package login (model) berisi struct data untuk fitur login: user dan sesi login.
// Satu struct model mewakili satu tabel di database.
//
// Di package lain, import dengan alias loginmodel:
//
//	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
package login

import (
	"errors"
	"time"
)

// Role user. Nilainya sama persis dengan isi kolom users.role.
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleTutor      = "tutor"
	RoleStudent    = "student"
)

// Status akun.
const (
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
)

// User mewakili satu baris di tabel "users".
//
// Catatan Go: tipe *string (pointer) dipakai untuk kolom yang boleh NULL.
// Kalau nilainya nil, artinya NULL di database.
type User struct {
	ID               string
	Role             string
	BranchID         *string
	Username         string
	PasswordHash     string
	RealName         string
	Email            *string
	Phone            *string
	ParentPhone      *string
	SchoolOrigin     *string
	EducationLevelID *string
	GradeID          *string
	Status           string
	LastLoginAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// TableName memberi tahu GORM nama tabel untuk model ini.
func (User) TableName() string {
	return "users"
}

// IsActive bernilai true kalau akun boleh dipakai login.
func (u User) IsActive() bool {
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// RefreshToken mewakili satu baris di tabel "refresh_tokens" (satu sesi login).
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	UserAgent *string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// TableName memberi tahu GORM nama tabel untuk model ini.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// Daftar error untuk login & sesi.
var (
	// Sengaja satu error untuk "username tidak ada" dan "password salah",
	// supaya orang lain tidak bisa menebak username mana yang terdaftar.
	ErrInvalidCredentials = errors.New("username atau kata sandi salah")
	ErrUserInactive       = errors.New("akun tidak aktif")
	ErrSessionExpired     = errors.New("sesi sudah berakhir")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
)
