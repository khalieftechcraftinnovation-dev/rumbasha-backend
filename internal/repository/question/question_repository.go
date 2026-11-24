// Package question (repository) mengakses tabel questions, serta membaca
// master_data_items & media_assets untuk validasi soal.
//
// Di package lain, import dengan alias questionrepo:
//
//	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
package question

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// QuestionRepository mengakses tabel questions.
type QuestionRepository struct {
	db *gorm.DB
}

// NewQuestionRepository membuat QuestionRepository. Dipanggil sekali di main.go.
func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// FindMasterDataByIDs mengambil item master data (bab, sub bab, topik) untuk dicek placement-nya.
// Item yang dihapus (soft delete) tidak ikut. Hasilnya berupa map id → item. [Native SQL, 1 tabel]
//
//	SELECT id, type, parent_id, subject_id, education_level_id, grade_id, status
//	FROM master_data_items WHERE id IN (...) AND deleted_at IS NULL
func (r *QuestionRepository) FindMasterDataByIDs(ctx context.Context, ids []string) (map[string]questionmodel.MasterDataItem, error) {
	var items []questionmodel.MasterDataItem
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, type, parent_id, subject_id, education_level_id, grade_id, status
			FROM master_data_items
			WHERE id IN ? AND deleted_at IS NULL`, ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]questionmodel.MasterDataItem, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

// FindExistingImageIDs mengembalikan ID gambar yang benar-benar ada (belum dihapus, tipe image/*).
// Hasilnya berupa set (map id → true) supaya mudah dicek satu per satu. [Native SQL]
func (r *QuestionRepository) FindExistingImageIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	var found []string
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id
			FROM media_assets
			WHERE id IN ? AND deleted_at IS NULL AND mime_type LIKE 'image/%'`, ids).
		Scan(&found).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(found))
	for _, id := range found {
		result[id] = true
	}
	return result, nil
}

// Create menyimpan soal baru, lalu mengisi ID, Code, CreatedAt, UpdatedAt dari database. [Native SQL]
//
// Kolom JSONB dikirim sebagai teks JSON lalu di-cast (?::jsonb), supaya SQL-nya
// sama persis dengan yang bisa dicoba di DBeaver.
func (r *QuestionRepository) Create(ctx context.Context, q *questionmodel.Question) error {
	var tableJSON *string // NULL kalau soal tidak punya tabel
	if q.Table != nil {
		b, err := json.Marshal(q.Table)
		if err != nil {
			return fmt.Errorf("gagal mengubah table ke JSON: %w", err)
		}
		s := string(b)
		tableJSON = &s
	}

	displayOrderJSON, err := json.Marshal(q.DisplayOrder)
	if err != nil {
		return fmt.Errorf("gagal mengubah display_order ke JSON: %w", err)
	}
	usedForJSON, err := json.Marshal(q.UsedFor)
	if err != nil {
		return fmt.Errorf("gagal mengubah used_for ke JSON: %w", err)
	}

	var row struct {
		ID        string
		Code      string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	err = r.db.WithContext(ctx).
		Raw(`
			INSERT INTO questions (
				type, question_text, formula, image_id, table_data, explanation, display_order,
				education_level_id, grade_id, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
				difficulty, used_for, status, answer, created_by
			) VALUES (
				?, ?, ?, ?, ?::jsonb, ?, ?::jsonb,
				?, ?, ?, ?, ?, ?, ?,
				?, ?::jsonb, ?, ?::jsonb, ?
			)
			RETURNING id, code, created_at, updated_at`,
			q.Type, q.QuestionText, q.Formula, q.ImageID, tableJSON, q.Explanation, string(displayOrderJSON),
			q.EducationLevelID, q.GradeID, q.SubjectID, q.SubjectFieldID, q.ChapterID, q.SubchapterID, q.TopicID,
			q.Difficulty, string(usedForJSON), q.Status, string(q.Answer), q.CreatedBy,
		).
		Scan(&row).Error
	if err != nil {
		return err
	}

	q.ID = row.ID
	q.Code = row.Code
	q.CreatedAt = row.CreatedAt
	q.UpdatedAt = row.UpdatedAt
	return nil
}
