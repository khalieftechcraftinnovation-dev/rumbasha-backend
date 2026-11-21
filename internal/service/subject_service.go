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
	// repo bertipe interface, bukan struct tertentu. Jadi service tidak peduli
	// data disimpan di memori atau PostgreSQL.
	repo repository.SubjectRepository
}

// NewSubjectService membuat SubjectService. Dipanggil sekali di main.go.
func NewSubjectService(repo repository.SubjectRepository) *SubjectService {
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
		Code:      strings.ToUpper(strings.TrimSpace(req.Code)), // aturan bisnis: kode selalu huruf besar
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
	return s.repo.FindByID(ctx, id)
}

// List mengambil daftar mata pelajaran per halaman, beserta total datanya.
func (s *SubjectService) List(ctx context.Context, page, perPage int) ([]model.Subject, int, error) {
	offset := (page - 1) * perPage
	return s.repo.FindAll(ctx, offset, perPage)
}
