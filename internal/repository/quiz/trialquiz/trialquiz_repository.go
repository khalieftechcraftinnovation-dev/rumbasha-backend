// Package trialquiz (repository) berisi query database fitur Quiz Trial publik.
//
// Di package lain, import dengan alias trialquizrepo:
//
//	trialquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/trialquiz"
package trialquiz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
	trialquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/trialquiz"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/question"
)

// TrialQuizRepository mengakses tabel trial_quiz_questions dan trial_quiz_attempts.
type TrialQuizRepository struct {
	db *gorm.DB
}

// NewTrialQuizRepository membuat TrialQuizRepository. Dipanggil sekali di main.go.
func NewTrialQuizRepository(db *gorm.DB) *TrialQuizRepository {
	return &TrialQuizRepository{db: db}
}

// ----------------------------------------------------------------------------
// Soal (admin)
// ----------------------------------------------------------------------------

// IsActiveSubject mengecek subject_id adalah mapel yang ada & aktif di master data. [Native SQL]
func (r *TrialQuizRepository) IsActiveSubject(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM master_data_items
			WHERE id = ? AND type = 'subject' AND status = 'active' AND deleted_at IS NULL)`, id).Scan(&ok).Error
	return ok, err
}

// lockGroup mengunci 1 golongan sampai transaksi selesai (advisory lock), supaya 2 admin yang
// menambah soal bersamaan tidak membuat golongan berisi lebih dari 10 soal.
func lockGroup(tx *gorm.DB, group string) error {
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "trial_quiz:"+group).Error
}

// countInGroup = jumlah soal (belum dihapus) di 1 golongan, tanpa menghitung excludeID.
func countInGroup(tx *gorm.DB, group, excludeID string) (int, error) {
	var total int
	err := tx.Raw(`
		SELECT COUNT(*)::int FROM trial_quiz_questions
		WHERE grade_group = ? AND deleted_at IS NULL AND id::text <> ?`, group, excludeID).Scan(&total).Error
	return total, err
}

// contentArgs mengubah kolom JSON soal menjadi teks untuk query.
func contentArgs(q trialquizmodel.Question) (table *string, displayOrder string, err error) {
	if q.Table != nil {
		b, err := json.Marshal(q.Table)
		if err != nil {
			return nil, "", err
		}
		s := string(b)
		table = &s
	}
	b, err := json.Marshal(q.DisplayOrder)
	if err != nil {
		return nil, "", err
	}
	return table, string(b), nil
}

// Create menyimpan 1 soal baru dan mengembalikan ID-nya. Golongan penuh (10 soal) →
// ErrGroupFull. [Native SQL, 1 transaksi]
func (r *TrialQuizRepository) Create(ctx context.Context, q trialquizmodel.Question) (string, error) {
	table, displayOrder, err := contentArgs(q)
	if err != nil {
		return "", err
	}

	var id string
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, q.GradeGroup); err != nil {
			return err
		}
		total, err := countInGroup(tx, q.GradeGroup, "")
		if err != nil {
			return err
		}
		if total >= trialquizmodel.MaxQuestionsPerGroup {
			return trialquizmodel.ErrGroupFull
		}

		return tx.Raw(`
			INSERT INTO trial_quiz_questions (
				grade_group, subject_id, sort_order, type, question_text, formula, image_id, image_url,
				table_data, explanation, display_order, difficulty, answer, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?::jsonb, ?)
			RETURNING id`,
			q.GradeGroup, q.SubjectID, q.SortOrder, q.Type, q.QuestionText, q.Formula, q.ImageID, q.ImageURL,
			table, q.Explanation, displayOrder, q.Difficulty, string(q.Answer), q.CreatedBy).
			Scan(&id).Error
	})
	return id, err
}

// Update mengganti seluruh isi 1 soal. Soal tidak ada → ErrQuestionNotFound. Pindah ke golongan
// yang sudah penuh → ErrGroupFull. [Native SQL, 1 transaksi]
func (r *TrialQuizRepository) Update(ctx context.Context, q trialquizmodel.Question) error {
	table, displayOrder, err := contentArgs(q)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, q.GradeGroup); err != nil {
			return err
		}
		total, err := countInGroup(tx, q.GradeGroup, q.ID)
		if err != nil {
			return err
		}
		if total >= trialquizmodel.MaxQuestionsPerGroup {
			return trialquizmodel.ErrGroupFull
		}

		res := tx.Exec(`
			UPDATE trial_quiz_questions SET
				grade_group = ?, subject_id = ?, sort_order = ?, type = ?, question_text = ?, formula = ?,
				image_id = ?, image_url = ?, table_data = ?::jsonb, explanation = ?, display_order = ?::jsonb,
				difficulty = ?, answer = ?::jsonb, updated_by = ?, updated_at = NOW()
			WHERE id = ? AND deleted_at IS NULL`,
			q.GradeGroup, q.SubjectID, q.SortOrder, q.Type, q.QuestionText, q.Formula,
			q.ImageID, q.ImageURL, table, q.Explanation, displayOrder,
			q.Difficulty, string(q.Answer), q.UpdatedBy, q.ID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return trialquizmodel.ErrQuestionNotFound
		}
		return nil
	})
}

// SoftDelete menghapus 1 soal (soft delete). Tidak ada / sudah dihapus → ErrQuestionNotFound. [Native SQL]
func (r *TrialQuizRepository) SoftDelete(ctx context.Context, id, deletedBy string) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE trial_quiz_questions SET deleted_at = NOW(), deleted_by = ?
		WHERE id = ? AND deleted_at IS NULL`, deletedBy, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return trialquizmodel.ErrQuestionNotFound
	}
	return nil
}

