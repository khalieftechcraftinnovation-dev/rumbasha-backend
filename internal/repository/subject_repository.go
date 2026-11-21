// Package repository adalah layer REPOSITORY: satu-satunya layer yang
// menyentuh database.
//
// Ada 2 gaya query di file ini, pilih sesuai kebutuhan:
//   - GORM      → untuk insert, update, delete, dan get sederhana
//   - Native SQL → db.Raw(...) / db.Exec(...) untuk query yang lebih kompleks
//     (JOIN, agregasi, laporan). SQL-nya bisa langsung dicoba di DBeaver.
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
)

// SubjectRepository mengakses tabel subjects.
type SubjectRepository struct {
	db *gorm.DB
}

// NewSubjectRepository membuat SubjectRepository. Dipanggil sekali di main.go.
func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
	return &SubjectRepository{db: db}
}

// Create menyimpan subject baru. [GORM]
//
// SQL yang dijalankan kira-kira:
//
//	INSERT INTO subjects (id, code, name, created_at, updated_at) VALUES (...)
func (r *SubjectRepository) Create(ctx context.Context, subject model.Subject) error {
	err := r.db.WithContext(ctx).Create(&subject).Error

	// Kolom code punya constraint UNIQUE. Kalau kembar, PostgreSQL menolak,
	// dan GORM mengubahnya menjadi gorm.ErrDuplicatedKey.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.ErrSubjectCodeTaken
	}
	return err
}

// FindByID mencari satu subject berdasarkan ID. [GORM]
//
//	SELECT * FROM subjects WHERE id = ? ORDER BY id LIMIT 1
func (r *SubjectRepository) FindByID(ctx context.Context, id string) (model.Subject, error) {
	var subject model.Subject
	err := r.db.WithContext(ctx).First(&subject, "id = ?", id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Subject{}, model.ErrSubjectNotFound
	}
	return subject, err
}

// FindAll mengambil daftar subject per halaman, beserta jumlah total datanya. [Native SQL]
func (r *SubjectRepository) FindAll(ctx context.Context, offset, limit int) ([]model.Subject, int, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM subjects`).
		Scan(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Tanda ? diganti dengan nilai limit & offset secara aman (anti SQL injection).
	// JANGAN pernah menyambung string input user langsung ke dalam SQL.
	subjects := []model.Subject{}
	err = r.db.WithContext(ctx).
		Raw(`
			SELECT id, code, name, created_at, updated_at
			FROM subjects
			ORDER BY created_at DESC, id
			LIMIT ? OFFSET ?`, limit, offset).
		Scan(&subjects).Error
	if err != nil {
		return nil, 0, err
	}

	return subjects, int(total), nil
}

// Update mengubah code dan name sebuah subject. [GORM]
//
//	UPDATE subjects SET code = ?, name = ?, updated_at = ? WHERE id = ?
func (r *SubjectRepository) Update(ctx context.Context, subject model.Subject) error {
	result := r.db.WithContext(ctx).
		Model(&model.Subject{}).
		Where("id = ?", subject.ID).
		Updates(map[string]any{
			"code":       subject.Code,
			"name":       subject.Name,
			"updated_at": subject.UpdatedAt,
		})

	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return model.ErrSubjectCodeTaken
	}
	if result.Error != nil {
		return result.Error
	}
	// RowsAffected = jumlah baris yang berubah. 0 artinya ID tidak ada.
	if result.RowsAffected == 0 {
		return model.ErrSubjectNotFound
	}
	return nil
}

// Delete menghapus sebuah subject secara permanen. [GORM]
//
//	DELETE FROM subjects WHERE id = ?
func (r *SubjectRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.Subject{}, "id = ?", id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrSubjectNotFound
	}
	return nil
}
