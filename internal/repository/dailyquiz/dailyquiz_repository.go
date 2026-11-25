// Package dailyquiz (repository) mengakses tabel daily_quizzes & daily_quiz_questions, serta
// membaca users, master_data_items, questions, media_assets untuk generate Daily Quiz.
//
// Di package lain, import dengan alias dailyquizrepo:
//
//	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dailyquiz"
package dailyquiz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// DailyQuizRepository mengakses tabel Daily Quiz.
type DailyQuizRepository struct {
	db *gorm.DB
}

// NewDailyQuizRepository membuat DailyQuizRepository. Dipanggil sekali di main.go.
func NewDailyQuizRepository(db *gorm.DB) *DailyQuizRepository {
	return &DailyQuizRepository{db: db}
}

// FindStudent mengambil jenjang & kelas siswa. [Native SQL, primary key]
func (r *DailyQuizRepository) FindStudent(ctx context.Context, userID string) (dailyquizmodel.Student, error) {
	var rows []dailyquizmodel.Student
	err := r.db.WithContext(ctx).
		Raw(`SELECT education_level_id, grade_id FROM users WHERE id = ? AND deleted_at IS NULL`, userID).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return dailyquizmodel.Student{}, err
	}
	return rows[0], nil
}

// FindGradesInLevel mengambil semua kelas aktif dalam 1 jenjang, urut sort_order. [Native SQL, 1 tabel]
func (r *DailyQuizRepository) FindGradesInLevel(ctx context.Context, educationLevelID string) ([]dailyquizmodel.GradeOption, error) {
	grades := []dailyquizmodel.GradeOption{}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, code, name, parent_id
			FROM master_data_items
			WHERE type = 'grade' AND parent_id = ? AND status = 'active' AND deleted_at IS NULL
			ORDER BY sort_order, name`, educationLevelID).
		Scan(&grades).Error
	return grades, err
}

// FindMasterDataByIDs mengambil item master data (1 query), hasilnya map id → item. [Native SQL]
func (r *DailyQuizRepository) FindMasterDataByIDs(ctx context.Context, ids []string) (map[string]dailyquizmodel.MasterDataItem, error) {
	var items []dailyquizmodel.MasterDataItem
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, type, name, parent_id, grade_id, status
			FROM master_data_items
			WHERE id IN ? AND deleted_at IS NULL`, ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]dailyquizmodel.MasterDataItem, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

// PickRandomQuestionIDs mengambil maksimal f.Limit ID soal acak yang cocok untuk Daily Quiz:
// aktif, belum dihapus, bertanda daily_quiz, level (difficulty) sama, di bidang materi & kelas
// yang dipilih (dan bab/sub bab/topik kalau dipilih). Tipe soal boleh campur. [Native SQL, 1 tabel]
//
// Kalau hasilnya kurang dari f.Limit, berarti soal yang tersedia memang hanya sebanyak itu.
// ORDER BY random() cukup cepat: kandidat sudah dipersempit index bidang materi
// (questions_subject_field_idx) menjadi puluhan/ratusan soal saja.
func (r *DailyQuizRepository) PickRandomQuestionIDs(ctx context.Context, f dailyquizmodel.QuestionFilter) ([]string, error) {
	conds := []string{
		"deleted_at IS NULL",
		"status = 'active'",
		`used_for @> '["daily_quiz"]'::jsonb`,
		"subject_field_id = ?",
		"grade_id = ?",
		"difficulty = ?",
	}
	args := []any{f.SubjectFieldID, f.GradeID, f.Difficulty}
	optional := []struct {
		column string
		value  *string
	}{
		{"chapter_id", f.ChapterID},
		{"subchapter_id", f.SubchapterID},
		{"topic_id", f.TopicID},
	}
	for _, o := range optional {
		if o.value != nil {
			conds = append(conds, o.column+" = ?")
			args = append(args, *o.value)
		}
	}
	args = append(args, f.Limit)

	ids := []string{}
	err := r.db.WithContext(ctx).
		Raw(`SELECT id FROM questions WHERE `+strings.Join(conds, " AND ")+` ORDER BY random() LIMIT ?`, args...).
		Scan(&ids).Error
	return ids, err
}

