// Package quiz (repository) mengakses tabel quizzes & quiz_questions, serta membaca
// master_data_items & questions untuk validasi quiz.
//
// Di package lain, import dengan alias quizrepo:
//
//	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/accesscode"
	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz"
)

// QuizRepository mengakses tabel quizzes & quiz_questions.
type QuizRepository struct {
	db *gorm.DB
}

// NewQuizRepository membuat QuizRepository. Dipanggil sekali di main.go.
func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

// FindMasterDataByIDs mengambil item master data letak quiz (1 query), hasilnya map id → item.
// Item yang dihapus (soft delete) tidak ikut. [Native SQL, 1 tabel]
func (r *QuizRepository) FindMasterDataByIDs(ctx context.Context, ids []string) (map[string]quizmodel.MasterDataItem, error) {
	var items []quizmodel.MasterDataItem
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, type, code, parent_id, grade_id, status
			FROM master_data_items
			WHERE id IN ? AND deleted_at IS NULL`, ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]quizmodel.MasterDataItem, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

// FindQuestionsByIDs mengambil soal yang dipilih untuk dicek, 1 query. Soal yang sudah dihapus
// ikut diambil (Deleted = true) supaya pesan error bisa menyebut kodenya (berguna untuk soal
// dari bundle lama). Hasilnya map id → soal; ID yang tidak ada di map berarti soal tidak ada.
// [Native SQL, 1 tabel, pakai primary key]
func (r *QuizRepository) FindQuestionsByIDs(ctx context.Context, ids []string) (map[string]quizmodel.QuestionCandidate, error) {
	var rows []struct {
		ID, Code, Status, UsedFor string // used_for masih teks JSON
		Deleted                   bool
		SubjectID, GradeID        string
		SubjectFieldID, ChapterID string
		SubchapterID, TopicID     *string
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, code, status, used_for::text AS used_for, deleted_at IS NOT NULL AS deleted,
			       subject_id, grade_id, subject_field_id, chapter_id, subchapter_id, topic_id
			FROM questions
			WHERE id IN ?`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]quizmodel.QuestionCandidate, len(rows))
	for _, row := range rows {
		q := quizmodel.QuestionCandidate{
			ID: row.ID, Code: row.Code, Status: row.Status, Deleted: row.Deleted,
			SubjectID: row.SubjectID, GradeID: row.GradeID,
			SubjectFieldID: row.SubjectFieldID, ChapterID: row.ChapterID,
			SubchapterID: row.SubchapterID, TopicID: row.TopicID,
		}
		if err := json.Unmarshal([]byte(row.UsedFor), &q.UsedFor); err != nil {
			return nil, fmt.Errorf("soal %s: used_for rusak: %w", q.Code, err)
		}
		result[q.ID] = q
	}
	return result, nil
}

