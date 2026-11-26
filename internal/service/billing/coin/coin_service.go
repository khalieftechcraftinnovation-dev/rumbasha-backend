// Package coin (service) berisi logika fitur coin: paket isi coin (CRUD), top-up coin siswa oleh
// admin per paket (banyak siswa sekaligus), dan riwayat coin siswa.
//
// Di package lain, import dengan alias coinservice:
//
//	coinservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/coin"
package coin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	coindto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/billing/coin"
	coinmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/coin"
	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/billing/coin"
)

// CoinService berisi logika fitur coin.
type CoinService struct {
	repo *coinrepo.CoinRepository
}

// NewCoinService membuat CoinService. Dipanggil sekali di main.go.
func NewCoinService(repo *coinrepo.CoinRepository) *CoinService {
	return &CoinService{repo: repo}
}

// ---------------------------------------------------------------------------
// Paket isi coin
// ---------------------------------------------------------------------------

// ListPackages mengambil paket coin. onlyActive = true untuk siswa (hanya paket aktif, filter status
// dari query diabaikan).
func (s *CoinService) ListPackages(ctx context.Context, q coindto.ListPackagesQuery, onlyActive bool) ([]coinmodel.Package, error) {
	f := coinmodel.PackageFilter{Status: q.Status, Search: strings.TrimSpace(q.Search)}
	if onlyActive {
		f.Status = coinmodel.PackageActive
	}
	return s.repo.ListPackages(ctx, f)
}

// GetPackage mengambil 1 paket. Bukan UUID / tidak ada / sudah dihapus → ErrPackageNotFound.
func (s *CoinService) GetPackage(ctx context.Context, id string) (coinmodel.Package, error) {
	if uuid.Validate(id) != nil {
		return coinmodel.Package{}, coinmodel.ErrPackageNotFound
	}
	return s.repo.FindPackage(ctx, id)
}

// CreatePackage membuat paket baru lalu mengembalikan datanya.
func (s *CoinService) CreatePackage(ctx context.Context, req coindto.PackageRequest, adminID string) (coinmodel.Package, error) {
	p, err := buildPackage(req)
	if err != nil {
		return coinmodel.Package{}, err
	}
	id, err := s.repo.CreatePackage(ctx, p, adminID)
	if err != nil {
		return coinmodel.Package{}, packageNameError(err)
	}
	return s.repo.FindPackage(ctx, id)
}

// UpdatePackage mengganti isi 1 paket (ganti penuh). Harga & jumlah coin baru hanya berlaku untuk
// top-up berikutnya; riwayat lama menyimpan harga saat itu (price_paid).
func (s *CoinService) UpdatePackage(ctx context.Context, id string, req coindto.PackageRequest, adminID string) (coinmodel.Package, error) {
	if uuid.Validate(id) != nil {
		return coinmodel.Package{}, coinmodel.ErrPackageNotFound
	}
	p, err := buildPackage(req)
	if err != nil {
		return coinmodel.Package{}, err
	}
	p.ID = id
	if err := s.repo.UpdatePackage(ctx, p, adminID); err != nil {
		return coinmodel.Package{}, packageNameError(err)
	}
	return s.repo.FindPackage(ctx, id)
}

// DeletePackages menghapus 1-100 paket (soft delete). Semua ID harus ada; kalau tidak, tidak ada yang
// terhapus dan kesalahannya per field "ids[i]".
func (s *CoinService) DeletePackages(ctx context.Context, ids []string, adminID string) ([]string, error) {
	missing, err := s.repo.SoftDeletePackages(ctx, ids, adminID)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		isMissing := make(map[string]bool, len(missing))
		for _, id := range missing {
			isMissing[id] = true
		}
		errs := []coinmodel.FieldError{}
		for i, id := range ids {
			if isMissing[id] {
				errs = append(errs, coinmodel.FieldError{Field: fmt.Sprintf("ids[%d]", i), Message: "paket tidak ditemukan atau sudah dihapus"})
			}
		}
		return nil, &coinmodel.ValidationError{Fields: errs}
	}
	return ids, nil
}

// buildPackage merapikan isi request menjadi paket (nama di-trim, default sort_order 0 & status active).
func buildPackage(req coindto.PackageRequest) (coinmodel.Package, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return coinmodel.Package{}, &coinmodel.ValidationError{Fields: []coinmodel.FieldError{{Field: "name", Message: "wajib diisi"}}}
	}
	p := coinmodel.Package{Name: name, CoinAmount: req.CoinAmount, Price: *req.Price, Status: req.Status}
	if req.Description != nil {
		if d := strings.TrimSpace(*req.Description); d != "" {
			p.Description = &d
		}
	}
	if req.SortOrder != nil {
		p.SortOrder = *req.SortOrder
	}
	if p.Status == "" {
		p.Status = coinmodel.PackageActive
	}
	return p, nil
}

