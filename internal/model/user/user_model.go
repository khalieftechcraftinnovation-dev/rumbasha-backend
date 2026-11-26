// Package user (model) berisi struct data untuk menu User di panel admin: list, detail, ubah,
// dan hapus akun (siswa, tutor, admin, super admin).
//
// Di package lain, import dengan alias usermodel:
//
//	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
package user

import (
	"errors"
	"strconv"
	"strings"
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// ListFilter adalah filter list user. Field kosong ("") = tidak difilter.
type ListFilter struct {
	Role             string
	EducationLevelID string
	GradeID          string
	BranchID         string
	Status           string
	Search           string // nama, username, atau email
	// Filter paket akses (hanya siswa yang cocok): PlanID = paket aktif hari ini;
	// Subscription = active (punya paket aktif), expiring (aktif & berakhir ≤ 7 hari lagi),
	// none (tidak punya paket aktif).
	PlanID        string
	Subscription  string
	Today         string // tanggal WIB "2026-11-26", diisi service
	ExpiringUntil string // Today + 6 hari (berakhir dalam 7 hari termasuk hari ini), diisi service
	Page, PerPage int
}

// Filter paket akses untuk ListFilter.Subscription.
const (
	SubscriptionActive   = "active"
	SubscriptionExpiring = "expiring"
	SubscriptionNone     = "none"
)

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// ListItem adalah 1 baris list user.
type ListItem struct {
	ID             string
	Username       string
	RealName       string
	Email          *string
	Phone          *string
	Role           string
	Branch         *Ref
	EducationLevel *Ref
	Grade          *Ref
	StudentCode    *string // hanya siswa
	CoinBalance    *int    // hanya siswa
	Status         string
	JoinDate       time.Time  // tanggal bergabung
	EndDate        *time.Time // tanggal berakhir (inklusif); lewat = tidak bisa login
	LastLoginAt    *time.Time
	AvatarURL      *string // foto profil (null = belum ada)
	CreatedAt      time.Time
	// Akses hari ini (paket aktif + akses khusus); hanya siswa, nil untuk role lain.
	Access *subscriptionmodel.Access
}

// RoleCounts adalah jumlah user per role (angka di tab menu User).
type RoleCounts struct {
	SuperAdmin, Admin, Tutor, Student int
}

// Detail adalah data lengkap 1 user (+ data siswa kalau role student).
type Detail struct {
	ListItem
	ParentPhone  *string
	SchoolOrigin *string
	DateOfBirth  *time.Time // hanya siswa
	TargetSchool *string    // hanya siswa
	UpdatedBy    *Ref       // admin yang terakhir mengubah lewat menu User
	UpdatedAt    time.Time
}

// Update adalah isi baru 1 user (ganti penuh). PasswordHash nil = password tidak diubah.
type Update struct {
	ID               string
	Username         string
	RealName         string
	Email            *string
	Phone            *string
	ParentPhone      *string
	SchoolOrigin     *string
	BranchID         *string
	EducationLevelID *string
	GradeID          *string
	Status           string
	JoinDate         time.Time
	EndDate          *time.Time
	PasswordHash     *string
	// Khusus siswa (tabel students).
	DateOfBirth  *time.Time
	TargetSchool *string
}

// NewUser adalah 1 akun baru yang siap disimpan (sudah divalidasi, password sudah di-hash).
type NewUser struct {
	Role             string
	Username         string
	PasswordHash     string
	RealName         string
	Email            *string
	Phone            *string
	ParentPhone      *string
	SchoolOrigin     *string
	BranchID         *string
	EducationLevelID *string
	GradeID          *string
	Status           string
	JoinDate         time.Time
	EndDate          *time.Time
	// Khusus siswa (tabel students).
	DateOfBirth  *time.Time
	TargetSchool *string
}

// Lookups adalah data pendukung validasi yang diambil sekali untuk 1 request (bukan per user):
// cabang aktif, jenjang/kelas, dan username yang sudah dipakai.
type Lookups struct {
	ActiveBranches map[string]bool
	MasterData     map[string]MasterDataItem
	TakenUsernames map[string]bool // termasuk akun yang sudah dihapus
}

// Target adalah user yang akan diubah / dihapus (dicek hak aksesnya dulu).
type Target struct {
	ID, Role, RealName string
}

// MasterDataItem adalah jenjang / kelas yang dicek saat ubah user.
type MasterDataItem struct {
	ID, Type, Name string
	ParentID       *string
	Status         string
}

// CanManage = true kalau actorRole boleh membuat / mengubah / menghapus akun ber-role targetRole:
// super admin boleh semua; admin hanya tutor & siswa.
func CanManage(actorRole, targetRole string) bool {
	if actorRole == loginmodel.RoleSuperAdmin {
		return true
	}
	return actorRole == loginmodel.RoleAdmin &&
		(targetRole == loginmodel.RoleTutor || targetRole == loginmodel.RoleStudent)
}

// FieldError & ValidationError: kesalahan data request (dibalas 422).
type FieldError struct {
	Field, Message string
	Row            int    // khusus file Excel: nomor baris (baris 1 = header)
	Column         string // khusus file Excel: huruf kolom, contoh "C"
}

type ValidationError struct {
	Message string // pesan utama (opsional), contoh "File Excel berisi 3 kesalahan, semua data ditolak"
	Fields  []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		location := f.Field
		if f.Row > 0 {
			location = f.Column + strconv.Itoa(f.Row) + " " + f.Field
		}
		parts = append(parts, location+": "+f.Message)
	}
	return "validasi user gagal: " + strings.Join(parts, "; ")
}

// ImportedUser adalah 1 baris Excel yang valid, siap dikirim ke POST /api/v1/users.
type ImportedUser struct {
	Row                                       int // nomor baris di Excel
	Role, Username, Name, Password            string
	Email, Phone, ParentPhone, SchoolOrigin   *string
	BranchID, EducationLevelID, GradeID       *string
	BranchName, EducationLevelName, GradeName *string // nama dari file (untuk preview)
	Status, JoinDate                          string
	EndDate, DateOfBirth, TargetSchool        *string
}

var (
	ErrUserNotFound = errors.New("user tidak ditemukan")
	// ErrForbidden: admin mencoba mengubah / menghapus akun admin atau super admin.
	ErrForbidden = errors.New("tidak boleh mengubah akun ini")
	// ErrUsernameTaken: username sudah dipakai akun lain (termasuk akun yang sudah dihapus).
	ErrUsernameTaken = errors.New("username sudah dipakai")
)
