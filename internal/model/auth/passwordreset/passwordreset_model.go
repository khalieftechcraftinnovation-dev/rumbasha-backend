// Package passwordreset (model) berisi struct data permintaan "Lupa kata sandi" yang diselesaikan
// admin (tabel password_reset_requests). Tidak ada email / OTP: admin menghubungi user lalu mereset
// kata sandinya dari panel admin.
//
// Di package lain, import dengan alias passwordresetmodel:
//
//	passwordresetmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/passwordreset"
package passwordreset

import (
	"errors"
	"time"
)

// Status permintaan.
const (
	StatusPending  = "pending"  // menunggu admin
	StatusResolved = "resolved" // kata sandi sudah direset (oleh admin, atau diubah sendiri oleh user)
	StatusRejected = "rejected" // ditolak admin (contoh permintaan palsu)
)

// MaxMatches = jumlah akun maksimal yang dicatat untuk 1 permintaan (contoh no HP dipakai 2 akun kakak-adik).
const MaxMatches = 5

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// User = akun pemilik permintaan (untuk dihubungi admin).
type User struct {
	ID, Username, RealName, Role, Status string
	Email, Phone, ParentPhone            *string
	Branch                               *Ref
}

// Request adalah 1 permintaan lupa kata sandi.
type Request struct {
	ID             string
	User           User
	Identifier     string
	Contact        *string
	Note           *string
	RequestCount   int
	Status         string
	IPAddress      *string
	RequestedAt    time.Time
	ResolvedAt     *time.Time
	ResolvedBy     *Ref
	ResolutionNote *string
	CreatedAt      time.Time
}

// NewRequest = isi form "Lupa kata sandi" yang sudah dirapikan.
type NewRequest struct {
	Identifier string
	Contact    *string
	Note       *string
	IPAddress  string
	UserAgent  string
}

// ListFilter adalah filter daftar permintaan.
type ListFilter struct {
	Status        string   // "" = semua
	Roles         []string // role pemilik akun yang boleh dilihat actor (admin: tutor & siswa)
	Search        string   // nama / username / email / no HP / yang diketik user
	Page, PerPage int
}

// Counts = jumlah permintaan per status (angka tab).
type Counts struct {
	Pending, Resolved, Rejected int
}

var (
	ErrNotFound = errors.New("permintaan reset kata sandi tidak ditemukan")
	// ErrNotPending: permintaan sudah diselesaikan / ditolak sebelumnya.
	ErrNotPending = errors.New("permintaan sudah diproses")
	// ErrForbidden: admin mencoba mereset kata sandi akun admin / super admin.
	ErrForbidden = errors.New("tidak boleh mereset kata sandi akun ini")
)
