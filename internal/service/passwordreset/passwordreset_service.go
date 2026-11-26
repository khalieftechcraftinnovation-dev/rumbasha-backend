// Package passwordreset (service) berisi logika "Lupa kata sandi" lewat admin (keputusan pemilik
// produk: tanpa email / OTP):
//
//  1. User mengisi form "Lupa kata sandi" (username, email, atau no HP + kontak & pesan opsional).
//     Jawaban SELALU sama, baik akunnya ada maupun tidak, supaya form tidak bisa dipakai menebak akun.
//  2. Permintaan tampil di panel admin (1 akun = 1 permintaan menunggu; kirim lagi = jumlahnya bertambah).
//  3. Admin menghubungi user, lalu mereset kata sandinya (semua sesi user dicabut) atau menolak.
//     Admin hanya untuk akun tutor & siswa; super admin semua akun.
//
// Di package lain, import dengan alias passwordresetservice:
//
//	passwordresetservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/passwordreset"
package passwordreset

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	passwordresetdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/passwordreset"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	passwordresetmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/passwordreset"
	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
	passwordresetrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/passwordreset"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
)

// PasswordResetService berisi logika lupa kata sandi lewat admin.
type PasswordResetService struct {
	repo *passwordresetrepo.PasswordResetRepository
}

// NewPasswordResetService membuat PasswordResetService. Dipanggil sekali di main.go.
func NewPasswordResetService(repo *passwordresetrepo.PasswordResetRepository) *PasswordResetService {
	return &PasswordResetService{repo: repo}
}

var nonDigit = regexp.MustCompile(`\D`)

// phoneOf mengubah isian menjadi no HP angka saja (awalan 62 → 0). Kurang dari 8 angka = bukan no HP.
func phoneOf(s string) string {
	digits := nonDigit.ReplaceAllString(s, "")
	if strings.HasPrefix(digits, "62") {
		digits = "0" + digits[2:]
	}
	if len(digits) < 8 {
		return ""
	}
	return digits
}

func optional(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

// Forgot mencatat permintaan lupa kata sandi untuk akun yang cocok (username / email / no HP).
// Akun tidak ditemukan → tidak ada yang dicatat, tetapi tetap dianggap berhasil (tanpa error), supaya
// jawaban API sama untuk semua isian.
func (s *PasswordResetService) Forgot(ctx context.Context, req passwordresetdto.ForgotRequest, ip, userAgent string) error {
	identifier := strings.TrimSpace(req.Identifier)
	lower := strings.ToLower(identifier)
	ids, err := s.repo.FindUserIDs(ctx, lower, lower, phoneOf(identifier))
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		log.Printf("lupa kata sandi: tidak ada akun untuk isian yang dikirim (ip %s)", ip) // isian tidak dicetak
		return nil
	}
	return s.repo.UpsertPending(ctx, ids, passwordresetmodel.NewRequest{
		Identifier: identifier, Contact: optional(req.Contact), Note: optional(req.Note), IPAddress: ip, UserAgent: userAgent,
	}, time.Now())
}

// rolesFor = role akun yang permintaannya boleh dilihat & diproses actor.
func rolesFor(actorRole string) []string {
	if actorRole == loginmodel.RoleSuperAdmin {
		return []string{loginmodel.RoleSuperAdmin, loginmodel.RoleAdmin, loginmodel.RoleTutor, loginmodel.RoleStudent}
	}
	return []string{loginmodel.RoleTutor, loginmodel.RoleStudent}
}

// filterFrom menyusun filter daftar (default status pending, halaman 1, 20 per halaman).
func filterFrom(q passwordresetdto.ListQuery, actorRole string) passwordresetmodel.ListFilter {
	f := passwordresetmodel.ListFilter{Status: q.Status, Roles: rolesFor(actorRole), Search: strings.TrimSpace(q.Search), Page: q.Page, PerPage: q.PerPage}
	switch f.Status {
	case "":
		f.Status = passwordresetmodel.StatusPending
	case "all":
		f.Status = ""
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	return f
}

// List mengambil 1 halaman permintaan (terbaru dulu). Admin hanya melihat akun tutor & siswa.
func (s *PasswordResetService) List(ctx context.Context, q passwordresetdto.ListQuery, actorRole string) ([]passwordresetmodel.Request, passwordresetmodel.ListFilter, int, error) {
	f := filterFrom(q, actorRole)
	items, total, err := s.repo.List(ctx, f)
	return items, f, total, err
}

// Counts menghitung permintaan per status (search ikut, status diabaikan).
func (s *PasswordResetService) Counts(ctx context.Context, q passwordresetdto.ListQuery, actorRole string) (passwordresetmodel.Counts, error) {
	return s.repo.CountByStatus(ctx, filterFrom(q, actorRole))
}

// find mengambil 1 permintaan yang boleh diproses actor. Akun admin / super admin untuk admin →
// ErrForbidden; tidak ada → ErrNotFound.
func (s *PasswordResetService) find(ctx context.Context, id, actorRole string) (passwordresetmodel.Request, error) {
	if uuid.Validate(id) != nil {
		return passwordresetmodel.Request{}, passwordresetmodel.ErrNotFound
	}
	r, err := s.repo.Find(ctx, id)
	if err != nil {
		return r, err
	}
	if !usermodel.CanManage(actorRole, r.User.Role) {
		return r, passwordresetmodel.ErrForbidden
	}
	return r, nil
}

// Get mengambil 1 permintaan.
func (s *PasswordResetService) Get(ctx context.Context, id, actorRole string) (passwordresetmodel.Request, error) {
	return s.find(ctx, id, actorRole)
}

// Resolve mereset kata sandi pemilik permintaan (semua sesinya dicabut) dan menandai permintaan selesai.
// Admin menyampaikan kata sandi baru ke user secara langsung (WA / telepon).
func (s *PasswordResetService) Resolve(ctx context.Context, id string, req passwordresetdto.ResolveRequest, actorID, actorRole string) (passwordresetmodel.Request, error) {
	r, err := s.find(ctx, id, actorRole)
	if err != nil {
		return r, err
	}
	if r.Status != passwordresetmodel.StatusPending {
		return r, passwordresetmodel.ErrNotPending
	}
	hash, err := loginservice.HashPassword(req.NewPassword)
	if err != nil {
		return r, err
	}
	if err := s.repo.Resolve(ctx, id, r.User.ID, hash, optional(req.Note), actorID, time.Now()); err != nil {
		return r, err
	}
	return s.repo.Find(ctx, id)
}

// Reject menolak permintaan yang masih menunggu (kata sandi tidak berubah).
func (s *PasswordResetService) Reject(ctx context.Context, id string, req passwordresetdto.RejectRequest, actorID, actorRole string) (passwordresetmodel.Request, error) {
	if _, err := s.find(ctx, id, actorRole); err != nil {
		return passwordresetmodel.Request{}, err
	}
	if err := s.repo.Reject(ctx, id, optional(req.Note), actorID, time.Now()); err != nil {
		return passwordresetmodel.Request{}, err
	}
	return s.repo.Find(ctx, id)
}
