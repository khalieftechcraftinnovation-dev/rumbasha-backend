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
	"strings"
	"time"

	"github.com/google/uuid"
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

// insertColumns adalah kolom yang diisi saat menyimpan soal (urutannya sama dengan questionArgs):
// id + kolom isi soal (contentArgs) + created_by.
const insertColumns = `
	id, type, question_text, formula, image_id, image_url, table_data, explanation, display_order,
	education_level_id, grade_id, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
	difficulty, used_for, status, answer, created_by`

// insertRowPlaceholders adalah placeholder 1 baris. Kolom JSONB dikirim sebagai teks JSON
// lalu di-cast (?::jsonb), supaya SQL-nya sama persis dengan yang bisa dicoba di DBeaver.
const insertRowPlaceholders = `(?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?::jsonb, ?)`

// updateContentSQL adalah bagian SET untuk kolom isi soal (urutannya sama dengan contentArgs).
const updateContentSQL = `
	type = ?, question_text = ?, formula = ?, image_id = ?, image_url = ?, table_data = ?::jsonb,
	explanation = ?, display_order = ?::jsonb,
	education_level_id = ?, grade_id = ?, subject_id = ?, subject_field_id = ?,
	chapter_id = ?, subchapter_id = ?, topic_id = ?,
	difficulty = ?, used_for = ?::jsonb, status = ?, answer = ?::jsonb`

