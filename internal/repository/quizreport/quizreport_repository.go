// Package quizreport (repository) menghitung report quiz Uji Kom & Tryout dari quizzes,
// quiz_attempts, dan quiz_questions.
//
// Peserta = siswa yang SUDAH submit (akun yang sudah dihapus tidak dihitung, sama dengan leaderboard).
//
// Di package lain, import dengan alias quizreportrepo:
//
//	quizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quizreport"
package quizreport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quizreport"
)

// QuizReportRepository membaca data report quiz.
type QuizReportRepository struct {
	db *gorm.DB
}

// NewQuizReportRepository membuat QuizReportRepository. Dipanggil sekali di main.go.
func NewQuizReportRepository(db *gorm.DB) *QuizReportRepository {
	return &QuizReportRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// summaryRow menampung 1 baris ringkasan (flat, untuk Scan).
type summaryRow struct {
	QuizID, Name, Type        string
	LevelID, LevelName        string
	GradeID, GradeName        string
	SubjectID, SubjectName    string
	StartAt, EndAt            time.Time
	TotalQuestions            int
	Participants              int
	AverageScore              *float64
	HighestScore, LowestScore *float64
}

func (r summaryRow) toSummary() quizreportmodel.Summary {
	return quizreportmodel.Summary{
		QuizID: r.QuizID, Name: r.Name, Type: r.Type,
		EducationLevel: quizreportmodel.Ref{ID: r.LevelID, Name: r.LevelName},
		Grade:          quizreportmodel.Ref{ID: r.GradeID, Name: r.GradeName},
		Subject:        quizreportmodel.Ref{ID: r.SubjectID, Name: r.SubjectName},
		StartAt:        r.StartAt, EndAt: r.EndAt, TotalQuestions: r.TotalQuestions,
		Participants: r.Participants, AverageScore: r.AverageScore,
		HighestScore: r.HighestScore, LowestScore: r.LowestScore,
	}
}

// summarySelect = kolom ringkasan quiz (alias z; %s = klausa FROM yang berisi z) + statistik nilai peserta (LATERAL, per quiz
// memakai index quiz_attempts_report_idx). Peserta dari akun yang sudah dihapus tidak dihitung.
const summarySelect = `
	SELECT z.id AS quiz_id, z.name, z.type,
	       z.education_level_id AS level_id, l.name AS level_name,
	       z.grade_id, g.name AS grade_name,
	       z.subject_id, sub.name AS subject_name,
	       z.start_at, z.end_at, z.total_questions,
	       COALESCE(st.participants, 0) AS participants,
	       st.average_score, st.highest_score, st.lowest_score
	%s
	JOIN master_data_items l   ON l.id   = z.education_level_id
	JOIN master_data_items g   ON g.id   = z.grade_id
	JOIN master_data_items sub ON sub.id = z.subject_id
	LEFT JOIN LATERAL (
		SELECT COUNT(*)::int                          AS participants,
		       ROUND(AVG(a.score)::numeric, 2)::float8 AS average_score,
		       MAX(a.score)::float8                   AS highest_score,
		       MIN(a.score)::float8                   AS lowest_score
		FROM quiz_attempts a
		JOIN users u ON u.id = a.student_id AND u.deleted_at IS NULL
		WHERE a.quiz_id = z.id AND a.submitted_at IS NOT NULL
	) st ON TRUE`

// reportQuizWhere = quiz yang masuk report: Uji Kom & Tryout yang sudah publish dan belum dihapus.
const reportQuizWhere = `z.deleted_at IS NULL AND z.status = 'published' AND z.type IN ('uji_kom', 'tryout')`

// List mengambil 1 halaman ringkasan report (event terbaru dulu) + total. [Native SQL]
//
// Halaman quiz diambil dulu (CTE, index quizzes_report_idx), baru statistik peserta dihitung untuk
// ≤ per_page quiz itu saja. COUNT dilewati di halaman terakhir yang tidak penuh.
func (r *QuizReportRepository) List(ctx context.Context, f quizreportmodel.ListFilter) ([]quizreportmodel.Summary, int, error) {
	conds := []string{reportQuizWhere}
	args := []any{}
	for _, e := range []struct{ column, value string }{
		{"z.type", f.Type},
		{"z.education_level_id::text", f.EducationLevelID},
		{"z.grade_id::text", f.GradeID},
		{"z.subject_id::text", f.SubjectID},
	} {
		if e.value != "" {
			conds = append(conds, e.column+" = ?")
			args = append(args, e.value)
		}
	}
	if f.Search != "" {
		conds = append(conds, "z.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	if f.From != nil {
		conds = append(conds, "z.start_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conds = append(conds, "z.start_at < ?")
		args = append(args, *f.To)
	}
	where := strings.Join(conds, " AND ")

	var rows []summaryRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		WITH page AS (
			SELECT z.id FROM quizzes z
			WHERE `+where+`
			ORDER BY z.start_at DESC, z.id DESC
			LIMIT ? OFFSET ?
		)`+fmt.Sprintf(summarySelect, "FROM page p JOIN quizzes z ON z.id = p.id")+`
		ORDER BY z.start_at DESC, z.id DESC`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]quizreportmodel.Summary, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toSummary())
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM quizzes z WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// FindSummary mengambil ringkasan 1 quiz report. Bukan quiz report → ErrQuizNotFound. [Native SQL]
func (r *QuizReportRepository) FindSummary(ctx context.Context, quizID string) (quizreportmodel.Summary, error) {
	var rows []summaryRow
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(summarySelect, "FROM quizzes z")+` WHERE z.id = ? AND `+reportQuizWhere, quizID).Scan(&rows).Error
	if err != nil {
		return quizreportmodel.Summary{}, err
	}
	if len(rows) == 0 {
		return quizreportmodel.Summary{}, quizreportmodel.ErrQuizNotFound
	}
	return rows[0].toSummary(), nil
}

// Participants mengambil 1 halaman peserta quiz urut peringkat + total (setelah pencarian).
// Peringkat dihitung dari SEMUA peserta (sebelum pencarian), aturan sama dengan leaderboard &
// hadiah: point tertinggi, lalu waktu tersingkat; sama persis → peringkat sama. [Native SQL]
func (r *QuizReportRepository) Participants(ctx context.Context, quizID string, f quizreportmodel.ParticipantFilter) ([]quizreportmodel.Participant, int, error) {
	const ranked = `
		WITH ranked AS (
			SELECT a.student_id, u.real_name AS name, s.student_code,
			       a.correct_count, a.score::float8 AS score, a.points, a.duration_seconds, a.submitted_at,
			       RANK() OVER (ORDER BY a.points DESC, a.duration_seconds ASC)::int AS rank
			FROM quiz_attempts a
			JOIN users u          ON u.id = a.student_id AND u.deleted_at IS NULL
			LEFT JOIN students s  ON s.user_id = a.student_id
			WHERE a.quiz_id = ? AND a.submitted_at IS NOT NULL
		)`
	where := "TRUE"
	args := []any{quizID}
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		where = "(name ILIKE ? OR student_code ILIKE ?)"
		args = append(args, like, like)
	}

	items := []quizreportmodel.Participant{}
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(ranked+`
		SELECT rank, student_id, name, student_code, correct_count, score, points, duration_seconds, submitted_at
		FROM ranked WHERE `+where+`
		ORDER BY rank, name, student_id
		LIMIT ? OFFSET ?`, pageArgs...).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(ranked+` SELECT COUNT(*) FROM ranked WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// QuestionStats menghitung per soal quiz: jumlah peserta yang menjawab benar, salah, dan tidak
// menjawab (dari kolom JSONB answers hasil submit). Semua soal quiz ikut (urut nomor soal), termasuk
// yang belum dijawab siapa pun. 1 query. [Native SQL]
func (r *QuizReportRepository) QuestionStats(ctx context.Context, quizID string) ([]quizreportmodel.QuestionStat, error) {
	stats := []quizreportmodel.QuestionStat{}
	err := r.db.WithContext(ctx).Raw(`
		WITH attempts AS (
			SELECT a.answers
			FROM quiz_attempts a
			JOIN users u ON u.id = a.student_id AND u.deleted_at IS NULL
			WHERE a.quiz_id = ? AND a.submitted_at IS NOT NULL
		),
		per_question AS (
			SELECT e ->> 'question_id' AS question_id,
			       COUNT(*) FILTER (WHERE (e ->> 'is_correct')::boolean)                                        AS correct,
			       COUNT(*) FILTER (WHERE NOT (e ->> 'is_correct')::boolean
			                          AND COALESCE(e -> 'answer', 'null'::jsonb) <> 'null'::jsonb)             AS wrong,
			       COUNT(*) FILTER (WHERE NOT (e ->> 'is_correct')::boolean
			                          AND COALESCE(e -> 'answer', 'null'::jsonb) = 'null'::jsonb)              AS unanswered
			FROM attempts, jsonb_array_elements(attempts.answers) AS e
			GROUP BY 1
		)
		SELECT qq.sort_order AS no, q.id AS question_id, q.code, q.type, q.question_text, q.difficulty,
		       COALESCE(pq.correct, 0)::int    AS correct,
		       COALESCE(pq.wrong, 0)::int      AS wrong,
		       COALESCE(pq.unanswered, 0)::int AS unanswered,
		       (SELECT COUNT(*) FROM attempts)::int AS participants
		FROM quiz_questions qq
		JOIN questions q          ON q.id = qq.question_id
		LEFT JOIN per_question pq ON pq.question_id = q.id::text
		WHERE qq.quiz_id = ?
		ORDER BY qq.sort_order`, quizID, quizID).
		Scan(&stats).Error
	return stats, err
}
