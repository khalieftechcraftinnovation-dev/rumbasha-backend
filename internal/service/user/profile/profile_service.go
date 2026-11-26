// Package profile (service) berisi logika halaman Profil siswa:
//
//   - lihat profil: data diri, paket aktif (sisa hari, fitur), coin, statistik (peringkat Daily Quiz
//     bulan ini di kelasnya, point, quiz selesai bulan ini);
//   - ubah data diri: nama lengkap, email, no HP, tanggal lahir (keputusan BD-014);
//   - kontak orang tua / wali: no HP (ditampilkan tersamar);
//   - foto profil: upload (PNG/JPG/WebP, maks 2 MB, dicek dari isi file) & hapus. File publik di
//     STORAGE_DIR/images/avatar/<tahun>/<bulan>/<uuid>.<ext>; foto lama dihapus setelah diganti.
//
// Di package lain, import dengan alias profileservice:
//
//	profileservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user/profile"
package profile

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	_ "image/jpeg" // daftarkan decoder JPEG untuk image.DecodeConfig
	_ "image/png"  // daftarkan decoder PNG
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // daftarkan decoder WebP

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	profiledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user/profile"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/subscription"
	profilemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user/profile"
	profilerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user/profile"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/storage"
)

// MaxAvatarSize = ukuran maksimal foto profil.
const MaxAvatarSize = 2 << 20 // 2 MB

// avatarTypes = jenis foto yang diterima (dibaca dari ISI file) → ekstensi yang disimpan.
var avatarTypes = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp"}

// phoneFormat = no HP: angka, spasi, strip, tanda +, 8-20 karakter.
var phoneFormat = regexp.MustCompile(`^\+?[0-9][0-9 -]{6,18}[0-9]$`)

// ProfileService berisi logika Profil siswa.
type ProfileService struct {
	repo          *profilerepo.ProfileRepository
	subscriptions *subscriptionservice.SubscriptionService
	storage       *storage.Local
}

// NewProfileService membuat ProfileService. Dipanggil sekali di main.go.
func NewProfileService(repo *profilerepo.ProfileRepository, subscriptions *subscriptionservice.SubscriptionService, store *storage.Local) *ProfileService {
	return &ProfileService{repo: repo, subscriptions: subscriptions, storage: store}
}

// Summary mengambil data halaman Profil: data diri, akses hari ini, dan statistik bulan ini (WIB).
func (s *ProfileService) Summary(ctx context.Context, userID string) (profilemodel.Profile, subscriptionmodel.Access, profilemodel.Stats, error) {
	p, err := s.repo.Find(ctx, userID)
	if err != nil {
		return p, subscriptionmodel.Access{}, profilemodel.Stats{}, err
	}
	accessMap, err := s.subscriptions.AccessMany(ctx, []string{userID})
	if err != nil {
		return p, subscriptionmodel.Access{}, profilemodel.Stats{}, err
	}

	now := time.Now().In(clock.Jakarta)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, clock.Jakarta)
	var gradeID *string
	if p.Grade != nil {
		gradeID = &p.Grade.ID
	}
	stats, err := s.repo.Stats(ctx, userID, gradeID, from.Format("2006-01"), from, from.AddDate(0, 1, 0))
	return p, accessMap[userID], stats, err
}

// Update mengganti data diri (nama, email, no HP, tanggal lahir). Kesalahan dikumpulkan sekaligus.
func (s *ProfileService) Update(ctx context.Context, userID string, req profiledto.UpdateRequest) (profilemodel.Profile, error) {
	var errs []profilemodel.FieldError
	add := func(field, message string) {
		errs = append(errs, profilemodel.FieldError{Field: field, Message: message})
	}

	u := profilemodel.Update{RealName: strings.TrimSpace(req.Name), Email: optional(req.Email), Phone: optional(req.Phone)}
	if u.RealName == "" {
		add("name", "wajib diisi")
	}
	if u.Email != nil {
		lower := strings.ToLower(*u.Email)
		u.Email = &lower
	}
	if u.Phone != nil && !phoneFormat.MatchString(*u.Phone) {
		add("phone", "hanya angka, spasi, strip, dan + di depan (8-20 karakter)")
	}
	if req.DateOfBirth != nil {
		dob := profiledto.ParseDate(*req.DateOfBirth)
		if dob.Format(profiledto.DateLayout) > clock.TodayInJakarta(time.Now()) {
			add("date_of_birth", "tidak boleh di masa depan")
		}
		u.DateOfBirth = &dob
	}
	if len(errs) > 0 {
		return profilemodel.Profile{}, &profilemodel.ValidationError{Fields: errs}
	}
	if err := s.repo.Update(ctx, userID, u, time.Now()); err != nil {
		return profilemodel.Profile{}, err
	}
	return s.repo.Find(ctx, userID)
}

