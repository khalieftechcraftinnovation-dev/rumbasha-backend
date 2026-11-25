// Package coin (dto) berisi bentuk request dan response JSON untuk fitur coin.
//
// Di package lain, import dengan alias coindto:
//
//	coindto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/coin"
package coin

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/coin"
)

// ---------------------------------------------------------------------------
// Paket isi coin
// ---------------------------------------------------------------------------

// PackageRequest adalah body POST /api/v1/coin-packages dan PUT /api/v1/coin-packages/:id (ganti penuh).
//
//	{"name": "300 coin", "coin_amount": 300, "price": 68000, "description": "Hemat 10%", "sort_order": 2, "status": "active"}
type PackageRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	CoinAmount  int     `json:"coin_amount" binding:"required,min=1,max=1000000"`
	Price       *int64  `json:"price" binding:"required,min=0,max=100000000"` // rupiah, 0 = gratis
	Description *string `json:"description" binding:"omitempty,max=150"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty,min=0,max=10000"`   // default 0
	Status      string  `json:"status" binding:"omitempty,oneof=active inactive"` // default active
}

// ListPackagesQuery adalah query string GET /api/v1/coin-packages.
// Siswa selalu hanya melihat paket aktif (status diabaikan).
type ListPackagesQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=active inactive"`
	Search string `form:"search" binding:"omitempty,max=100"`
}

// DeletePackagesRequest adalah body DELETE /api/v1/coin-packages: hapus 1-100 paket (soft delete).
type DeletePackagesRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeletePackagesResponse berisi ID paket yang dihapus.
type DeletePackagesResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// PackageResponse adalah 1 paket coin (untuk admin).
type PackageResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CoinAmount  int       `json:"coin_amount"`
	Price       int64     `json:"price"` // rupiah
	Description *string   `json:"description"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedBy   *UserRef  `json:"created_by"` // null = data awal
	UpdatedBy   *UserRef  `json:"updated_by"` // null = belum pernah diubah
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StudentPackageResponse adalah paket coin untuk siswa (tab "Isi coin"): tanpa data admin.
type StudentPackageResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	CoinAmount  int     `json:"coin_amount"`
	Price       int64   `json:"price"`
	Description *string `json:"description"`
}

// UserRef adalah id + nama user (nama bisa null).
type UserRef struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

func userRef(id, name *string) *UserRef {
	if id == nil {
		return nil
	}
	return &UserRef{ID: *id, Name: name}
}

// ToPackageResponse mengubah 1 paket menjadi response admin.
func ToPackageResponse(p coinmodel.Package) PackageResponse {
	return PackageResponse{
		ID: p.ID, Name: p.Name, CoinAmount: p.CoinAmount, Price: p.Price, Description: p.Description,
		SortOrder: p.SortOrder, Status: p.Status,
		CreatedBy: userRef(p.CreatedBy, p.CreatedByName), UpdatedBy: userRef(p.UpdatedBy, p.UpdatedByName),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ToPackageResponses mengubah daftar paket menjadi response admin. Kosong = [].
func ToPackageResponses(items []coinmodel.Package) []PackageResponse {
	result := make([]PackageResponse, 0, len(items))
	for _, p := range items {
		result = append(result, ToPackageResponse(p))
	}
	return result
}

// ToStudentPackageResponse mengubah 1 paket menjadi response siswa.
func ToStudentPackageResponse(p coinmodel.Package) StudentPackageResponse {
	return StudentPackageResponse{ID: p.ID, Name: p.Name, CoinAmount: p.CoinAmount, Price: p.Price, Description: p.Description}
}

// ToStudentPackageResponses mengubah daftar paket menjadi response siswa. Kosong = [].
func ToStudentPackageResponses(items []coinmodel.Package) []StudentPackageResponse {
	result := make([]StudentPackageResponse, 0, len(items))
	for _, p := range items {
		result = append(result, ToStudentPackageResponse(p))
	}
	return result
}

// ---------------------------------------------------------------------------
// Top-up coin siswa oleh admin
// ---------------------------------------------------------------------------

// TopUpRequest adalah body POST /api/v1/coin-topups: tambah coin 1-100 siswa sekaligus sesuai 1 paket
// (dicatat sebagai pembelian tunai di cabang).
//
//	{"coin_package_id": "...", "student_ids": ["...", "..."], "note": "Bayar tunai", "request_id": "..."}
type TopUpRequest struct {
	CoinPackageID string   `json:"coin_package_id" binding:"required,uuid"`
	StudentIDs    []string `json:"student_ids" binding:"required,min=1,max=100,unique,dive,uuid"`
	Note          *string  `json:"note" binding:"omitempty,max=150"` // catatan, ikut di deskripsi transaksi
	// Opsional tapi disarankan: UUID baru dari FE setiap kali tombol simpan ditekan. Request yang sama
	// terkirim 2x (klik ganda / jaringan) → yang kedua ditolak 409, coin tidak bertambah 2x.
	RequestID *string `json:"request_id" binding:"omitempty,uuid"`
}

// TopUpResponse adalah hasil top-up: paket yang dipakai + hasil setiap siswa (urutan sama dengan request).
type TopUpResponse struct {
	Package    StudentPackageResponse `json:"package"`
	TotalCoins int                    `json:"total_coins"` // coin_amount x jumlah siswa
	TotalPrice int64                  `json:"total_price"` // price x jumlah siswa (rupiah)
	Students   []TopUpStudentResponse `json:"students"`
}

// TopUpStudentResponse adalah hasil top-up 1 siswa.
type TopUpStudentResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StudentCode   string `json:"student_code"`
	CoinAdded     int    `json:"coin_added"`
	BalanceAfter  int    `json:"balance_after"`
	TransactionID string `json:"transaction_id"`
}

// ToTopUpResponse mengubah hasil top-up menjadi response.
func ToTopUpResponse(t coinmodel.TopUp) TopUpResponse {
	r := TopUpResponse{
		Package:    ToStudentPackageResponse(t.Package),
		TotalCoins: t.Package.CoinAmount * len(t.Results),
		TotalPrice: t.Package.Price * int64(len(t.Results)),
		Students:   make([]TopUpStudentResponse, 0, len(t.Results)),
	}
	for _, s := range t.Results {
		r.Students = append(r.Students, TopUpStudentResponse{
			ID: s.StudentID, Name: s.StudentName, StudentCode: s.StudentCode,
			CoinAdded: s.CoinAdded, BalanceAfter: s.BalanceAfter, TransactionID: s.TransactionID,
		})
	}
	return r
}

// ---------------------------------------------------------------------------
// Riwayat coin
// ---------------------------------------------------------------------------

// TransactionsQuery adalah query string riwayat coin.
type TransactionsQuery struct {
	Type    string `form:"type" binding:"omitempty,oneof=plan_grant purchase usage admin_adjustment refund reward"`
	Page    int    `form:"page" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// HistoryResponse adalah data riwayat coin 1 siswa: ringkasan saldo + 1 halaman transaksi.
