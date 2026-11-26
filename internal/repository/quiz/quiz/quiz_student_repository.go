package quiz

import (
	"context"
	"strings"
	"time"

	quizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/quiz"
)

// studentQuizWhere menyusun kondisi tab / cari / mapel untuk quiz alias z.
func studentQuizWhere(f quizmodel.StudentQuizFilter) (string, []any) {
	conds := []string{"z.type IN ?"}
	args := []any{f.Types}
	if f.SubjectID != "" {
		conds = append(conds, "z.subject_id = ?")
		args = append(args, f.SubjectID)
	}
	if f.Search != "" {
		conds = append(conds, "z.name ILIKE ?")
		args = append(args, "%"+likeEscaper.Replace(f.Search)+"%")
	}
	return strings.Join(conds, " AND "), args
}

// StudentUpcoming mengambil bagian "Akan datang": quiz publish untuk kelas siswa yang event-nya belum
// selesai dan belum dikumpulkan siswa, urut paling dekat dimulai. [Native SQL, index quizzes_announcement_idx]
func (r *QuizRepository) StudentUpcoming(ctx context.Context, studentID, gradeID string, now time.Time, f quizmodel.StudentQuizFilter) ([]quizmodel.UpcomingQuiz, int, error) {
	where, args := studentQuizWhere(f)
	base := `
		FROM quizzes z
		LEFT JOIN quiz_attempts a ON a.quiz_id = z.id AND a.student_id = ?
		WHERE z.deleted_at IS NULL AND z.status = 'published' AND z.grade_id = ? AND z.end_at > ?
		  AND a.submitted_at IS NULL AND ` + where
	baseArgs := append([]any{studentID, gradeID, now}, args...)

	var rows []struct {
		ID, Name, Type, SubjectID, SubjectName string
		FieldID, FieldName                     *string
		ChapterID, ChapterName                 *string
		TotalQuestions, DurationMinutes        int
		CoinCost                               int
		StartAt, EndAt                         time.Time
		AttemptStartedAt                       *time.Time
	}
	pageArgs := append(append([]any{}, baseArgs...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(`
		SELECT z.id, z.name, z.type, z.subject_id, s.name AS subject_name,
		       z.subject_field_id AS field_id, sf.name AS field_name, z.chapter_id, c.name AS chapter_name,
		       z.total_questions, z.duration_minutes, z.coin_cost, z.start_at, z.end_at,
		       a.started_at AS attempt_started_at`+strings.Replace(base, "FROM quizzes z", `FROM quizzes z
		JOIN master_data_items s       ON s.id = z.subject_id
		LEFT JOIN master_data_items sf ON sf.id = z.subject_field_id
		LEFT JOIN master_data_items c  ON c.id = z.chapter_id`, 1)+`
		ORDER BY z.start_at, z.id
		LIMIT ? OFFSET ?`, pageArgs...).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]quizmodel.UpcomingQuiz, 0, len(rows))
	for _, row := range rows {
		item := quizmodel.UpcomingQuiz{
			ID: row.ID, Name: row.Name, Type: row.Type, Subject: quizmodel.Ref{ID: row.SubjectID, Name: row.SubjectName},
			TotalQuestions: row.TotalQuestions, DurationMinutes: row.DurationMinutes, CoinCost: row.CoinCost,
			StartAt: row.StartAt, EndAt: row.EndAt, AttemptStartedAt: row.AttemptStartedAt,
		}
		if row.FieldID != nil {
			item.SubjectField = &quizmodel.Ref{ID: *row.FieldID, Name: deref(row.FieldName)}
		}
		if row.ChapterID != nil {
			item.Chapter = &quizmodel.Ref{ID: *row.ChapterID, Name: deref(row.ChapterName)}
		}
		items = append(items, item)
	}
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) `+base, baseArgs...).Scan(&total).Error
	return items, total, err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// doneRow = 1 baris "Sudah dikerjakan" + peringkat.
type doneRow struct {
	QuizID, Name, Type, SubjectID, SubjectName string
	StartAt, EndAt                             time.Time
	TotalQuestions                             int
	SubmittedAt                                time.Time
	Score                                      float64
	CorrectCount, Points, DurationSeconds      int
	Rank, RankTotal                            int
}

func (row doneRow) toDone() quizmodel.DoneQuiz {
	return quizmodel.DoneQuiz{
		QuizID: row.QuizID, Name: row.Name, Type: row.Type, Subject: quizmodel.Ref{ID: row.SubjectID, Name: row.SubjectName},
		StartAt: row.StartAt, EndAt: row.EndAt, TotalQuestions: row.TotalQuestions, SubmittedAt: row.SubmittedAt,
		Score: row.Score, CorrectCount: row.CorrectCount, Points: row.Points, DurationSeconds: row.DurationSeconds,
		Rank: row.Rank, RankTotal: row.RankTotal,
	}
}

// doneSelect = CTE "mine" (pengerjaan siswa yang dipilih) → peringkat setiap quiz-nya (RANK per quiz,
// aturan leaderboard: point tertinggi, lalu waktu tersingkat; akun terhapus tidak dihitung; index
// quiz_attempts_leaderboard_idx) → kolom akhir. %s = isi CTE mine (kolom quiz_id).
const doneSelect = `
	WITH mine AS (%s),
	ranked AS (
		SELECT a.quiz_id, a.student_id,
		       RANK() OVER (PARTITION BY a.quiz_id ORDER BY a.points DESC, a.duration_seconds ASC)::int AS rank,
		       COUNT(*) OVER (PARTITION BY a.quiz_id)::int AS rank_total
		FROM quiz_attempts a
		JOIN users u ON u.id = a.student_id AND u.deleted_at IS NULL
		WHERE a.submitted_at IS NOT NULL AND a.quiz_id IN (SELECT quiz_id FROM mine)
	)
	SELECT z.id AS quiz_id, z.name, z.type, z.subject_id, s.name AS subject_name, z.start_at, z.end_at, z.total_questions,
	       a.submitted_at, a.score::float8 AS score, a.correct_count, a.points, a.duration_seconds,
	       rk.rank, rk.rank_total
	FROM mine m
	JOIN quiz_attempts a     ON a.quiz_id = m.quiz_id AND a.student_id = ?
	JOIN quizzes z           ON z.id = m.quiz_id
	JOIN master_data_items s ON s.id = z.subject_id
	JOIN ranked rk           ON rk.quiz_id = m.quiz_id AND rk.student_id = a.student_id
	ORDER BY a.submitted_at DESC, z.id`

// StudentHistory mengambil 1 halaman "Sudah dikerjakan" (terbaru dulu) + peringkat di setiap quiz,
// 1 query + COUNT. Quiz yang sudah dihapus admin tidak tampil. [Native SQL, index quiz_attempts_student_history_idx]
func (r *QuizRepository) StudentHistory(ctx context.Context, studentID string, f quizmodel.StudentQuizFilter) ([]quizmodel.DoneQuiz, int, error) {
	where, args := studentQuizWhere(f)
	minePart := `
		SELECT a.quiz_id FROM quiz_attempts a
		JOIN quizzes z ON z.id = a.quiz_id AND z.deleted_at IS NULL
		WHERE a.student_id = ? AND a.submitted_at IS NOT NULL AND ` + where
	mineArgs := append([]any{studentID}, args...)

	var rows []doneRow
	queryArgs := append(append([]any{}, mineArgs...), f.PerPage, (f.Page-1)*f.PerPage, studentID)
	err := r.db.WithContext(ctx).Raw(strings.Replace(doneSelect, "%s",
		minePart+` ORDER BY a.submitted_at DESC, a.quiz_id LIMIT ? OFFSET ?`, 1), queryArgs...).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]quizmodel.DoneQuiz, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDone())
	}
	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM (`+minePart+`) x`, mineArgs...).Scan(&total).Error
	return items, total, err
}

