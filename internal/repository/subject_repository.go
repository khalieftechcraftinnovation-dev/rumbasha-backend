// Package repository adalah layer REPOSITORY: satu-satunya layer yang
// menyentuh database.
package repository

import (
	"context"
	"sync"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
)

// SubjectRepository adalah kontrak (interface) untuk akses data subject.
//
// Service hanya mengenal interface ini, bukan implementasinya. Jadi nanti
// SubjectMemoryRepository bisa diganti versi PostgreSQL tanpa mengubah service.
//
// Catatan Go: sebuah struct otomatis dianggap memenuhi interface kalau punya
// semua method-nya. Tidak perlu menulis "implements" seperti di PHP/Java.
type SubjectRepository interface {
	Create(ctx context.Context, subject model.Subject) error
	FindByID(ctx context.Context, id string) (model.Subject, error)
	FindAll(ctx context.Context, offset, limit int) ([]model.Subject, int, error)
}

// SubjectMemoryRepository menyimpan data di memori (RAM).
// Data hilang setiap server restart. Ini hanya sementara sampai PostgreSQL dipasang.
type SubjectMemoryRepository struct {
	mu       sync.Mutex // kunci agar data aman diakses banyak request bersamaan
	subjects []model.Subject
}

// NewSubjectMemoryRepository membuat repository in-memory yang masih kosong.
func NewSubjectMemoryRepository() *SubjectMemoryRepository {
	return &SubjectMemoryRepository{}
}

// Create menyimpan subject baru. Kode subject tidak boleh kembar.
func (r *SubjectMemoryRepository) Create(ctx context.Context, subject model.Subject) error {
	r.mu.Lock()
	defer r.mu.Unlock() // defer: dijalankan otomatis saat fungsi selesai

	for _, existing := range r.subjects {
		if existing.Code == subject.Code {
			return model.ErrSubjectCodeTaken
		}
	}

	r.subjects = append(r.subjects, subject)
	return nil
}

// FindByID mencari subject berdasarkan ID.
func (r *SubjectMemoryRepository) FindByID(ctx context.Context, id string) (model.Subject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, subject := range r.subjects {
		if subject.ID == id {
			return subject, nil
		}
	}
	return model.Subject{}, model.ErrSubjectNotFound
}

// FindAll mengambil data mulai dari posisi offset, sebanyak limit,
// beserta jumlah total semua data (untuk pagination).
func (r *SubjectMemoryRepository) FindAll(ctx context.Context, offset, limit int) ([]model.Subject, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := len(r.subjects)
	if offset >= total {
		return []model.Subject{}, total, nil
	}

	end := min(offset+limit, total)

	// Salin datanya supaya pemanggil tidak bisa mengubah isi repository.
	result := make([]model.Subject, end-offset)
	copy(result, r.subjects[offset:end])

	return result, total, nil
}