type HistoryResponse struct {
	Student StudentCoinResponse `json:"student"`
	// Jatah Daily Quiz hari ini; hanya di riwayat milik siswa sendiri (null untuk admin).
	DailyQuiz    *DailyQuotaResponse   `json:"daily_quiz"`
	Transactions []TransactionResponse `json:"transactions"`
}

// StudentCoinResponse = siswa + saldo coin saat ini.
type StudentCoinResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StudentCode string `json:"student_code"`
	CoinBalance int    `json:"coin_balance"`
}

// DailyQuotaResponse = jatah Daily Quiz gratis hari ini (WIB), contoh "Jatah gratis hari ini 1 dari 2".
type DailyQuotaResponse struct {
	FreePerDay    int `json:"free_per_day"`
	UsedToday     int `json:"used_today"`
	RemainingFree int `json:"remaining_free"`
	CoinCostAfter int `json:"coin_cost_after"` // biaya coin per Daily Quiz setelah jatah habis
}

// TransactionResponse adalah 1 baris riwayat coin.
type TransactionResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`       // plan_grant, purchase, usage, admin_adjustment, refund, reward
	TypeLabel    string        `json:"type_label"` // contoh "Isi coin", "Pemakaian", "Hadiah"
	Amount       int           `json:"amount"`     // positif = masuk, negatif = keluar
	BalanceAfter int           `json:"balance_after"`
	PricePaid    *int64        `json:"price_paid"` // rupiah, hanya pembelian
	Reference    *ReferenceRef `json:"reference"`  // asal transaksi; null kalau tidak ada
	Description  *string       `json:"description"`
	CreatedBy    *UserRef      `json:"created_by"` // admin pencatat; null = sistem / siswa sendiri
	CreatedAt    time.Time     `json:"created_at"` // WIB
}

// ReferenceRef = asal transaksi, contoh {"type": "coin_package", "id": "...", "name": "300 coin"}.
type ReferenceRef struct {
	Type string  `json:"type"`
	ID   *string `json:"id"`
	Name *string `json:"name"`
}

// ToHistoryResponse mengubah riwayat coin menjadi response. quota nil = tidak ditampilkan.
func ToHistoryResponse(s coinmodel.StudentCoin, items []coinmodel.Transaction, quota *coinmodel.DailyQuota) HistoryResponse {
	r := HistoryResponse{
		Student:      StudentCoinResponse{ID: s.StudentID, Name: s.Name, StudentCode: s.StudentCode, CoinBalance: s.CoinBalance},
		Transactions: make([]TransactionResponse, 0, len(items)),
	}
	if quota != nil {
		r.DailyQuiz = &DailyQuotaResponse{
			FreePerDay: quota.FreePerDay, UsedToday: quota.UsedToday,
			RemainingFree: max(quota.FreePerDay-quota.UsedToday, 0), CoinCostAfter: quota.CoinCost,
		}
	}
	for _, t := range items {
		item := TransactionResponse{
			ID: t.ID, Type: t.Type, TypeLabel: coinmodel.TypeLabels[t.Type], Amount: t.Amount,
			BalanceAfter: t.BalanceAfter, PricePaid: t.PricePaid, Description: t.Description,
			CreatedBy: userRef(t.CreatedBy, t.CreatedByName), CreatedAt: t.CreatedAt.In(clock.Jakarta),
		}
		if t.ReferenceType != nil {
			item.Reference = &ReferenceRef{Type: *t.ReferenceType, ID: t.ReferenceID, Name: t.ReferenceName}
		}
		r.Transactions = append(r.Transactions, item)
	}
	return r
}