// StudentStats menghitung statistik Uji Kom & Tryout siswa (jumlah sesi, rata-rata nilai, total point;
// tab ikut f.Types) + Tryout terakhir yang dikumpulkan beserta peringkatnya. 2 query. [Native SQL]
func (r *QuizRepository) StudentStats(ctx context.Context, studentID string, f quizmodel.StudentQuizFilter) (quizmodel.StudentQuizStats, error) {
	where, args := studentQuizWhere(f)
	var row struct {
		Sessions, TotalPoints int
		AverageScore          *float64
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)::int AS sessions, ROUND(AVG(a.score)::numeric, 2)::float8 AS average_score,
		       COALESCE(SUM(a.points), 0)::int AS total_points
		FROM quiz_attempts a
		JOIN quizzes z ON z.id = a.quiz_id AND z.deleted_at IS NULL
		WHERE a.student_id = ? AND a.submitted_at IS NOT NULL AND `+where, append([]any{studentID}, args...)...).
		Scan(&row).Error
	if err != nil {
		return quizmodel.StudentQuizStats{}, err
	}
	stats := quizmodel.StudentQuizStats{Sessions: row.Sessions, AverageScore: row.AverageScore, TotalPoints: row.TotalPoints}

	var last []doneRow
	err = r.db.WithContext(ctx).Raw(strings.Replace(doneSelect, "%s", `
		SELECT a.quiz_id FROM quiz_attempts a
		JOIN quizzes z ON z.id = a.quiz_id AND z.deleted_at IS NULL AND z.type = 'tryout'
		WHERE a.student_id = ? AND a.submitted_at IS NOT NULL
		ORDER BY a.submitted_at DESC LIMIT 1`, 1), studentID, studentID).Scan(&last).Error
	if err != nil {
		return stats, err
	}
	if len(last) > 0 {
		d := last[0].toDone()
		stats.LastTryout = &d
	}
	return stats, nil
}
