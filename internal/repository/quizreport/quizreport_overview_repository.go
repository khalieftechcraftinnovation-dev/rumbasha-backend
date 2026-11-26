package quizreport

import (
	"context"
	"errors"
	"strings"
	"time"

	quizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quizreport"
)

// sessionSource menyusun CTE "s" = pengerjaan yang sudah submit di rentang [from, to) untuk 1 tab
// report, dengan kolom yang sama untuk Daily Quiz maupun Uji Kom / Tryout:
//
//	score, submitted_at, level_id, quiz_id, student_id, timed_out, answers
//
// Uji Kom / Tryout: quiz_attempts (index quiz_attempts_period_idx) + quiz publish yang belum dihapus.
// timed_out = dikumpulkan saat waktu sudah habis (≥ batas waktu = mulai + durasi, atau akhir event
// kalau lebih dulu), biasanya submit otomatis FE ketika timer habis.
// Daily Quiz: daily_quizzes (index daily_quizzes_report_idx), tanpa timer (timed_out NULL).
// Akun yang sudah dihapus tidak dihitung (sama dengan report lain).
func sessionSource(f quizreportmodel.OverviewFilter, from, to time.Time) (string, []any) {
	var sb strings.Builder
	args := []any{}
	var alias string // tabel yang menyimpan letak (jenjang, kelas, mapel)
	if f.Type == quizreportmodel.TypeDailyQuiz {
		alias = "d"
		sb.WriteString(`
		WITH s AS (
			SELECT d.score::float8 AS score, d.submitted_at, d.education_level_id AS level_id,
			       NULL::uuid AS quiz_id, d.student_id, NULL::boolean AS timed_out, d.answers
			FROM daily_quizzes d
			JOIN users u ON u.id = d.student_id AND u.deleted_at IS NULL
			WHERE d.submitted_at IS NOT NULL AND d.submitted_at >= ? AND d.submitted_at < ?`)
		args = append(args, from, to)
	} else {
		alias = "z"
		sb.WriteString(`
		WITH s AS (
			SELECT a.score::float8 AS score, a.submitted_at, z.education_level_id AS level_id,
			       a.quiz_id, a.student_id,
			       a.submitted_at >= LEAST(a.started_at + make_interval(mins => z.duration_minutes), z.end_at) AS timed_out,
			       a.answers
			FROM quiz_attempts a
			JOIN quizzes z ON z.id = a.quiz_id AND z.deleted_at IS NULL AND z.status = 'published' AND z.type = ?
			JOIN users u   ON u.id = a.student_id AND u.deleted_at IS NULL
			WHERE a.submitted_at IS NOT NULL AND a.submitted_at >= ? AND a.submitted_at < ?`)
		args = append(args, f.Type, from, to)
	}
	for _, e := range []struct{ column, value string }{
		{alias + ".education_level_id::text", f.EducationLevelID},
		{alias + ".grade_id::text", f.GradeID},
		{alias + ".subject_id::text", f.SubjectID},
	} {
		if e.value != "" {
			sb.WriteString(" AND " + e.column + " = ?")
			args = append(args, e.value)
		}
	}
	sb.WriteString("\n\t\t)")
	return sb.String(), args
}

// OverviewTotals menghitung angka kartu + sebaran nilai dari semua pengerjaan di periode, 1 query.
// [Native SQL]
func (r *QuizReportRepository) OverviewTotals(ctx context.Context, f quizreportmodel.OverviewFilter) (quizreportmodel.Totals, error) {
	source, args := sessionSource(f, f.From, f.To)
	var row struct {
		Sessions, DistinctQuizzes, Students, TimedOut int
		Average                                       *float64
		B1, B2, B3, B4, B5                            int
	}
	err := r.db.WithContext(ctx).Raw(source+`
		SELECT COUNT(*)::int                                AS sessions,
		       COUNT(DISTINCT quiz_id)::int                 AS distinct_quizzes,
		       COUNT(DISTINCT student_id)::int              AS students,
		       ROUND(AVG(score)::numeric, 2)::float8        AS average,
		       COUNT(*) FILTER (WHERE timed_out)::int       AS timed_out,
		       COUNT(*) FILTER (WHERE score <= 20)::int                 AS b1,
		       COUNT(*) FILTER (WHERE score > 20 AND score <= 40)::int  AS b2,
		       COUNT(*) FILTER (WHERE score > 40 AND score <= 60)::int  AS b3,
		       COUNT(*) FILTER (WHERE score > 60 AND score <= 80)::int  AS b4,
		       COUNT(*) FILTER (WHERE score > 80)::int                  AS b5
		FROM s`, args...).
		Scan(&row).Error
	return quizreportmodel.Totals{
		Sessions: row.Sessions, DistinctQuizzes: row.DistinctQuizzes, Students: row.Students,
		Average: row.Average, TimedOut: row.TimedOut, Buckets: [5]int{row.B1, row.B2, row.B3, row.B4, row.B5},
	}, err
}

