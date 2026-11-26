// Package profile (model) berisi struct data halaman Profil siswa: data diri, kontak orang tua,
// foto profil, dan ringkasan (paket, coin, peringkat, point, quiz selesai bulan ini).
//
// Keputusan pemilik produk (BD-014): siswa boleh mengubah sendiri nama lengkap, no HP, email,
// tanggal lahir, foto profil, dan no HP orang tua / wali. Username, jenjang, kelas, cabang, asal &
// target sekolah, status, dan tanggal bergabung / berakhir tetap hanya admin.
//
// Di package lain, import dengan alias profilemodel:
//
//	profilemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user/profile"
package profile

import (
	"errors"
	"strings"
	"time"
)

// Ref adalah pasangan id + nama.
type Ref struct {
	ID, Name string
}

// Profile adalah data diri siswa.
type Profile struct {
	ID             string
	Username       string
	RealName       string
	StudentCode    *string
	Email          *string
	Phone          *string
	ParentPhone    *string
	SchoolOrigin   *string
	TargetSchool   *string
	DateOfBirth    *time.Time
	EducationLevel *Ref
	Grade          *Ref
	Branch         *Ref
	AvatarPath     *string
	AvatarURL      *string
	CoinBalance    int
	JoinDate       time.Time
	EndDate        *time.Time
}

// Stats = ringkasan di halaman Profil.
type Stats struct {
	Month            string // bulan WIB, contoh 2026-11
	DailyRank        *int   // peringkat Daily Quiz bulan ini di kelasnya (nil = belum ikut)
	DailyRankTotal   int    // jumlah siswa di papan Daily Quiz kelasnya bulan ini
	PointsThisMonth  int    // point Daily Quiz bulan ini
	TotalPoints      int    // semua point (Daily Quiz semua bulan + Uji Kom + Tryout)
	QuizzesThisMonth int    // Daily Quiz + Uji Kom + Tryout yang dikumpulkan bulan ini
}

// Update = data diri yang boleh diubah siswa (ganti penuh).
type Update struct {
	RealName    string
	Email       *string
	Phone       *string
	DateOfBirth *time.Time
}

// FieldError & ValidationError: kesalahan data request (dibalas 422).
type FieldError struct {
	Field, Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validasi profil gagal: " + strings.Join(parts, "; ")
}

// ErrNotFound: akun siswa tidak ada / sudah dihapus.
var ErrNotFound = errors.New("profil tidak ditemukan")

// MaskPhone menyamarkan no HP untuk ditampilkan: 4 angka depan & 4 angka belakang terlihat,
// contoh "081234562210" → "0812-xxxx-2210". Kurang dari 9 angka → semua kecuali 2 terakhir disamarkan.
func MaskPhone(phone string) string {
	digits := make([]rune, 0, len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	n := len(digits)
	switch {
	case n == 0:
		return ""
	case n < 9:
		keep := min(2, n)
		return strings.Repeat("x", n-keep) + string(digits[n-keep:])
	}
	return string(digits[:4]) + "-xxxx-" + string(digits[n-4:])
}
