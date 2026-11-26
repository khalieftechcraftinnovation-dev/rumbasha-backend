package dailyquiz

import (
	"context"
	"strings"
	"time"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
)

// historyWhere menyusun kondisi riwayat Daily Quiz 1 siswa (yang sudah dikumpulkan).
func historyWhere(studentID string, f dailyquizmodel.HistoryFilter) (string, []any) {
	conds := []string{"d.student_id = ?", "d.submitted_at IS NOT NULL"}
	args := []any{studentID}
	if f.From != nil {
		conds = append(conds, "d.submitted_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conds = append(conds, "d.submitted_at < ?")
		args = append(args, *f.To)
	}
	if f.SubjectID != "" {
		conds = append(conds, "d.subject_id = ?")
		args = append(args, f.SubjectID)
	}
	if f.Difficulty != "" {
		conds = append(conds, "d.difficulty = ?")
		args = append(args, f.Difficulty)
	}
	return strings.Join(conds, " AND "), args
}

// History mengambil 1 halaman riwayat Daily Quiz siswa (terbaru dulu) + nama materi, + total.
// [Native SQL, index daily_quizzes_student_report_idx; JOIN master data lewat primary key]
func (r *DailyQuizRepository) History(ctx context.Context, studentID string, f dailyquizmodel.HistoryFilter) ([]dailyquizmodel.HistoryItem, int, error) {
	where, args := historyWhere(studentID, f)
	var rows []struct {
		ID                                         string
		SubjectID, SubjectName, FieldID, FieldName string
		ChapterID, ChapterName                     *string
		SubchapterID, SubchapterName               *string
		TopicID, TopicName                         *string
		GradeID, GradeName                         string
		Difficulty                                 string
		TotalQuestions, CorrectCount               int
		Score                                      float64
		Points, DurationSeconds, CoinCharged       int
		SubmittedAt                                time.Time
	}
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		SELECT d.id, d.subject_id, s.name AS subject_name, d.subject_field_id AS field_id, sf.name AS field_name,
		       d.chapter_id, c.name AS chapter_name, d.subchapter_id, sc.name AS subchapter_name,
		       d.topic_id, t.name AS topic_name, d.grade_id, g.name AS grade_name,
		       d.difficulty, d.total_questions, d.correct_count, d.score::float8 AS score,
		       d.points, COALESCE(d.duration_seconds, 0) AS duration_seconds, d.coin_charged, d.submitted_at
		FROM daily_quizzes d
		JOIN master_data_items s       ON s.id = d.subject_id
		JOIN master_data_items sf      ON sf.id = d.subject_field_id
		JOIN master_data_items g       ON g.id = d.grade_id
		LEFT JOIN master_data_items c  ON c.id = d.chapter_id
		LEFT JOIN master_data_items sc ON sc.id = d.subchapter_id
		LEFT JOIN master_data_items t  ON t.id = d.topic_id
		WHERE `+where+`
		ORDER BY d.submitted_at DESC, d.id DESC
		LIMIT ? OFFSET ?`, pageArgs...).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	opt := func(id, name *string) *dailyquizmodel.Ref {
		if id == nil {
			return nil
		}
		r := &dailyquizmodel.Ref{ID: *id}
		if name != nil {
			r.Name = *name
		}
		return r
	}
	items := make([]dailyquizmodel.HistoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dailyquizmodel.HistoryItem{
			ID: row.ID, Subject: dailyquizmodel.Ref{ID: row.SubjectID, Name: row.SubjectName},
			SubjectField: dailyquizmodel.Ref{ID: row.FieldID, Name: row.FieldName},
			Chapter:      opt(row.ChapterID, row.ChapterName), Subchapter: opt(row.SubchapterID, row.SubchapterName),
			Topic: opt(row.TopicID, row.TopicName), Grade: dailyquizmodel.Ref{ID: row.GradeID, Name: row.GradeName},
			Difficulty: row.Difficulty, TotalQuestions: row.TotalQuestions, CorrectCount: row.CorrectCount,
			Score: row.Score, Points: row.Points, DurationSeconds: row.DurationSeconds, CoinCharged: row.CoinCharged,
			SubmittedAt: row.SubmittedAt,
		})
	}
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM daily_quizzes d WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// HistoryStats menghitung statistik Daily Quiz siswa, 1 query: jumlah / rata-rata nilai / point bulan
// period ([from, to)) dan sepanjang waktu, serta peringkat bulan itu di antara siswa sekelas (papan
// leaderboard_entries, aturan leaderboard). [Native SQL]
func (r *DailyQuizRepository) HistoryStats(ctx context.Context, studentID string, gradeID *string, period string, from, to time.Time) (dailyquizmodel.HistoryStats, error) {
	var row struct {
		MonthQuizzes, MonthPoints int
		MonthAverage              *float64
		AllQuizzes, AllPoints     int
		AllAverage                *float64
		Rank                      *int
		RankTotal                 int
	}
	err := r.db.WithContext(ctx).Raw(`
		WITH mine AS (
			SELECT d.score, d.points, d.submitted_at >= ? AND d.submitted_at < ? AS in_month
			FROM daily_quizzes d
			WHERE d.student_id = ? AND d.submitted_at IS NOT NULL
		),
		board AS (
			SELECT e.student_id, RANK() OVER (ORDER BY e.points DESC, e.duration_seconds ASC)::int AS rank
			FROM leaderboard_entries e
			JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
			WHERE e.board = 'daily_quiz' AND e.period = ? AND e.grade_id IS NOT DISTINCT FROM ?::uuid
		)
		SELECT COUNT(*) FILTER (WHERE in_month)::int                              AS month_quizzes,
		       COALESCE(SUM(points) FILTER (WHERE in_month), 0)::int              AS month_points,
		       ROUND((AVG(score) FILTER (WHERE in_month))::numeric, 2)::float8    AS month_average,
		       COUNT(*)::int                                                      AS all_quizzes,
		       COALESCE(SUM(points), 0)::int                                      AS all_points,
		       ROUND(AVG(score)::numeric, 2)::float8                              AS all_average,
		       (SELECT rank FROM board WHERE student_id = ?)                      AS rank,
		       (SELECT COUNT(*)::int FROM board)                                  AS rank_total
		FROM mine`, from, to, studentID, period, gradeID, studentID).Scan(&row).Error
	return dailyquizmodel.HistoryStats{
		Month:     period,
		ThisMonth: dailyquizmodel.Summary{Quizzes: row.MonthQuizzes, AverageScore: row.MonthAverage, Points: row.MonthPoints},
		AllTime:   dailyquizmodel.Summary{Quizzes: row.AllQuizzes, AverageScore: row.AllAverage, Points: row.AllPoints},
		Rank:      row.Rank, RankTotal: row.RankTotal,
	}, err
}