// questionSelect = kolom soal lengkap (admin) + nama mapel, pembuat, dan pengubah terakhir.
const questionSelect = `
	SELECT q.id, q.code, q.grade_group, q.subject_id, s.name AS subject_name, q.sort_order,
	       q.type, q.question_text, q.formula, q.image_id::text AS image_id, q.image_url,
	       q.table_data::text AS table_data, q.explanation, q.display_order::text AS display_order,
	       q.difficulty, q.answer::text AS answer,
	       q.created_by, cu.real_name AS created_by_name,
	       q.updated_by::text AS updated_by, uu.real_name AS updated_by_name,
	       q.created_at, q.updated_at
	FROM trial_quiz_questions q
	JOIN master_data_items s ON s.id = q.subject_id
	LEFT JOIN users cu ON cu.id = q.created_by
	LEFT JOIN users uu ON uu.id = q.updated_by`

// questionRow = 1 baris hasil questionSelect (kolom JSON masih teks).
type questionRow struct {
	ID, Code, GradeGroup, SubjectID, SubjectName string
	SortOrder                                    int
	Type, QuestionText                           string
	Formula, ImageID, ImageURL, TableData        *string
	Explanation, DisplayOrder, Difficulty        string
	Answer                                       string
	CreatedBy                                    string
	CreatedByName, UpdatedBy, UpdatedByName      *string
	CreatedAt, UpdatedAt                         time.Time
}

