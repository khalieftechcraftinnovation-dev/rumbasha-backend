// Package landingpage (service) berisi aturan bisnis Landing Page (BRD 3.19, BR-028).
//
//   - Admin mengubah section (isi, tampil, urutan) = draft. Tidak langsung tayang.
//   - Publish = salin section yang tampil ke snapshot halaman publik.
//   - Halaman publik dibaca dari snapshot (1 baris) dan disimpan di memori (cache) selama
//     publicCacheTTL, supaya request publik yang ramai tidak selalu ke database.
//
// Di package lain, import dengan alias landingpageservice:
//
//	landingpageservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/site/landingpage"
package landingpage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	landingpagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/site/landingpage"
	landingpagerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/site/landingpage"
)

// publicCacheTTL = lama isi halaman publik disimpan di memori. Publish langsung menghapus cache
// halaman itu, jadi TTL ini hanya berpengaruh kalau API dijalankan lebih dari 1 server.
const publicCacheTTL = time.Minute

// maxContentBytes = ukuran maksimal isi 1 section (JSON).
const maxContentBytes = 200 << 10 // 200 KB

// LandingPageService berisi logika bisnis Landing Page.
type LandingPageService struct {
	repo *landingpagerepo.LandingPageRepository

	mu    sync.RWMutex
	cache map[string]cachedPage // kode halaman → isi publik
}

type cachedPage struct {
	page      landingpagemodel.PublishedPage
	expiresAt time.Time
}

// NewLandingPageService membuat LandingPageService. Dipanggil sekali di main.go.
func NewLandingPageService(repo *landingpagerepo.LandingPageRepository) *LandingPageService {
	return &LandingPageService{repo: repo, cache: map[string]cachedPage{}}
}

// ListPages mengambil semua tab Landing Page beserta info publish & jumlah perubahan tertunda.
func (s *LandingPageService) ListPages(ctx context.Context) ([]landingpagemodel.Page, error) {
	return s.repo.ListPages(ctx)
}

// GetDraft mengambil 1 halaman untuk panel admin: info halaman, section draft, dan perubahan
// yang belum dipublish.
func (s *LandingPageService) GetDraft(ctx context.Context, pageCode string) (landingpagemodel.Page, []landingpagemodel.Section, []landingpagemodel.Change, error) {
	page, err := s.repo.FindPage(ctx, pageCode)
	if err != nil {
		return page, nil, nil, err
	}
	sections, err := s.repo.ListSections(ctx, pageCode)
	if err != nil {
		return page, nil, nil, fmt.Errorf("gagal membaca section %s: %w", pageCode, err)
	}
	changes, err := s.repo.ListPendingChanges(ctx, pageCode)
	if err != nil {
		return page, nil, nil, fmt.Errorf("gagal membaca perubahan %s: %w", pageCode, err)
	}
	return page, sections, changes, nil
}

// UpdateContent mengganti isi 1 section (draft). Isi harus object JSON, maksimal 200 KB.
func (s *LandingPageService) UpdateContent(ctx context.Context, pageCode, key string, content json.RawMessage, userID string) error {
	trimmed := bytes.TrimSpace(content)
	switch {
	case len(trimmed) == 0 || trimmed[0] != '{':
		return validationError("content", "harus berupa object JSON")
	case len(trimmed) > maxContentBytes:
		return validationError("content", "maksimal 200 KB")
	}

	// Rapikan (compact) supaya isi yang sama dengan spasi berbeda tidak dianggap perubahan.
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return validationError("content", "format JSON tidak valid")
	}

	if err := s.ensurePage(ctx, pageCode); err != nil {
		return err
	}
	_, err := s.repo.UpdateSectionContent(ctx, pageCode, key, compact.Bytes(), userID)
	return err
}

// SetVisible mengubah saklar "Tampil" 1 section (draft).
func (s *LandingPageService) SetVisible(ctx context.Context, pageCode, key string, visible bool, userID string) error {
	if err := s.ensurePage(ctx, pageCode); err != nil {
		return err
	}
	_, err := s.repo.SetSectionVisible(ctx, pageCode, key, visible, userID)
	return err
}

// Reorder mengatur ulang urutan section (draft). keys wajib berisi SEMUA section halaman,
// masing-masing tepat 1 kali.
func (s *LandingPageService) Reorder(ctx context.Context, pageCode string, keys []string, userID string) error {
	if err := s.ensurePage(ctx, pageCode); err != nil {
		return err
	}
	sections, err := s.repo.ListSections(ctx, pageCode)
	if err != nil {
		return fmt.Errorf("gagal membaca section %s: %w", pageCode, err)
	}

	known := make(map[string]bool, len(sections))
	for _, sec := range sections {
		known[sec.Key] = true
	}
	errs := []landingpagemodel.FieldError{}
	seen := map[string]bool{}
	for i, key := range keys {
		field := fmt.Sprintf("keys[%d]", i)
		switch {
		case !known[key]:
			errs = append(errs, landingpagemodel.FieldError{Field: field, Message: "section " + key + " tidak ada di halaman ini"})
		case seen[key]:
			errs = append(errs, landingpagemodel.FieldError{Field: field, Message: "tidak boleh ada yang sama"})
		}
		seen[key] = true
	}
	for _, sec := range sections {
		if !seen[sec.Key] {
			errs = append(errs, landingpagemodel.FieldError{Field: "keys", Message: "section " + sec.Key + " belum dicantumkan"})
		}
	}
	if len(errs) > 0 {
		return &landingpagemodel.ValidationError{Fields: errs}
	}

	_, err = s.repo.ReorderSections(ctx, pageCode, keys, userID)
	return err
}

// Publish menayangkan draft ke halaman publik, lalu menghapus cache halaman itu.
func (s *LandingPageService) Publish(ctx context.Context, pageCode, userID string) (landingpagemodel.PublishResult, error) {
	result, err := s.repo.Publish(ctx, pageCode, userID)
	if err != nil {
		return result, err
	}
	s.mu.Lock()
	delete(s.cache, pageCode)
	s.mu.Unlock()
	return result, nil
}

// GetPublished mengambil isi halaman publik (dari cache memori kalau masih berlaku).
func (s *LandingPageService) GetPublished(ctx context.Context, pageCode string) (landingpagemodel.PublishedPage, error) {
	now := time.Now()
	s.mu.RLock()
	cached, ok := s.cache[pageCode]
	s.mu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return cached.page, nil
	}

	page, err := s.repo.FindPublished(ctx, pageCode)
	if err != nil {
		return page, err
	}
	s.mu.Lock()
	s.cache[pageCode] = cachedPage{page: page, expiresAt: now.Add(publicCacheTTL)}
	s.mu.Unlock()
	return page, nil
}

// ensurePage memastikan halaman ada, supaya kode halaman yang salah dibalas "halaman tidak
// ditemukan" (bukan "section tidak ditemukan").
func (s *LandingPageService) ensurePage(ctx context.Context, pageCode string) error {
	_, err := s.repo.FindPage(ctx, pageCode)
	return err
}

func validationError(field, message string) error {
	return &landingpagemodel.ValidationError{Fields: []landingpagemodel.FieldError{{Field: field, Message: message}}}
}