// OverviewByLevel menghitung rata-rata nilai per jenjang (urut jenjang), 1 query. [Native SQL]
func (r *QuizReportRepository) OverviewByLevel(ctx context.Context, f quizreportmodel.OverviewFilter) ([]quizreportmodel.LevelAverage, error) {
	source, args := sessionSource(f, f.From, f.To)
	var rows []struct {
		LevelID, LevelName string
		Sessions           int
		Average            float64
	}
	err := r.db.WithContext(ctx).Raw(source+`
		SELECT s.level_id, l.name AS level_name, COUNT(*)::int AS sessions,
		       ROUND(AVG(s.score)::numeric, 2)::float8 AS average
		FROM s
		JOIN master_data_items l ON l.id = s.level_id
		GROUP BY s.level_id, l.name, l.sort_order
		ORDER BY l.sort_order, l.name`, args...).
		Scan(&rows).Error
	items := make([]quizreportmodel.LevelAverage, 0, len(rows))
	for _, row := range rows {
		items = append(items, quizreportmodel.LevelAverage{
			Level: quizreportmodel.Ref{ID: row.LevelID, Name: row.LevelName}, Sessions: row.Sessions, Average: row.Average,
		})
	}
	return items, err
}

// OverviewWrongQuestions menghitung per soal (dari JSONB answers hasil submit) berapa kali dikerjakan
// & salah (termasuk tidak dijawab) di periode, lalu mengambil soal yang salahnya LEBIH DARI ambang
// persen, urut persentase salah tertinggi. 1 query; hasil selalu ≥ 1 baris supaya jumlah soal
// berbeda tetap terbaca walau tidak ada soal yang lewat ambang. [Native SQL]
func (r *QuizReportRepository) OverviewWrongQuestions(ctx context.Context, f quizreportmodel.OverviewFilter) (quizreportmodel.WrongSummary, error) {
	source, args := sessionSource(f, f.From, f.To)
	args = append(args, f.WrongThresholdPercent)
	var rows []struct {
		TotalQuestions       int
		QuestionID           *string
		Code, QuestionText   *string
		ChapterID, Chapter   *string
		Answered, WrongCount int
	}
	err := r.db.WithContext(ctx).Raw(source+`,
		per_q AS (
			SELECT (e ->> 'question_id')::uuid AS question_id,
			       COUNT(*)::int                                                AS answered,
			       COUNT(*) FILTER (WHERE NOT (e ->> 'is_correct')::boolean)::int AS wrong
			FROM s, jsonb_array_elements(s.answers) AS e
			GROUP BY 1
		)
		SELECT t.total_questions, w.question_id, w.code, w.question_text, w.chapter_id, w.chapter,
		       COALESCE(w.answered, 0) AS answered, COALESCE(w.wrong, 0) AS wrong_count
		FROM (SELECT COUNT(*)::int AS total_questions FROM per_q) t
		LEFT JOIN LATERAL (
			SELECT q.id::text AS question_id, q.code, q.question_text,
			       q.chapter_id::text AS chapter_id, c.name AS chapter, p.answered, p.wrong
			FROM per_q p
			JOIN questions q              ON q.id = p.question_id
			LEFT JOIN master_data_items c ON c.id = q.chapter_id
			WHERE p.wrong * 100 > ? * p.answered
		) w ON TRUE
		ORDER BY w.wrong::float8 / NULLIF(w.answered, 0) DESC NULLS LAST, w.wrong DESC, w.code`, args...).
		Scan(&rows).Error
	if err != nil {
		return quizreportmodel.WrongSummary{}, err
	}
	summary := quizreportmodel.WrongSummary{Questions: []quizreportmodel.WrongQuestion{}}
	for _, row := range rows {
		summary.TotalQuestions = row.TotalQuestions
		if row.QuestionID == nil {
			continue // baris kosong dari LEFT JOIN: tidak ada soal yang lewat ambang
		}
		q := quizreportmodel.WrongQuestion{
			QuestionID: *row.QuestionID, Code: deref(row.Code), QuestionText: deref(row.QuestionText),
			Answered: row.Answered, Wrong: row.WrongCount,
		}
		if row.ChapterID != nil {
			q.Chapter = &quizreportmodel.Ref{ID: *row.ChapterID, Name: deref(row.Chapter)}
		}
		summary.Questions = append(summary.Questions, q)
	}
	return summary, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// OverviewTrend menghitung jumlah & total nilai pengerjaan per minggu (Senin-Minggu, WIB) per jenjang
// di rentang [from, to). Rata-rata dihitung di service supaya rata-rata semua jenjang tetap tepat
// (bukan rata-rata dari rata-rata). 1 query. [Native SQL]
func (r *QuizReportRepository) OverviewTrend(ctx context.Context, f quizreportmodel.OverviewFilter, from, to time.Time) ([]quizreportmodel.WeekLevel, error) {
	source, args := sessionSource(f, from, to)
	var rows []struct {
		WeekStart          time.Time
		LevelID, LevelName string
		LevelSort          int
		Sessions           int
		ScoreSum           float64
	}
	err := r.db.WithContext(ctx).Raw(source+`
		SELECT date_trunc('week', s.submitted_at AT TIME ZONE 'Asia/Jakarta')::date AS week_start,
		       s.level_id, l.name AS level_name, l.sort_order AS level_sort,
		       COUNT(*)::int AS sessions, SUM(s.score)::float8 AS score_sum
		FROM s
		JOIN master_data_items l ON l.id = s.level_id
		GROUP BY 1, s.level_id, l.name, l.sort_order
		ORDER BY 1, l.sort_order`, args...).
		Scan(&rows).Error
	items := make([]quizreportmodel.WeekLevel, 0, len(rows))
	for _, row := range rows {
		items = append(items, quizreportmodel.WeekLevel{
			WeekStart: row.WeekStart, Level: quizreportmodel.Ref{ID: row.LevelID, Name: row.LevelName},
			LevelSort: row.LevelSort, Sessions: row.Sessions, ScoreSum: row.ScoreSum,
		})
	}
	return items, err
}

// FindSettings mengambil pengaturan report quiz (1 baris). [Native SQL, primary key]
func (r *QuizReportRepository) FindSettings(ctx context.Context) (quizreportmodel.Settings, error) {
	var rows []struct {
		WrongThresholdPercent, DefaultRangeDays int
		UpdatedByID, UpdatedByName              *string
		UpdatedAt                               time.Time
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.wrong_threshold_percent, s.default_range_days,
		       s.updated_by AS updated_by_id, u.real_name AS updated_by_name, s.updated_at
		FROM quiz_report_settings s
		LEFT JOIN users u ON u.id = s.updated_by
		WHERE s.id = 1`).Scan(&rows).Error
	if err != nil {
		return quizreportmodel.Settings{}, err
	}
	if len(rows) == 0 {
		return quizreportmodel.Settings{}, errors.New("baris quiz_report_settings tidak ada (sudah menjalankan migrate-up?)")
	}
	row := rows[0]
	s := quizreportmodel.Settings{WrongThresholdPercent: row.WrongThresholdPercent, DefaultRangeDays: row.DefaultRangeDays, UpdatedAt: row.UpdatedAt}
	if row.UpdatedByID != nil {
		s.UpdatedBy = &quizreportmodel.Ref{ID: *row.UpdatedByID, Name: deref(row.UpdatedByName)}
	}
	return s, nil
}

// UpdateSettings mengganti pengaturan report quiz. [Native SQL, primary key]
func (r *QuizReportRepository) UpdateSettings(ctx context.Context, s quizreportmodel.Settings, updatedBy string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE quiz_report_settings SET wrong_threshold_percent = ?, default_range_days = ?, updated_by = ?, updated_at = NOW()
		WHERE id = 1`, s.WrongThresholdPercent, s.DefaultRangeDays, updatedBy).Error
}