// FindBundleQuestionIDs mengambil ID soal setiap bundle (quiz yang belum dihapus), urut nomor soal.
// Hasilnya map quiz id → daftar ID soal; quiz yang tidak ada di map berarti tidak ditemukan.
// [Native SQL, 1 query, JOIN 2 tabel lewat primary key]
func (r *QuizRepository) FindBundleQuestionIDs(ctx context.Context, quizIDs []string) (map[string][]string, error) {
	var rows []struct {
		QuizID     string
		QuestionID string
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT z.id AS quiz_id, qq.question_id
			FROM quizzes z
			JOIN quiz_questions qq ON qq.quiz_id = z.id
			WHERE z.id IN ? AND z.deleted_at IS NULL
			ORDER BY z.id, qq.sort_order`, quizIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string, len(quizIDs))
	for _, row := range rows {
		result[row.QuizID] = append(result[row.QuizID], row.QuestionID)
	}
	return result, nil
}

// Create menyimpan quiz + soal-soalnya dalam 1 transaksi (semua tersimpan atau tidak sama sekali).
// [Native SQL, 2 query]
//
//	INSERT INTO quizzes (...) VALUES (...) RETURNING id, created_at, updated_at
//	INSERT INTO quiz_questions (quiz_id, question_id, sort_order) VALUES (?, ?, 1), (?, ?, 2), ...
//
// Kalau kode akses sudah dipakai quiz lain, mengembalikan gorm.ErrDuplicatedKey (index unik
// quizzes_access_code_unique = satu-satunya data unik yang diisi di sini); service lalu
// membuat kode baru dan mencoba lagi.
// Setelah selesai, q.ID, q.CreatedAt, q.UpdatedAt sudah terisi.
func (r *QuizRepository) Create(ctx context.Context, q *quizmodel.Quiz) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var saved struct {
		ID        string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	err := tx.Raw(`
		INSERT INTO quizzes (
			name, type,
			education_level_id, grade_id, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
			start_at, duration_minutes, coin_cost, total_questions, status,
			access_code, access_code_updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), ?)
		RETURNING id, created_at, updated_at`,
		q.Name, q.Type,
		q.EducationLevelID, q.GradeID, q.SubjectID, q.SubjectFieldID, q.ChapterID, q.SubchapterID, q.TopicID,
		q.StartAt, q.DurationMinutes, q.CoinCost, q.TotalQuestions, q.Status,
		q.AccessCode, q.CreatedBy).
		Scan(&saved).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return quizmodel.ErrAccessCodeTaken
	}
	if err != nil {
		return err
	}

	// Semua soal dalam 1 INSERT banyak baris. Nomor soal (sort_order) = posisi di request + 1.
	rows := make([]string, 0, len(q.QuestionIDs))
	args := make([]any, 0, len(q.QuestionIDs)*3)
	for i, questionID := range q.QuestionIDs {
		rows = append(rows, "(?, ?, ?)")
		args = append(args, saved.ID, questionID, i+1)
	}
	err = tx.Exec(`INSERT INTO quiz_questions (quiz_id, question_id, sort_order) VALUES `+
		strings.Join(rows, ", "), args...).Error
	if err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	q.ID = saved.ID
	q.CreatedAt = saved.CreatedAt
	q.UpdatedAt = saved.UpdatedAt
	return nil
}

// quizDetailQuery mengambil 1 quiz dengan nama letak & nama pembuatnya. [Native SQL]
// master_data_items di-JOIN 7x (1 per tingkat), users 2x (pembuat & pengubah terakhir).
// Semua JOIN lewat primary key. Tingkat opsional memakai LEFT JOIN.
const quizDetailQuery = `
	SELECT
		z.id, z.name, z.type,
		z.education_level_id, level.name   AS education_level_name,
		z.grade_id,           grade.name   AS grade_name,
		z.subject_id,         subject.name AS subject_name,
		z.subject_field_id,   field.name   AS subject_field_name,
		z.chapter_id,         chapter.name AS chapter_name,
		z.subchapter_id,      sub.name     AS subchapter_name,
		z.topic_id,           topic.name   AS topic_name,
		z.start_at, z.duration_minutes, z.coin_cost, z.total_questions, z.status,
		z.access_code, z.access_code_updated_at,
		z.created_by,         u.real_name  AS created_by_name,
		z.updated_by,         uu.real_name AS updated_by_name,
		z.created_at, z.updated_at
	FROM quizzes z
	JOIN      master_data_items AS level   ON level.id   = z.education_level_id
	JOIN      master_data_items AS grade   ON grade.id   = z.grade_id
	JOIN      master_data_items AS subject ON subject.id = z.subject_id
	LEFT JOIN master_data_items AS field   ON field.id   = z.subject_field_id
	LEFT JOIN master_data_items AS chapter ON chapter.id = z.chapter_id
	LEFT JOIN master_data_items AS sub     ON sub.id     = z.subchapter_id
	LEFT JOIN master_data_items AS topic   ON topic.id   = z.topic_id
	LEFT JOIN users             AS u       ON u.id       = z.created_by
	LEFT JOIN users             AS uu      ON uu.id      = z.updated_by
	WHERE z.id = ? AND z.deleted_at IS NULL`

// quizDetailRow menampung hasil quizDetailQuery apa adanya.
type quizDetailRow struct {
	ID, Name, Type string

	EducationLevelID, EducationLevelName string
	GradeID, GradeName                   string
	SubjectID, SubjectName               string
	SubjectFieldID, SubjectFieldName     *string
	ChapterID, ChapterName               *string
	SubchapterID, SubchapterName         *string
	TopicID, TopicName                   *string

	StartAt                                   time.Time
	DurationMinutes, CoinCost, TotalQuestions int
	Status                                    string
	AccessCode                                *string
	AccessCodeUpdatedAt                       *time.Time

	CreatedBy                string
	CreatedByName            *string
	UpdatedBy, UpdatedByName *string
	CreatedAt, UpdatedAt     time.Time
}

// FindDetailByID mengambil 1 quiz lengkap + daftar soalnya (2 query).
// Mengembalikan quizmodel.ErrQuizNotFound kalau tidak ada atau sudah dihapus.
func (r *QuizRepository) FindDetailByID(ctx context.Context, id string) (quizmodel.QuizDetail, error) {
	var rows []quizDetailRow
	if err := r.db.WithContext(ctx).Raw(quizDetailQuery, id).Scan(&rows).Error; err != nil {
		return quizmodel.QuizDetail{}, err
	}
	if len(rows) == 0 {
		return quizmodel.QuizDetail{}, quizmodel.ErrQuizNotFound
	}
	row := rows[0]

	// Soal quiz urut nomor soal (primary key quiz_questions diawali quiz_id → pakai index).
	questions := []quizmodel.QuizQuestion{}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT qq.sort_order, q.id, q.code, q.type, q.question_text, q.difficulty
			FROM quiz_questions qq
			JOIN questions q ON q.id = qq.question_id
			WHERE qq.quiz_id = ?
			ORDER BY qq.sort_order`, id).
		Scan(&questions).Error
	if err != nil {
		return quizmodel.QuizDetail{}, err
	}

	return quizmodel.QuizDetail{
		Quiz: quizmodel.Quiz{
			ID: row.ID, Name: row.Name, Type: row.Type,
			EducationLevelID: row.EducationLevelID, GradeID: row.GradeID, SubjectID: row.SubjectID,
			SubjectFieldID: row.SubjectFieldID, ChapterID: row.ChapterID,
			SubchapterID: row.SubchapterID, TopicID: row.TopicID,
			StartAt: row.StartAt, DurationMinutes: row.DurationMinutes, CoinCost: row.CoinCost,
			TotalQuestions: row.TotalQuestions, Status: row.Status,
			AccessCode: row.AccessCode, AccessCodeUpdatedAt: row.AccessCodeUpdatedAt,
			CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
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
		Questions:          questions,
	}, nil
}