// CreateMany menyimpan semua soal dengan 1 query INSERT banyak baris. [Native SQL]
// Karena hanya 1 query, hasilnya otomatis "semua tersimpan atau tidak sama sekali".
//
//	INSERT INTO questions (id, type, ...) VALUES (...), (...), (...)
//	RETURNING id, code, created_at, updated_at
//
// ID dibuat di Go (bukan di database) supaya baris hasil RETURNING bisa dicocokkan ke
// soalnya lewat ID. PostgreSQL tidak menjamin urutan baris RETURNING sama dengan VALUES.
// Setelah selesai, ID, Code, CreatedAt, UpdatedAt di setiap soal sudah terisi.
func (r *QuestionRepository) CreateMany(ctx context.Context, questions []questionmodel.Question) error {
	rows := make([]string, 0, len(questions))
	args := make([]any, 0, len(questions)*21)
	indexByID := make(map[string]int, len(questions)) // ID → posisi soal di slice

	for i := range questions {
		questions[i].ID = uuid.NewString()
		indexByID[questions[i].ID] = i

		rowArgs, err := questionArgs(questions[i])
		if err != nil {
			return fmt.Errorf("soal ke-%d: %w", i, err)
		}
		rows = append(rows, insertRowPlaceholders)
		args = append(args, rowArgs...)
	}

	var saved []struct {
		ID        string
		Code      string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	err := r.db.WithContext(ctx).
		Raw(`INSERT INTO questions (`+insertColumns+`)
			VALUES `+strings.Join(rows, ",\n")+`
			RETURNING id, code, created_at, updated_at`, args...).
		Scan(&saved).Error
	if err != nil {
		return err
	}
	if len(saved) != len(questions) {
		return fmt.Errorf("jumlah soal tersimpan %d, seharusnya %d", len(saved), len(questions))
	}

	for _, row := range saved {
		q := &questions[indexByID[row.ID]]
		q.Code = row.Code
		q.CreatedAt = row.CreatedAt
		q.UpdatedAt = row.UpdatedAt
	}
	return nil
}

// questionArgs menyusun nilai 1 baris INSERT sesuai urutan insertColumns.
func questionArgs(q questionmodel.Question) ([]any, error) {
	content, err := contentArgs(q)
	if err != nil {
		return nil, err
	}
	args := make([]any, 0, len(content)+2)
	args = append(args, q.ID)
	args = append(args, content...)
	return append(args, q.CreatedBy), nil
}

// contentArgs menyusun nilai kolom isi soal (type s/d answer), dipakai INSERT & UPDATE.
func contentArgs(q questionmodel.Question) ([]any, error) {
	var tableJSON *string // NULL kalau soal tidak punya tabel
	if q.Table != nil {
		b, err := json.Marshal(q.Table)
		if err != nil {
			return nil, fmt.Errorf("gagal mengubah table ke JSON: %w", err)
		}
		s := string(b)
		tableJSON = &s
	}

	displayOrderJSON, err := json.Marshal(q.DisplayOrder)
	if err != nil {
		return nil, fmt.Errorf("gagal mengubah display_order ke JSON: %w", err)
	}
	usedForJSON, err := json.Marshal(q.UsedFor)
	if err != nil {
		return nil, fmt.Errorf("gagal mengubah used_for ke JSON: %w", err)
	}

	return []any{
		q.Type, q.QuestionText, q.Formula, q.ImageID, q.ImageURL, tableJSON, q.Explanation, string(displayOrderJSON),
		q.EducationLevelID, q.GradeID, q.SubjectID, q.SubjectFieldID, q.ChapterID, q.SubchapterID, q.TopicID,
		q.Difficulty, string(usedForJSON), q.Status, string(q.Answer),
	}, nil
}

// questionDetailQuery mengambil 1 soal lengkap dengan nama placement & nama pembuatnya. [Native SQL]
//
// master_data_items di-JOIN 7 kali dengan alias berbeda (satu per tingkat placement).
// users di-JOIN 2 kali: "u" untuk pembuat, "uu" untuk yang terakhir mengubah.
// Semua JOIN memakai primary key (id), jadi tetap cepat. Sub bab, topik, dan user
// memakai LEFT JOIN karena boleh kosong. Kolom JSONB diambil sebagai teks (::text).
//
// Bisa dicoba langsung di DBeaver dengan mengganti ? menjadi id soal.
const questionDetailQuery = `
	SELECT
		q.id, q.code, q.type, q.question_text, q.formula, q.image_id, q.image_url,
		q.table_data::text    AS table_data,
		q.explanation,
		q.display_order::text AS display_order,
		q.education_level_id, level.name   AS education_level_name,
		q.grade_id,           grade.name   AS grade_name,
		q.subject_id,         subject.name AS subject_name,
		q.subject_field_id,   field.name   AS subject_field_name,
		q.chapter_id,         chapter.name AS chapter_name,
		q.subchapter_id,      sub.name     AS subchapter_name,
		q.topic_id,           topic.name   AS topic_name,
		q.difficulty,
		q.used_for::text      AS used_for,
		q.status,
		q.answer::text        AS answer,
		q.created_by,         u.real_name  AS created_by_name,
		q.updated_by,         uu.real_name AS updated_by_name,
		q.created_at, q.updated_at
	FROM questions q
	JOIN      master_data_items AS level   ON level.id   = q.education_level_id
	JOIN      master_data_items AS grade   ON grade.id   = q.grade_id
	JOIN      master_data_items AS subject ON subject.id = q.subject_id
	JOIN      master_data_items AS field   ON field.id   = q.subject_field_id
	JOIN      master_data_items AS chapter ON chapter.id = q.chapter_id
	LEFT JOIN master_data_items AS sub     ON sub.id     = q.subchapter_id
	LEFT JOIN master_data_items AS topic   ON topic.id   = q.topic_id
	LEFT JOIN users             AS u       ON u.id       = q.created_by
	LEFT JOIN users             AS uu      ON uu.id      = q.updated_by
	WHERE q.id = ? AND q.deleted_at IS NULL`

// questionDetailRow menampung hasil questionDetailQuery apa adanya (JSON masih berupa teks).
type questionDetailRow struct {
	ID, Code, Type, QuestionText string
	Formula, ImageID, ImageURL   *string
	TableData                    *string
	Explanation, DisplayOrder    string

	EducationLevelID, EducationLevelName string
	GradeID, GradeName                   string
	SubjectID, SubjectName               string
	SubjectFieldID, SubjectFieldName     string
	ChapterID, ChapterName               string
	SubchapterID, SubchapterName         *string
	TopicID, TopicName                   *string

	Difficulty, UsedFor, Status, Answer string
	CreatedBy                           string
	CreatedByName                       *string
	UpdatedBy, UpdatedByName            *string
	CreatedAt, UpdatedAt                time.Time
}

// FindDetailByID mengambil 1 soal lengkap (termasuk kunci jawaban & pembahasan).
// Mengembalikan questionmodel.ErrQuestionNotFound kalau tidak ada atau sudah dihapus.
func (r *QuestionRepository) FindDetailByID(ctx context.Context, id string) (questionmodel.QuestionDetail, error) {
	var rows []questionDetailRow
	if err := r.db.WithContext(ctx).Raw(questionDetailQuery, id).Scan(&rows).Error; err != nil {
		return questionmodel.QuestionDetail{}, err
	}
	if len(rows) == 0 {
		return questionmodel.QuestionDetail{}, questionmodel.ErrQuestionNotFound
	}
	row := rows[0]

	// Ubah kolom JSON (teks) ke struct Go.
	detail := questionmodel.QuestionDetail{
		Question: questionmodel.Question{
			ID: row.ID, Code: row.Code, Type: row.Type, QuestionText: row.QuestionText,
			Formula: row.Formula, ImageID: row.ImageID, ImageURL: row.ImageURL, Explanation: row.Explanation,
			EducationLevelID: row.EducationLevelID, GradeID: row.GradeID, SubjectID: row.SubjectID,
			SubjectFieldID: row.SubjectFieldID, ChapterID: row.ChapterID,
			SubchapterID: row.SubchapterID, TopicID: row.TopicID,
			Difficulty: row.Difficulty, Status: row.Status,
			CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		},
		EducationLevelName: row.EducationLevelName,
		GradeName:          row.GradeName,
		SubjectName:        row.SubjectName,
		SubjectFieldName:   row.SubjectFieldName,
		ChapterName:        row.ChapterName,
		SubchapterName:     row.SubchapterName,
		TopicName:          row.TopicName,
		CreatedByName:      row.CreatedByName,
		UpdatedByName:      row.UpdatedByName,
	}

	if row.TableData != nil {
		detail.Table = &questionmodel.Table{}
		if err := json.Unmarshal([]byte(*row.TableData), detail.Table); err != nil {
			return questionmodel.QuestionDetail{}, fmt.Errorf("soal %s: table_data rusak: %w", row.Code, err)
		}
	}
	if err := json.Unmarshal([]byte(row.DisplayOrder), &detail.DisplayOrder); err != nil {
		return questionmodel.QuestionDetail{}, fmt.Errorf("soal %s: display_order rusak: %w", row.Code, err)
	}
	if err := json.Unmarshal([]byte(row.UsedFor), &detail.UsedFor); err != nil {
		return questionmodel.QuestionDetail{}, fmt.Errorf("soal %s: used_for rusak: %w", row.Code, err)
	}
	answer, err := questionmodel.CanonicalAnswer(row.Type, []byte(row.Answer))
	if err != nil {
		return questionmodel.QuestionDetail{}, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
	}
	detail.Answer = answer

	return detail, nil
}

// ExistsByID mengecek soal ada dan belum dihapus. [Native SQL, pakai primary key]
func (r *QuestionRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Raw(`SELECT EXISTS (SELECT 1 FROM questions WHERE id = ? AND deleted_at IS NULL)`, id).
		Scan(&exists).Error
	return exists, err
}

// Update mengganti seluruh isi soal (kecuali id, code, created_by, created_at). [Native SQL]
// updated_by & updated_at ikut diisi. Mengembalikan ErrQuestionNotFound kalau soal
// tidak ada / sudah dihapus.
func (r *QuestionRepository) Update(ctx context.Context, q questionmodel.Question) error {
	args, err := contentArgs(q) // urutan sama dengan updateContentSQL
	if err != nil {
		return err
	}
	args = append(args, q.UpdatedBy, q.ID)

	result := r.db.WithContext(ctx).Exec(`
		UPDATE questions SET `+updateContentSQL+`,
			updated_by = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`, args...)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return questionmodel.ErrQuestionNotFound
	}
	return nil
}