func (row questionRow) toModel() (trialquizmodel.Question, error) {
	q := trialquizmodel.Question{
		ID: row.ID, Code: row.Code, GradeGroup: row.GradeGroup, SubjectID: row.SubjectID,
		SubjectName: row.SubjectName, SortOrder: row.SortOrder,
		Type: row.Type, QuestionText: row.QuestionText, Formula: row.Formula,
		ImageID: row.ImageID, ImageURL: row.ImageURL, Explanation: row.Explanation, Difficulty: row.Difficulty,
		CreatedBy: row.CreatedBy, CreatedByName: row.CreatedByName,
		UpdatedBy: row.UpdatedBy, UpdatedByName: row.UpdatedByName,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.TableData != nil {
		q.Table = &questionmodel.Table{}
		if err := json.Unmarshal([]byte(*row.TableData), q.Table); err != nil {
			return q, fmt.Errorf("soal %s: table_data rusak: %w", row.Code, err)
		}
	}
	if err := json.Unmarshal([]byte(row.DisplayOrder), &q.DisplayOrder); err != nil {
		return q, fmt.Errorf("soal %s: display_order rusak: %w", row.Code, err)
	}
	answer, err := questionmodel.CanonicalAnswer(row.Type, []byte(row.Answer))
	if err != nil {
		return q, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
	}
	q.Answer = answer
	return q, nil
}

// FindByID mengambil 1 soal lengkap (admin). Tidak ada / sudah dihapus → ErrQuestionNotFound. [Native SQL]
func (r *TrialQuizRepository) FindByID(ctx context.Context, id string) (trialquizmodel.Question, error) {
	var rows []questionRow
	if err := r.db.WithContext(ctx).Raw(questionSelect+` WHERE q.id = ? AND q.deleted_at IS NULL`, id).
		Scan(&rows).Error; err != nil {
		return trialquizmodel.Question{}, err
	}
	if len(rows) == 0 {
		return trialquizmodel.Question{}, trialquizmodel.ErrQuestionNotFound
	}
	return rows[0].toModel()
}

// ListByGroup mengambil semua soal (lengkap) 1 golongan, urut nomor soal. "" = semua golongan.
// Isinya kecil (maks 10 per golongan), jadi tanpa pagination. [Native SQL]
func (r *TrialQuizRepository) ListByGroup(ctx context.Context, group string) ([]trialquizmodel.Question, error) {
	var rows []questionRow
	err := r.db.WithContext(ctx).Raw(questionSelect+`
		WHERE q.deleted_at IS NULL AND (? = '' OR q.grade_group = ?)
		ORDER BY q.grade_group, q.sort_order, q.created_at`, group, group).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]trialquizmodel.Question, 0, len(rows))
	for _, row := range rows {
		q, err := row.toModel()
		if err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	return result, nil
}

// CountByGroup = jumlah soal (belum dihapus) per golongan. [Native SQL]
func (r *TrialQuizRepository) CountByGroup(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		GradeGroup string
		Total      int
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT grade_group, COUNT(*)::int AS total FROM trial_quiz_questions
		WHERE deleted_at IS NULL GROUP BY grade_group`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := map[string]int{}
	for _, row := range rows {
		result[row.GradeGroup] = row.Total
	}
	return result, nil
}

// ----------------------------------------------------------------------------
// Publik
// ----------------------------------------------------------------------------

// ListGrades mengambil semua kelas aktif beserta jenjangnya, urut jenjang lalu kelas. [Native SQL]
func (r *TrialQuizRepository) ListGrades(ctx context.Context) ([]trialquizmodel.GradeOption, error) {
	var grades []trialquizmodel.GradeOption
	err := r.db.WithContext(ctx).Raw(`
		SELECT l.id AS level_id, l.name AS level_name, l.code AS level_code,
		       g.id AS grade_id, g.name AS grade_name, g.code AS grade_code
		FROM master_data_items g
		JOIN master_data_items l ON l.id = g.parent_id
		WHERE g.type = 'grade' AND g.status = 'active' AND g.deleted_at IS NULL
		  AND l.status = 'active' AND l.deleted_at IS NULL
		ORDER BY l.sort_order, g.sort_order, g.name`).Scan(&grades).Error
	return grades, err
}

// FindPublicQuestions mengambil soal 1 golongan untuk halaman trial publik (urut nomor soal):
// isi soal + kunci + pembahasan, gambar Image Library sudah berupa link. [Native SQL, 2 query]
func (r *TrialQuizRepository) FindPublicQuestions(ctx context.Context, group string) ([]trialquizmodel.PublicQuestion, error) {
	var rows []struct {
		ID, Code, Type, SubjectID, SubjectName, QuestionText string
		Formula, ImageURL, TableData                         *string
		DisplayOrder, Difficulty, Explanation, Answer        string
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT q.id, q.code, q.type, q.subject_id, s.name AS subject_name, q.question_text, q.formula,
		       COALESCE(m.url, q.image_url) AS image_url, q.table_data::text AS table_data,
		       q.display_order::text AS display_order, q.difficulty, q.explanation, q.answer::text AS answer
		FROM trial_quiz_questions q
		JOIN master_data_items s ON s.id = q.subject_id
		LEFT JOIN media_assets m ON m.id = q.image_id
		WHERE q.grade_group = ? AND q.deleted_at IS NULL
		ORDER BY q.sort_order, q.created_at`, group).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// Link gambar opsi jawaban (Image Library) semua soal sekaligus.
	imageIDs := []string{}
	for _, row := range rows {
		var a struct {
			Options []questionmodel.ChoiceOption `json:"options"`
		}
		if err := json.Unmarshal([]byte(row.Answer), &a); err != nil {
			return nil, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
		}
		for _, o := range a.Options {
			if o.ImageID != nil {
				imageIDs = append(imageIDs, *o.ImageID)
			}
		}
	}
	imageURLs, err := r.findImageURLs(ctx, imageIDs)
	if err != nil {
		return nil, err
	}

	result := make([]trialquizmodel.PublicQuestion, 0, len(rows))
	for _, row := range rows {
		q := trialquizmodel.PublicQuestion{
			ID: row.ID, Code: row.Code, Type: row.Type, SubjectID: row.SubjectID, SubjectName: row.SubjectName,
			QuestionText: row.QuestionText, Formula: row.Formula, ImageURL: row.ImageURL,
			Difficulty: row.Difficulty, Explanation: row.Explanation,
		}
		if row.TableData != nil {
			q.Table = &questionmodel.Table{}
			if err := json.Unmarshal([]byte(*row.TableData), q.Table); err != nil {
				return nil, fmt.Errorf("soal %s: table_data rusak: %w", row.Code, err)
			}
		}
		if err := json.Unmarshal([]byte(row.DisplayOrder), &q.DisplayOrder); err != nil {
			return nil, fmt.Errorf("soal %s: display_order rusak: %w", row.Code, err)
		}
		if q.Answer, err = questionrepo.AnswerWithImageURLs(row.Type, []byte(row.Answer), imageURLs); err != nil {
			return nil, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
		}
		result = append(result, q)
	}
	return result, nil
}