// likeEscaper membuat %, _, dan \ yang diketik user dicari apa adanya (bukan wildcard LIKE).
var likeEscaper = strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)

// bundleListQuery mengambil 1 halaman bundle (quiz) dengan nama jenjang, kelas, mapel. [Native SQL]
// Sama seperti list soal: halaman dipotong dulu di tabel quizzes (CTE), baru JOIN nama.
// %s diganti kondisi WHERE dari ListBundles.
const bundleListQuery = `
	WITH page AS (
		SELECT z.id, z.created_at
		FROM quizzes z
		WHERE %s
		ORDER BY z.created_at DESC, z.id DESC
		LIMIT ? OFFSET ?
	)
	SELECT
		z.id, z.name, z.type,
		z.education_level_id, level.name   AS education_level_name,
		z.grade_id,           grade.name   AS grade_name,
		z.subject_id,         subject.name AS subject_name,
		z.total_questions, z.created_at
	FROM page p
	JOIN quizzes           AS z       ON z.id       = p.id
	JOIN master_data_items AS level   ON level.id   = z.education_level_id
	JOIN master_data_items AS grade   ON grade.id   = z.grade_id
	JOIN master_data_items AS subject ON subject.id = z.subject_id
	ORDER BY p.created_at DESC, p.id DESC`

