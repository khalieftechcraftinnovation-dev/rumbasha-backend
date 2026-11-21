// Package subject (service) berisi aturan bisnis untuk mata pelajaran.
// Service tidak tahu soal HTTP (Gin) maupun SQL. Data dibaca/ditulis lewat repository.
//
// Di package lain, import dengan alias subjectservice:
//
//	subjectservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subject"
package subject

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	subjectdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subject"
	subjectmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subject"
	subjectrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/subject"
)

// SubjectService berisi logika bisnis untuk mata pelajaran.
type SubjectService struct {
	repo *subjectrepo.SubjectRepository
}

// NewSubjectService membuat SubjectService. Dipanggil sekali di main.go.
func NewSubjectService(repo *subjectrepo.SubjectRepository) *SubjectService {
	return &SubjectService{repo: repo}
}

// Create membuat mata pelajaran baru.
//
// Catatan Go: fungsi bisa mengembalikan lebih dari satu nilai. Polanya selalu
// (hasil, error): kalau error tidak nil, hasilnya diabaikan.
func (s *SubjectService) Create(ctx context.Context, req subjectdto.CreateSubjectRequest) (subjectmodel.Subject, error) {
	now := time.Now().UTC()

	subject := subjectmodel.Subject{
		ID:        uuid.NewString(),
		Code:      normalizeCode(req.Code),
		Name:      strings.TrimSpace(req.Name),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, subject); err != nil {
		return subjectmodel.Subject{}, err
	}
	return subject, nil
}

// GetByID mengambil satu mata pelajaran berdasarkan ID.
func (s *SubjectService) GetByID(ctx context.Context, id string) (subjectmodel.Subject, error) {
	if !isValidID(id) {
		return subjectmodel.Subject{}, subjectmodel.ErrSubjectNotFound
	}
	return s.repo.FindByID(ctx, id)
}

// List mengambil daftar mata pelajaran per halaman, beserta total datanya.
func (s *SubjectService) List(ctx context.Context, page, perPage int) ([]subjectmodel.Subject, int, error) {
	offset := (page - 1) * perPage
	return s.repo.FindAll(ctx, offset, perPage)
}

// Update mengubah data mata pelajaran, lalu mengembalikan data terbarunya.
func (s *SubjectService) Update(ctx context.Context, id string, req subjectdto.UpdateSubjectRequest) (subjectmodel.Subject, error) {
	if !isValidID(id) {
		return subjectmodel.Subject{}, subjectmodel.ErrSubjectNotFound
	}

	subject := subjectmodel.Subject{
		ID:        id,
		Code:      normalizeCode(req.Code),
		Name:      strings.TrimSpace(req.Name),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.Update(ctx, subject); err != nil {
		return subjectmodel.Subject{}, err
	}

	// Ambil ulang dari database supaya created_at ikut terisi di response.
	return s.repo.FindByID(ctx, id)
}

// Delete menghapus mata pelajaran.
func (s *SubjectService) Delete(ctx context.Context, id string) error {
	if !isValidID(id) {
		return subjectmodel.ErrSubjectNotFound
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