// Create menyimpan Daily Quiz + soal-soalnya dalam 1 transaksi. [Native SQL, 2 query]
// Setelah selesai, d.ID & d.CreatedAt sudah terisi.
func (r *DailyQuizRepository) Create(ctx context.Context, d *dailyquizmodel.DailyQuiz) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var saved struct {
		ID        string
		CreatedAt time.Time
	}
	err := tx.Raw(`
		INSERT INTO daily_quizzes (
			student_id, education_level_id, grade_id, subject_id, subject_field_id,
			chapter_id, subchapter_id, topic_id, difficulty, total_questions)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		d.StudentID, d.EducationLevelID, d.GradeID, d.SubjectID, d.SubjectFieldID,
		d.ChapterID, d.SubchapterID, d.TopicID, d.Difficulty, d.TotalQuestions).
		Scan(&saved).Error
	if err != nil {
		return err
	}

	rows := make([]string, 0, len(d.QuestionIDs))
	args := make([]any, 0, len(d.QuestionIDs)*3)
	for i, questionID := range d.QuestionIDs {
		rows = append(rows, "(?, ?, ?)")
		args = append(args, saved.ID, questionID, i+1)
	}
	err = tx.Exec(`INSERT INTO daily_quiz_questions (daily_quiz_id, question_id, sort_order) VALUES `+
		strings.Join(rows, ", "), args...).Error
	if err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	d.ID = saved.ID
	d.CreatedAt = saved.CreatedAt
	return nil
}

// FindStudentQuestions mengambil isi soal untuk siswa (urut sesuai ids), TANPA kunci jawaban.
// [Native SQL, 2 query]
//
//  1. Soal + link gambar soal (gambar Image Library di-JOIN ke media_assets, atau link luar).
//  2. Link gambar opsi jawaban dari Image Library (semua opsi semua soal sekaligus).
//
// Kunci jawaban dibuang di sini: dari kolom answer hanya diambil options (key, content, gambar)
// dan input_type; correct_key/correct_keys/accepted_answers tidak pernah keluar dari repository.
func (r *DailyQuizRepository) FindStudentQuestions(ctx context.Context, ids []string) ([]dailyquizmodel.StudentQuestion, error) {
	var rows []struct {
		ID, Type, QuestionText string
		Formula, ImageURL      *string
		TableData              *string
		DisplayOrder, Answer   string
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT q.id, q.type, q.question_text, q.formula,
			       COALESCE(m.url, q.image_url) AS image_url,
			       q.table_data::text AS table_data, q.display_order::text AS display_order,
			       q.answer::text AS answer
			FROM questions q
			LEFT JOIN media_assets m ON m.id = q.image_id
			WHERE q.id IN ?`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// Bagian answer yang boleh dilihat siswa (sama untuk semua tipe soal).
	type publicAnswer struct {
		Options   []questionmodel.ChoiceOption `json:"options"`
		InputType *string                      `json:"input_type"`
	}

	byID := make(map[string]dailyquizmodel.StudentQuestion, len(rows))
	optionImageIDs := []string{}
	answers := make(map[string]publicAnswer, len(rows))
	for _, row := range rows {
		q := dailyquizmodel.StudentQuestion{
			ID: row.ID, Type: row.Type, QuestionText: row.QuestionText, Formula: row.Formula, ImageURL: row.ImageURL,
		}
		if row.TableData != nil {
			q.Table = &questionmodel.Table{}
			if err := json.Unmarshal([]byte(*row.TableData), q.Table); err != nil {
				return nil, fmt.Errorf("soal %s: table_data rusak: %w", row.ID, err)
			}
		}
		if err := json.Unmarshal([]byte(row.DisplayOrder), &q.DisplayOrder); err != nil {
			return nil, fmt.Errorf("soal %s: display_order rusak: %w", row.ID, err)
		}
		var a publicAnswer
		if err := json.Unmarshal([]byte(row.Answer), &a); err != nil {
			return nil, fmt.Errorf("soal %s: answer rusak: %w", row.ID, err)
		}
		for _, o := range a.Options {
			if o.ImageID != nil {
				optionImageIDs = append(optionImageIDs, *o.ImageID)
			}
		}
		answers[row.ID] = a
		byID[row.ID] = q
	}

	imageURLs, err := r.findImageURLs(ctx, optionImageIDs)
	if err != nil {
		return nil, err
	}

	result := make([]dailyquizmodel.StudentQuestion, 0, len(ids))
	for _, id := range ids {
		q, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("soal %s tidak ditemukan", id)
		}
		a := answers[id]
		q.InputType = a.InputType
		for _, o := range a.Options {
			option := dailyquizmodel.Option{Key: o.Key, Content: o.Content, ImageURL: o.ImageURL}
			if o.ImageID != nil {
				if url, ok := imageURLs[*o.ImageID]; ok {
					option.ImageURL = &url
				}
			}
			q.Options = append(q.Options, option)
		}
		result = append(result, q)
	}
	return result, nil
}

// findImageURLs mengambil link gambar Image Library (map id → url). [Native SQL, primary key]
func (r *DailyQuizRepository) findImageURLs(ctx context.Context, ids []string) (map[string]string, error) {
	result := map[string]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct{ ID, URL string }
	if err := r.db.WithContext(ctx).Raw(`SELECT id, url FROM media_assets WHERE id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.URL
	}
	return result, nil
}