// ListBundles mengambil 1 halaman bundle (quiz yang belum dihapus), terbaru dulu, beserta totalnya.
func (r *QuizRepository) ListBundles(ctx context.Context, f quizmodel.BundleFilter) ([]quizmodel.Bundle, int, error) {
	conds := []string{"z.deleted_at IS NULL"}
	args := []any{}
	equals := []struct{ column, value string }{
		{"z.type", f.Type},
		{"z.education_level_id", f.EducationLevelID},
		{"z.grade_id", f.GradeID},
		{"z.subject_id", f.SubjectID},
	}
	for _, e := range equals {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.Search != "" {
		conds = append(conds, "z.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	where := strings.Join(conds, " AND ")

	bundles := []quizmodel.Bundle{}
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	if err := r.db.WithContext(ctx).Raw(fmt.Sprintf(bundleListQuery, where), pageArgs...).Scan(&bundles).Error; err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM quizzes z WHERE "+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	return bundles, total, nil
}

// RefreshLockKey adalah kunci advisory lock PostgreSQL untuk refresh kode akses. Kalau API
// berjalan di beberapa server, hanya 1 server yang mengerjakan refresh di jam yang sama.
const RefreshLockKey int64 = 720261125 // angka bebas, asal tidak dipakai fitur lain

// refreshChunkSize = jumlah quiz per 1 query UPDATE (2 parameter per quiz).
const refreshChunkSize = 1000

// RefreshAccessCodes mengganti kode akses semua quiz berkode yang BELUM SELESAI
// (start_at + durasi > now), dalam 1 transaksi. [Native SQL]
//
//  1. pg_try_advisory_xact_lock: kalau server lain sedang mengerjakan, langsung berhenti (skipped = true).
//  2. Ambil quiz yang belum selesai (index quizzes_access_code_start_at_idx), kunci barisnya (FOR UPDATE).
//  3. Buat kode baru yang tidak dipakai quiz mana pun (termasuk kode lama), lalu UPDATE sekaligus.
//
// Mengembalikan jumlah quiz yang kodenya diganti.
func (r *QuizRepository) RefreshAccessCodes(ctx context.Context, now time.Time) (refreshed int, skipped bool, err error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, false, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var locked bool
	if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(?)`, RefreshLockKey).Scan(&locked).Error; err != nil {
		return 0, false, err
	}
	if !locked {
		return 0, true, nil
	}

	// Durasi maksimal 600 menit, jadi quiz yang belum selesai pasti start_at > now - 600 menit
	// (kondisi ini yang memakai index); kondisi kedua menghitung waktu selesai sebenarnya.
	var ids []string
	err = tx.Raw(`
		SELECT id FROM quizzes
		WHERE deleted_at IS NULL AND access_code IS NOT NULL
		  AND start_at > ?::timestamptz - INTERVAL '600 minutes'
		  AND start_at + make_interval(mins => duration_minutes) > ?
		ORDER BY id
		FOR UPDATE`, now, now).
		Scan(&ids).Error
	if err != nil {
		return 0, false, err
	}
	if len(ids) == 0 {
		return 0, false, tx.Commit().Error
	}

	codes, err := newUnusedCodes(tx, len(ids))
	if err != nil {
		return 0, false, err
	}

	for start := 0; start < len(ids); start += refreshChunkSize {
		end := min(start+refreshChunkSize, len(ids))
		rows := make([]string, 0, end-start)
		args := []any{now}
		for i := start; i < end; i++ {
			rows = append(rows, "(?::uuid, ?)")
			args = append(args, ids[i], codes[i])
		}
		err := tx.Exec(`
			UPDATE quizzes AS z
			SET access_code = v.code, access_code_updated_at = ?
			FROM (VALUES `+strings.Join(rows, ", ")+`) AS v (id, code)
			WHERE z.id = v.id`, args...).Error
		if err != nil {
			return 0, false, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, false, err
	}
	return len(ids), false, nil
}

// newUnusedCodes membuat n kode akses berbeda yang belum dipakai quiz mana pun (yang belum dihapus).
// Kode yang ternyata sudah dipakai diganti lalu dicek lagi (sangat jarang: 36^6 ≈ 2,1 miliar kode).
func newUnusedCodes(tx *gorm.DB, n int) ([]string, error) {
	exclude := map[string]bool{}
	codes, err := accesscode.NewUnique(n, exclude)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 10; attempt++ {
		var used []string
		if err := tx.Raw(`SELECT access_code FROM quizzes WHERE access_code IN ? AND deleted_at IS NULL`, codes).
			Scan(&used).Error; err != nil {
			return nil, err
		}
		if len(used) == 0 {
			return codes, nil
		}

		// Ganti kode yang bentrok dengan kode baru yang tidak ada di daftar sekarang maupun yang bentrok.
		for _, c := range codes {
			exclude[c] = true
		}
		for _, c := range used {
			exclude[c] = true
		}
		usedSet := make(map[string]bool, len(used))
		for _, c := range used {
			usedSet[c] = true
		}
		for i, c := range codes {
			if usedSet[c] {
				replacement, err := accesscode.NewUnique(1, exclude)
				if err != nil {
					return nil, err
				}
				codes[i] = replacement[0]
				exclude[replacement[0]] = true
			}
		}
	}
	return nil, fmt.Errorf("gagal membuat %d kode akses unik setelah 10 percobaan", n)
}
