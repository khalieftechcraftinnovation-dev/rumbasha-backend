// Package coin (model) berisi struct data untuk fitur coin: paket isi coin, top-up oleh admin,
// dan riwayat coin siswa.
//
// Di package lain, import dengan alias coinmodel:
//
//	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/coin"
package coin

import (
	"errors"
	"strings"
	"time"
)

// Status paket coin.
const (
	PackageActive   = "active"
	PackageInactive = "inactive"
)

// Package adalah 1 paket isi coin (tabel coin_packages).
type Package struct {
	ID            string
	Name          string
	CoinAmount    int
	Price         int64 // rupiah
	Description   *string
	SortOrder     int
	Status        string
	CreatedBy     *string
	CreatedByName *string
	UpdatedBy     *string
	UpdatedByName *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PackageFilter adalah filter list paket. Status "" = semua.
type PackageFilter struct {
	Status string
	Search string // nama paket
}

// Student adalah siswa yang akan di-top-up (dicek sebelum coin ditambah).
type Student struct {
	ID          string
	Name        string
	StudentCode string
	Status      string
	CoinBalance int
}

// TopUpResult adalah hasil top-up 1 siswa.
type TopUpResult struct {
	StudentID     string
	StudentName   string
	StudentCode   string
	CoinAdded     int
	BalanceAfter  int
	TransactionID string
}

// TopUp adalah hasil top-up banyak siswa dengan 1 paket.
type TopUp struct {
	Package Package
	Results []TopUpResult
}

// Label jenis transaksi coin untuk ditampilkan FE.
var TypeLabels = map[string]string{
	"plan_grant":       "Bonus paket",
	"purchase":         "Isi coin",
	"usage":            "Pemakaian",
	"admin_adjustment": "Penyesuaian admin",
	"refund":           "Refund",
	"reward":           "Hadiah",
}

// Transaction adalah 1 baris riwayat coin (tabel coin_transactions) + nama referensinya.
type Transaction struct {
	ID            string
	Type          string
	Amount        int // positif = masuk, negatif = keluar
	BalanceAfter  int
	PricePaid     *int64
	ReferenceType *string
	ReferenceID   *string
	ReferenceName *string // contoh nama paket coin / nama quiz; nil kalau tidak ada
	Description   *string
	CreatedBy     *string
	CreatedByName *string
	CreatedAt     time.Time
}

// TransactionFilter adalah filter riwayat coin 1 siswa.
type TransactionFilter struct {
	Type          string // "" = semua
	Page, PerPage int
}

// StudentCoin adalah ringkasan coin 1 siswa (di atas riwayat).
type StudentCoin struct {
	StudentID   string
	Name        string
	StudentCode string
	CoinBalance int
}

// DailyQuota adalah jatah Daily Quiz gratis siswa hari ini (WIB).
type DailyQuota struct {
	FreePerDay int
	UsedToday  int
	CoinCost   int
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
	return "validasi coin gagal: " + strings.Join(parts, "; ")
}

var (
	ErrPackageNotFound = errors.New("paket coin tidak ditemukan")
	ErrStudentNotFound = errors.New("siswa tidak ditemukan")
	// ErrPackageNameTaken: nama paket sudah dipakai paket lain (index unik coin_packages_name_unique).
	ErrPackageNameTaken = errors.New("nama paket sudah dipakai")
	// ErrTopUpDuplicate: request_id top-up yang sama sudah pernah diproses (klik 2x / kirim ulang).
	ErrTopUpDuplicate = errors.New("top-up dengan request_id ini sudah diproses")
)