// packageNameError mengubah ErrPackageNameTaken menjadi pesan validasi field name.
func packageNameError(err error) error {
	if err == coinmodel.ErrPackageNameTaken {
		return &coinmodel.ValidationError{Fields: []coinmodel.FieldError{{Field: "name", Message: "sudah dipakai paket lain"}}}
	}
	return err
}

// ---------------------------------------------------------------------------
// Top-up coin siswa oleh admin
// ---------------------------------------------------------------------------

// TopUp menambah coin 1-100 siswa sekaligus sesuai 1 paket aktif. Dicatat sebagai pembelian tunai di
// cabang (coin_transactions type purchase, harga paket, admin pencatat). Semua atau tidak sama sekali:
// paket tidak aktif / siswa tidak ditemukan → 422 dengan field asalnya, tidak ada coin yang berubah.
func (s *CoinService) TopUp(ctx context.Context, req coindto.TopUpRequest, adminID string) (coinmodel.TopUp, error) {
	errs := []coinmodel.FieldError{}

	p, err := s.repo.FindPackage(ctx, req.CoinPackageID)
	switch {
	case err == coinmodel.ErrPackageNotFound:
		errs = append(errs, coinmodel.FieldError{Field: "coin_package_id", Message: "paket coin tidak ditemukan atau sudah dihapus"})
	case err != nil:
		return coinmodel.TopUp{}, err
	case p.Status != coinmodel.PackageActive:
		errs = append(errs, coinmodel.FieldError{Field: "coin_package_id", Message: fmt.Sprintf("paket %q tidak aktif", p.Name)})
	}

	students, err := s.repo.FindStudents(ctx, req.StudentIDs)
	if err != nil {
		return coinmodel.TopUp{}, err
	}
	for i, id := range req.StudentIDs {
		if _, ok := students[id]; !ok {
			errs = append(errs, coinmodel.FieldError{Field: fmt.Sprintf("student_ids[%d]", i), Message: "siswa tidak ditemukan"})
		}
	}
	if len(errs) > 0 {
		return coinmodel.TopUp{}, &coinmodel.ValidationError{Fields: errs}
	}

	description := "Isi coin paket " + p.Name + " (dicatat admin)"
	if req.Note != nil && strings.TrimSpace(*req.Note) != "" {
		description += ": " + strings.TrimSpace(*req.Note)
	}
	results, err := s.repo.TopUp(ctx, p, req.StudentIDs, req.RequestID, description, adminID, time.Now())
	if err != nil {
		return coinmodel.TopUp{}, err
	}

	// Urutkan hasil sesuai urutan student_ids di request.
	byID := make(map[string]coinmodel.TopUpResult, len(results))
	for _, r := range results {
		byID[r.StudentID] = r
	}
	ordered := make([]coinmodel.TopUpResult, 0, len(results))
	for _, id := range req.StudentIDs {
		ordered = append(ordered, byID[id])
	}
	return coinmodel.TopUp{Package: p, Results: ordered}, nil
}

// ---------------------------------------------------------------------------
// Riwayat coin
// ---------------------------------------------------------------------------

// History mengambil saldo + 1 halaman riwayat coin 1 siswa (terbaru dulu). withQuota = true untuk
// siswa sendiri: ikut jatah Daily Quiz gratis hari ini. Bukan siswa / tidak ada → ErrStudentNotFound.
func (s *CoinService) History(ctx context.Context, studentID string, q coindto.TransactionsQuery, withQuota bool) (coinmodel.StudentCoin, []coinmodel.Transaction, *coinmodel.DailyQuota, coinmodel.TransactionFilter, int, error) {
	f := coinmodel.TransactionFilter{Type: q.Type, Page: q.Page, PerPage: q.PerPage}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	if uuid.Validate(studentID) != nil {
		return coinmodel.StudentCoin{}, nil, nil, f, 0, coinmodel.ErrStudentNotFound
	}

	student, err := s.repo.FindStudentCoin(ctx, studentID)
	if err != nil {
		return coinmodel.StudentCoin{}, nil, nil, f, 0, err
	}
	items, total, err := s.repo.ListTransactions(ctx, studentID, f)
	if err != nil {
		return coinmodel.StudentCoin{}, nil, nil, f, 0, err
	}
	var quota *coinmodel.DailyQuota
	if withQuota {
		q, err := s.repo.FindDailyQuota(ctx, studentID, clock.StartOfDayInJakarta(time.Now()))
		if err != nil {
			return coinmodel.StudentCoin{}, nil, nil, f, 0, err
		}
		quota = &q
	}
	return student, items, quota, f, total, nil
}