// UpdateParentContact mengganti no HP orang tua / wali (kosong = dihapus).
func (s *ProfileService) UpdateParentContact(ctx context.Context, userID string, req profiledto.ParentContactRequest) (profilemodel.Profile, error) {
	phone := optional(req.ParentPhone)
	if phone != nil && !phoneFormat.MatchString(*phone) {
		return profilemodel.Profile{}, &profilemodel.ValidationError{Fields: []profilemodel.FieldError{{
			Field: "parent_phone", Message: "hanya angka, spasi, strip, dan + di depan (8-20 karakter)",
		}}}
	}
	if err := s.repo.UpdateParentPhone(ctx, userID, phone, time.Now()); err != nil {
		return profilemodel.Profile{}, err
	}
	return s.repo.Find(ctx, userID)
}

// UploadAvatar menyimpan foto profil baru, lalu menghapus file foto lama. data nil = file terlalu
// besar (tidak dibaca handler).
func (s *ProfileService) UploadAvatar(ctx context.Context, userID string, size int64, data []byte) (profilemodel.Profile, error) {
	current, err := s.repo.Find(ctx, userID)
	if err != nil {
		return current, err
	}
	ext, problem := inspectAvatar(size, data)
	if problem != "" {
		return current, &profilemodel.ValidationError{Fields: []profilemodel.FieldError{{Field: "photo", Message: problem}}}
	}

	now := time.Now().In(clock.Jakarta)
	path := fmt.Sprintf("images/avatar/%04d/%02d/%s%s", now.Year(), now.Month(), uuid.NewString(), ext)
	url := s.storage.URL(path)
	if err := s.storage.Save(path, data); err != nil {
		return current, fmt.Errorf("gagal menyimpan foto profil: %w", err)
	}
	if err := s.repo.UpdateAvatar(ctx, userID, &path, &url, time.Now()); err != nil {
		_ = s.storage.Remove(path)
		return current, err
	}
	s.removeFile(current.AvatarPath)
	return s.repo.Find(ctx, userID)
}

// DeleteAvatar menghapus foto profil (data & file). Belum ada foto → tetap berhasil.
func (s *ProfileService) DeleteAvatar(ctx context.Context, userID string) (profilemodel.Profile, error) {
	current, err := s.repo.Find(ctx, userID)
	if err != nil {
		return current, err
	}
	if current.AvatarPath == nil {
		return current, nil
	}
	if err := s.repo.UpdateAvatar(ctx, userID, nil, nil, time.Now()); err != nil {
		return current, err
	}
	s.removeFile(current.AvatarPath)
	return s.repo.Find(ctx, userID)
}

// removeFile menghapus file foto lama. Gagal hapus hanya dicatat (data sudah benar).
func (s *ProfileService) removeFile(path *string) {
	if path == nil {
		return
	}
	if err := s.storage.Remove(*path); err != nil {
		log.Printf("peringatan: gagal menghapus foto profil lama %s: %v", *path, err)
	}
}

// inspectAvatar mengecek ukuran & isi file foto. Hasil: ekstensi file atau pesan kesalahan.
func inspectAvatar(size int64, data []byte) (string, string) {
	switch {
	case size > MaxAvatarSize || data == nil:
		return "", fmt.Sprintf("ukuran maksimal %d MB", MaxAvatarSize>>20)
	case len(data) == 0:
		return "", "file kosong"
	}
	ext, ok := avatarTypes[http.DetectContentType(data)]
	if !ok {
		return "", "harus gambar PNG, JPG, atau WebP"
	}
	if _, _, err := stdimage.DecodeConfig(bytes.NewReader(data)); err != nil {
		return "", "file gambar rusak atau tidak bisa dibaca"
	}
	return ext, ""
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