// findImageURLs mengambil link gambar Image Library (map id → url). [Native SQL, primary key]
func (r *TrialQuizRepository) findImageURLs(ctx context.Context, ids []string) (map[string]string, error) {
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

// InsertAttempt menyimpan 1 hasil trial, mengembalikan ID & waktu submit. [Native SQL, 1 query]
func (r *TrialQuizRepository) InsertAttempt(ctx context.Context, a trialquizmodel.Attempt) (string, time.Time, error) {
	answers, err := json.Marshal(a.Result.Answers)
	if err != nil {
		return "", time.Time{}, err
	}
	var rows []struct {
		ID          string
		SubmittedAt time.Time
	}
	err = r.db.WithContext(ctx).Raw(`
		INSERT INTO trial_quiz_attempts (
			name, whatsapp, education_level_id, grade_id, grade_group,
			score, correct_count, total_questions, duration_seconds, answers)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb)
		RETURNING id, submitted_at`,
		a.Name, a.WhatsApp, a.EducationLevelID, a.GradeID, a.GradeGroup,
		a.Result.Score, a.Result.CorrectCount, len(a.Result.Answers), a.DurationSeconds, string(answers)).
		Scan(&rows).Error
	if err != nil {
		return "", time.Time{}, err
	}
	if len(rows) != 1 {
		return "", time.Time{}, fmt.Errorf("gagal menyimpan hasil quiz trial")
	}
	return rows[0].ID, rows[0].SubmittedAt, nil
}

// ----------------------------------------------------------------------------
// Report (admin)
// ----------------------------------------------------------------------------

// attemptWhere = filter report (periode & golongan), dipakai list & ringkasan.
// search hanya dipakai list (nama / no WhatsApp, tidak peka huruf besar/kecil).
func attemptWhere(f trialquizmodel.AttemptFilter, withSearch bool) (string, []any) {
	where := "WHERE TRUE"
	args := []any{}
	if f.From != nil {
		where += " AND t.submitted_at >= ?"
		args = append(args, *f.From)
	}
	if f.To != nil {
		where += " AND t.submitted_at < ?"
		args = append(args, *f.To)
	}
	if f.GradeGroup != "" {
		where += " AND t.grade_group = ?"
		args = append(args, f.GradeGroup)
	}
	if withSearch && f.Search != "" {
		where += " AND (t.name ILIKE ? OR t.whatsapp LIKE ?)"
		args = append(args, "%"+f.Search+"%", "%"+f.Search+"%")
	}
	return where, args
}

// ListAttempts mengambil 1 halaman peserta trial (terbaru dulu) + total yang cocok. [Native SQL, 2 query]
func (r *TrialQuizRepository) ListAttempts(ctx context.Context, f trialquizmodel.AttemptFilter) ([]trialquizmodel.AttemptListItem, int, error) {
	where, args := attemptWhere(f, true)

	var total int
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*)::int FROM trial_quiz_attempts t `+where, args...).
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	items := []trialquizmodel.AttemptListItem{}
	if total == 0 {
		return items, 0, nil
	}
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		SELECT t.id, t.name, t.whatsapp, t.education_level_id, l.name AS education_level_name,
		       t.grade_id, g.name AS grade_name, t.grade_group, t.score::float8 AS score,
		       t.correct_count, t.total_questions, t.duration_seconds, t.submitted_at
		FROM trial_quiz_attempts t
		JOIN master_data_items l ON l.id = t.education_level_id
		JOIN master_data_items g ON g.id = t.grade_id
		`+where+`
		ORDER BY t.submitted_at DESC, t.id
		LIMIT ? OFFSET ?`, pageArgs...).Scan(&items).Error
	return items, total, err
}

