// Package service adalah layer SERVICE: tempat semua aturan bisnis.
// Service tidak tahu soal HTTP (Gin) maupun SQL. Data dibaca/ditulis
// lewat repository.
package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository"
)

// SubjectService berisi logika bisnis untuk mata pelajaran.
type SubjectService struct {
	repo *repository.SubjectRepository
}

// NewSubjectService membuat SubjectService. Dipanggil sekali di main.go.
func NewSubjectService(repo *repository.SubjectRepository) *SubjectService {
	return &SubjectService{repo: repo}
}

// Create membuat mata pelajaran baru.
//
// Catatan Go: fungsi bisa mengembalikan lebih dari satu nilai. Polanya selalu
// (hasil, error): kalau error tidak nil, hasilnya diabaikan.
func (s *SubjectService) Create(ctx context.Context, req dto.CreateSubjectRequest) (model.Subject, error) {
	now := time.Now().UTC()

	subject := model.Subject{
		ID:        uuid.NewString(),
		Code:      normalizeCode(req.Code),
		Name:      strings.TrimSpace(req.Name),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, subject); err != nil {
		return model.Subject{}, err
	}
	return subject, nil
}

// GetByID mengambil satu mata pelajaran berdasarkan ID.
func (s *SubjectService) GetByID(ctx context.Context, id string) (model.Subject, error) {
	if !isValidID(id) {
		return model.Subject{}, model.ErrSubjectNotFound
	}
	return s.repo.FindByID(ctx, id)
}

// List mengambil daftar mata pelajaran per halaman, beserta total datanya.
func (s *SubjectService) List(ctx context.Context, page, perPage int) ([]model.Subject, int, error) {
	offset := (page - 1) * perPage
	return s.repo.FindAll(ctx, offset, perPage)
}

// Update mengubah data mata pelajaran, lalu mengembalikan data terbarunya.
func (s *SubjectService) Update(ctx context.Context, id string, req dto.UpdateSubjectRequest) (model.Subject, error) {
	if !isValidID(id) {
		return model.Subject{}, model.ErrSubjectNotFound
	}

	subject := model.Subject{
		ID:        id,
		Code:      normalizeCode(req.Code),
		Name:      strings.TrimSpace(req.Name),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.Update(ctx, subject); err != nil {
		return model.Subject{}, err
	}

	// Ambil ulang dari database supaya created_at ikut terisi di response.
	return s.repo.FindByID(ctx, id)
}

// Delete menghapus mata pelajaran.
func (s *SubjectService) Delete(ctx context.Context, id string) error {
	if !isValidID(id) {
		return model.ErrSubjectNotFound
	}
	return s.repo.Delete(ctx, id)
}

// normalizeCode adalah aturan bisnis: kode selalu huruf besar tanpa spasi di ujung.
func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// isValidID mengecek apakah id berformat UUID.
// Kolom id di database bertipe UUID, jadi query dengan id yang bukan UUID
// akan error di PostgreSQL. Lebih baik langsung dianggap "tidak ditemukan".
func isValidID(id string) bool {
	return uuid.Validate(id) == nil
}
