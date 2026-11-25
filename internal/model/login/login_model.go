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
	JoinDate         time.Time  // tanggal bergabung
	EndDate          *time.Time // tanggal berakhir (inklusif, WIB); lewat = tidak bisa login. nil = tanpa batas
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// TableName memberi tahu GORM nama tabel untuk model ini.
func (User) TableName() string {
	return "users"
}

// UserDetail adalah User + data jenjang & kelasnya dari tabel master_data_items.
// Ini hasil JOIN (bukan satu tabel), jadi hanya dipakai untuk membaca data.
//
// Catatan Go: "User" ditulis tanpa nama field = embedding. Semua field & method
// User ikut bisa dipakai langsung, contoh: detail.Role, detail.IsActive().
// Mirip "extends" di PHP.
type UserDetail struct {
	User

	// Jenjang (master_data_items type education_level). NULL kalau user tidak punya jenjang.
	EducationLevelType *string
	EducationLevelCode *string
	EducationLevelName *string

	// Kelas (master_data_items type grade). NULL kalau user tidak punya kelas.
	GradeType *string
	GradeCode *string
	GradeName *string
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
	ErrAccountEnded       = errors.New("masa aktif akun sudah berakhir")
	ErrSessionExpired     = errors.New("sesi sudah berakhir")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
)

// IsEnded bernilai true kalau tanggal berakhir akun (end_date) sudah lewat pada waktu now (WIB).
// end_date inklusif: di tanggal itu akun masih bisa login, besoknya tidak.
// Tanggal dibandingkan sebagai teks "YYYY-MM-DD" supaya tidak terpengaruh zona waktu kolom DATE.
func (u User) IsEnded(now time.Time) bool {
	if u.EndDate == nil {
		return false
	}
	return u.EndDate.Format(time.DateOnly) < now.In(jakarta).Format(time.DateOnly)
}

// jakarta = zona WIB (sama dengan clock.Jakarta; ditulis ulang supaya model tidak bergantung package lain).
var jakarta = time.FixedZone("WIB", 7*60*60)