// Summary menghitung ringkasan report: jumlah peserta & rata-rata nilai (total dan per golongan),
// serta 10 soal yang paling sering dijawab salah. [Native SQL, 2 query]
func (r *TrialQuizRepository) Summary(ctx context.Context, f trialquizmodel.AttemptFilter) (trialquizmodel.Summary, error) {
	where, args := attemptWhere(f, false)
	summary := trialquizmodel.Summary{ByGroup: []trialquizmodel.GroupStat{}, MostWrong: []trialquizmodel.WrongQuestion{}}

	// ROLLUP: 1 baris per golongan + 1 baris total (grade_group NULL).
	var groups []struct {
		GradeGroup   *string
		Participants int
		AverageScore float64
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT t.grade_group, COUNT(*)::int AS participants, COALESCE(ROUND(AVG(t.score), 2), 0)::float8 AS average_score
		FROM trial_quiz_attempts t `+where+`
		GROUP BY ROLLUP (t.grade_group)`, args...).Scan(&groups).Error; err != nil {
		return summary, err
	}
	for _, g := range groups {
		if g.GradeGroup == nil {
			summary.Participants, summary.AverageScore = g.Participants, g.AverageScore
			continue
		}
		summary.ByGroup = append(summary.ByGroup, trialquizmodel.GroupStat{
			GradeGroup: *g.GradeGroup, Participants: g.Participants, AverageScore: g.AverageScore,
		})
	}

	// Soal paling sering salah: buka JSON answers setiap pengerjaan (1 baris per soal).
	// Soal yang sudah dihapus tetap ditampilkan (LEFT JOIN tanpa filter deleted_at).
	if err := r.db.WithContext(ctx).Raw(`
		WITH per_question AS (
			SELECT a.elem->>'question_id' AS question_id,
			       COUNT(*)::int AS answered,
			       COUNT(*) FILTER (WHERE NOT (a.elem->>'is_correct')::boolean)::int AS wrong
			FROM trial_quiz_attempts t
			CROSS JOIN LATERAL jsonb_array_elements(t.answers) AS a(elem)
			`+where+`
			GROUP BY 1
		)
		SELECT p.question_id, q.code, q.question_text, s.name AS subject_name, q.grade_group,
		       p.answered, p.wrong
		FROM per_question p
		JOIN trial_quiz_questions q ON q.id::text = p.question_id
		JOIN master_data_items s ON s.id = q.subject_id
		WHERE p.wrong > 0
		ORDER BY p.wrong DESC, p.answered DESC, q.code
		LIMIT 10`, args...).Scan(&summary.MostWrong).Error; err != nil {
		return summary, err
	}
	return summary, nil
}