// SoftDeleteMany menghapus banyak soal sekaligus dengan soft delete (deleted_at & deleted_by diisi).
// [Native SQL, 1 query di dalam transaksi]
//
// Semua atau tidak sama sekali: kalau ada ID yang tidak ada / sudah dihapus, transaksi
// dibatalkan (ROLLBACK) dan tidak ada soal yang terhapus.
// Hasilnya adalah ID yang BERHASIL ditemukan; pemanggil membandingkannya dengan ids untuk
// mengetahui ID mana yang tidak ada. ok = true kalau semua terhapus.
func (r *QuestionRepository) SoftDeleteMany(ctx context.Context, ids []string, deletedBy string) (found []string, ok bool, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, false, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	err = tx.Raw(`
		UPDATE questions
		SET deleted_at = NOW(), deleted_by = ?, updated_at = NOW()
		WHERE id IN ? AND deleted_at IS NULL
		RETURNING id`, deletedBy, ids).
		Scan(&found).Error
	if err != nil {
		return nil, false, err
	}

	if len(found) != len(ids) {
		return found, false, nil // ROLLBACK lewat defer
	}
	if err := tx.Commit().Error; err != nil {
		return nil, false, err
	}
	return found, true, nil
}

// FindMasterDataByNames mengambil item hierarki mapel (jenjang s/d topik) yang namanya ada di
// names (tidak peka huruf besar/kecil). Dipakai upload Excel: 1 query untuk SEMUA baris file,
// lalu dicocokkan per baris di service. names harus sudah huruf kecil. [Native SQL]
func (r *QuestionRepository) FindMasterDataByNames(ctx context.Context, names []string) ([]questionmodel.MasterDataItem, error) {
	items := []questionmodel.MasterDataItem{}
	if len(names) == 0 {
		return items, nil
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, type, name, parent_id, subject_id, education_level_id, grade_id, status
			FROM master_data_items
			WHERE type IN ('education_level', 'grade', 'subject', 'subject_field', 'chapter', 'subchapter', 'topic')
			  AND LOWER(name) IN ?
			  AND deleted_at IS NULL`, names).
		Scan(&items).Error
	return items, err
}
