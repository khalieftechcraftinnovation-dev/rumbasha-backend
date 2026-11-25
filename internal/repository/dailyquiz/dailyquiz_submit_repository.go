package dailyquiz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/leaderboard"
	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/leaderboard"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/scoring"
)

// FindSubmission mengambil tahap & hasil Daily Quiz milik siswa. Tidak ada / bukan miliknya →
// ErrDailyQuizNotFound. [Native SQL, primary key]
func (r *DailyQuizRepository) FindSubmission(ctx context.Context, quizID, studentID string) (dailyquizmodel.Submission, error) {
	var rows []struct {
		OpenedAt, SubmittedAt *time.Time
		Score                 *float64
		CorrectCount          *int
		Answers               *string // JSON
		DurationSeconds       *int
		Points                *int
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT opened_at, submitted_at, score, correct_count, answers::text AS answers, duration_seconds, points
			FROM daily_quizzes WHERE id = ? AND student_id = ?`, quizID, studentID).
		Scan(&rows).Error
	if err != nil {
		return dailyquizmodel.Submission{}, err
	}
	if len(rows) == 0 {
		return dailyquizmodel.Submission{}, dailyquizmodel.ErrDailyQuizNotFound
	}
	row := rows[0]
	s := dailyquizmodel.Submission{OpenedAt: row.OpenedAt, SubmittedAt: row.SubmittedAt}
	if row.Score != nil {
		s.Score = *row.Score
	}
	if row.Points != nil {
		s.Points = *row.Points
	}
	if row.DurationSeconds != nil {
		s.DurationSeconds = *row.DurationSeconds
	}
	if row.CorrectCount != nil {
		s.CorrectCount = *row.CorrectCount
	}
	if row.Answers != nil {
		if err := json.Unmarshal([]byte(*row.Answers), &s.Answers); err != nil {
			return s, fmt.Errorf("jawaban daily quiz rusak: %w", err)
		}
	}
	return s, nil
}

// Submit menyimpan jawaban & nilai Daily Quiz yang sudah dibuka (1 UPDATE). Hanya sekali: kalau
// belum dibuka / sudah pernah submit (termasuk 2 request bersamaan), ok = false. [Native SQL]
//
// Dalam transaksi yang sama, point dicatat ke leaderboard Daily Quiz bulan itu
// (leaderboardrepo.RecordPoints): submit & leaderboard berhasil bersama atau gagal bersama.
func (r *DailyQuizRepository) Submit(ctx context.Context, quizID, studentID string, result scoring.Result, now time.Time, rec leaderboardmodel.PointRecord) (ok bool, err error) {
	answers, err := json.Marshal(result.Answers)
	if err != nil {
		return false, err
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	res := tx.Exec(`
		UPDATE daily_quizzes
		SET submitted_at = ?, score = ?, correct_count = ?, answers = ?::jsonb, duration_seconds = ?, points = ?
		WHERE id = ? AND student_id = ? AND opened_at IS NOT NULL AND submitted_at IS NULL`,
		now, result.Score, result.CorrectCount, string(answers), result.DurationSeconds, result.Points, quizID, studentID)
	if res.Error != nil || res.RowsAffected != 1 {
		return false, res.Error
	}
	if err := leaderboardrepo.RecordPoints(tx, rec); err != nil {
		return false, err
	}
	return true, tx.Commit().Error
}
