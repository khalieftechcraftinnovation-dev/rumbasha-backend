// Package leaderboard (repository) menghitung peringkat siswa dari hasil quiz yang sudah di-submit.
//
// Di package lain, import dengan alias leaderboardrepo:
//
//	leaderboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/leaderboard"
package leaderboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	leaderboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/leaderboard"
)

// LeaderboardRepository membaca daily_quizzes & quiz_attempts untuk leaderboard.
type LeaderboardRepository struct {
	db *gorm.DB
}

// NewLeaderboardRepository membuat LeaderboardRepository. Dipanggil sekali di main.go.
func NewLeaderboardRepository(db *gorm.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db: db}
}

// entryRow menampung 1 baris hasil query leaderboard.
type entryRow struct {
	Rank               int
	StudentID          string
	RealName           string
	StudentCode        *string
	SchoolOrigin       *string
	EducationLevelID   *string
	EducationLevelName *string
	GradeID            *string
	GradeName          *string
	Points             int
	DurationSeconds    int
	QuizCount          int
	Score              *float64
}

func (row entryRow) toEntry() leaderboardmodel.Entry {
	e := leaderboardmodel.Entry{
		Rank: row.Rank, StudentID: row.StudentID, Name: row.RealName,
		StudentCode: row.StudentCode, SchoolOrigin: row.SchoolOrigin,
		Points: row.Points, DurationSeconds: row.DurationSeconds, QuizCount: row.QuizCount, Score: row.Score,
	}
	if row.EducationLevelID != nil && row.EducationLevelName != nil {
		e.EducationLevel = &leaderboardmodel.Ref{ID: *row.EducationLevelID, Name: *row.EducationLevelName}
	}
	if row.GradeID != nil && row.GradeName != nil {
		e.Grade = &leaderboardmodel.Ref{ID: *row.GradeID, Name: *row.GradeName}
	}
	return e
}

// rankedSelect = kolom akhir leaderboard: peringkat + data siswa + nama jenjang & kelas (snapshot di
// leaderboard_entries). Dipakai setelah CTE "ranked" yang berisi kolom leaderboard_entries + rank.
const rankedSelect = `
	SELECT r.rank, r.student_id, u.real_name, s.student_code, u.school_origin,
	       r.education_level_id, level.name AS education_level_name,
	       r.grade_id,           grade.name AS grade_name,
	       r.points, r.duration_seconds, r.quiz_count, r.score
	FROM ranked r
	JOIN      users             AS u     ON u.id       = r.student_id
	LEFT JOIN students          AS s     ON s.user_id  = r.student_id
	LEFT JOIN master_data_items AS level ON level.id   = r.education_level_id
	LEFT JOIN master_data_items AS grade ON grade.id   = r.grade_id`

// rankedFrom = isi CTE "ranked": baris leaderboard_entries yang cocok + peringkatnya.
// Peringkat: point tertinggi, lalu total waktu paling singkat; sama persis → peringkat sama.
// %s = kondisi WHERE.
const rankedFrom = `
	WITH ranked AS (
		SELECT e.student_id, e.education_level_id, e.grade_id,
		       e.points, e.duration_seconds, e.quiz_count, e.score,
		       RANK() OVER (ORDER BY e.points DESC, e.duration_seconds ASC)::int AS rank
		FROM leaderboard_entries e
		JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
		WHERE %s
	)`

// board menjalankan query leaderboard (CTE ranked + rankedSelect) 3x: 1 halaman, total peserta,
// dan posisi siswa meID (kalau diisi). rankedCTE berisi "WITH ... ranked AS (...)" dengan
// placeholder args.
func (r *LeaderboardRepository) board(ctx context.Context, rankedCTE string, args []any, page, perPage int, meID string) (leaderboardmodel.Board, error) {
	var b leaderboardmodel.Board

	var rows []entryRow
	pageArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	err := r.db.WithContext(ctx).
		Raw(rankedCTE+rankedSelect+`
			ORDER BY r.rank, u.real_name, r.student_id
			LIMIT ? OFFSET ?`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return b, err
	}
	b.Entries = make([]leaderboardmodel.Entry, 0, len(rows))
	for _, row := range rows {
		b.Entries = append(b.Entries, row.toEntry())
	}

	if err := r.db.WithContext(ctx).Raw(rankedCTE+`SELECT COUNT(*) FROM ranked`, args...).Scan(&b.Total).Error; err != nil {
		return b, err
	}

	if meID != "" {
		var me []entryRow
		meArgs := append(append([]any{}, args...), meID)
		if err := r.db.WithContext(ctx).Raw(rankedCTE+rankedSelect+` WHERE r.student_id = ?`, meArgs...).Scan(&me).Error; err != nil {
			return b, err
		}
		if len(me) > 0 {
			e := me[0].toEntry()
			b.Me = &e
		}
	}
	return b, nil
}

// DailyQuiz membaca leaderboard Daily Quiz 1 bulan (period "YYYY-MM") dari leaderboard_entries,
// difilter jenjang / kelas (snapshot saat siswa submit). [Native SQL, index leaderboard_entries_daily_idx]
func (r *LeaderboardRepository) DailyQuiz(ctx context.Context, period string, f leaderboardmodel.DailyFilter, meID string) (leaderboardmodel.Board, error) {
	conds := []string{"e.board = 'daily_quiz'", "e.period = ?"}
	args := []any{period}
	if f.EducationLevelID != "" {
		conds = append(conds, "e.education_level_id = ?")
		args = append(args, f.EducationLevelID)
	}
	if f.GradeID != "" {
		conds = append(conds, "e.grade_id = ?")
		args = append(args, f.GradeID)
	}
	return r.board(ctx, fmt.Sprintf(rankedFrom, strings.Join(conds, " AND ")), args, f.Page, f.PerPage, meID)
}

// Quiz membaca leaderboard 1 quiz Uji Kom / Tryout dari leaderboard_entries (1 quiz = 1 papan,
// tidak digabung). [Native SQL, index leaderboard_entries_quiz_idx]
func (r *LeaderboardRepository) Quiz(ctx context.Context, quizID string, page, perPage int, meID string) (leaderboardmodel.Board, error) {
	return r.board(ctx, fmt.Sprintf(rankedFrom, "e.quiz_id = ?"), []any{quizID}, page, perPage, meID)
}

// FindQuiz mengambil info quiz Uji Kom / Tryout (belum dihapus). nil kalau tidak ada. [Native SQL]
func (r *LeaderboardRepository) FindQuiz(ctx context.Context, quizID string) (*leaderboardmodel.Quiz, error) {
	var rows []struct {
		ID, Name, Type, Status, GradeID    string
		StartAt, EndAt                     time.Time
		PrizeRank1, PrizeRank2, PrizeRank3 int
		PrizesDistributedAt                *time.Time
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT id, name, type, status, grade_id, start_at, end_at,
			       prize_rank_1 AS prize_rank1, prize_rank_2 AS prize_rank2, prize_rank_3 AS prize_rank3,
			       prizes_distributed_at
			FROM quizzes
			WHERE id = ? AND deleted_at IS NULL AND type IN ('uji_kom', 'tryout')`, quizID).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	row := rows[0]
	return &leaderboardmodel.Quiz{
		ID: row.ID, Name: row.Name, Type: row.Type, Status: row.Status, GradeID: row.GradeID,
		StartAt: row.StartAt, EndAt: row.EndAt,
		PrizeRank1: row.PrizeRank1, PrizeRank2: row.PrizeRank2, PrizeRank3: row.PrizeRank3,
		PrizesDistributedAt: row.PrizesDistributedAt,
	}, nil
}

// FindUserGrade mengambil kelas akun (siswa). nil kalau belum diisi. [Native SQL, primary key]
func (r *LeaderboardRepository) FindUserGrade(ctx context.Context, userID string) (*string, error) {
	var rows []struct{ GradeID *string }
	if err := r.db.WithContext(ctx).Raw(`SELECT grade_id FROM users WHERE id = ?`, userID).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].GradeID, nil
}

// RecordPoints mencatat point 1 pengerjaan ke leaderboard, di dalam transaksi submit (tx):
//
//  1. leaderboard_entries: tambah ke baris siswa di papan itu (buat baris baru kalau belum ada).
//     Snapshot jenjang & kelas diambil dari akun siswa saat ini.
//  2. leaderboard_point_logs: catat riwayatnya (1 pengerjaan hanya bisa tercatat 1x).
//
// Dipanggil repository quiz & daily quiz supaya submit + leaderboard berhasil/gagal bersama.
// [Native SQL, 2 query]
func RecordPoints(tx *gorm.DB, rec leaderboardmodel.PointRecord) error {
	var entryID string
	err := tx.Raw(`
		INSERT INTO leaderboard_entries (board, period, quiz_id, student_id, education_level_id, grade_id,
		                                 points, duration_seconds, quiz_count, score, last_submitted_at)
		SELECT ?, ?::varchar, ?::uuid, u.id, u.education_level_id, u.grade_id, ?, ?, 1, ?, ?
		FROM users u WHERE u.id = ?
		ON CONFLICT ON CONSTRAINT leaderboard_entries_unique DO UPDATE SET
			points             = leaderboard_entries.points + EXCLUDED.points,
			duration_seconds   = leaderboard_entries.duration_seconds + EXCLUDED.duration_seconds,
			quiz_count         = leaderboard_entries.quiz_count + 1,
			score              = EXCLUDED.score,
			education_level_id = EXCLUDED.education_level_id,
			grade_id           = EXCLUDED.grade_id,
			last_submitted_at  = EXCLUDED.last_submitted_at,
			updated_at         = NOW()
		RETURNING id`,
		rec.Board, rec.Period, rec.QuizID, rec.Points, rec.DurationSeconds, rec.Score, rec.SubmittedAt, rec.StudentID).
		Scan(&entryID).Error
	if err != nil {
		return fmt.Errorf("gagal mencatat leaderboard: %w", err)
	}
	if entryID == "" {
		return fmt.Errorf("gagal mencatat leaderboard: siswa %s tidak ditemukan", rec.StudentID)
	}

	err = tx.Exec(`
		INSERT INTO leaderboard_point_logs (entry_id, student_id, source_type, source_id, points, duration_seconds, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entryID, rec.StudentID, rec.SourceType, rec.SourceID, rec.Points, rec.DurationSeconds, rec.SubmittedAt).Error
	if err != nil {
		return fmt.Errorf("gagal mencatat riwayat point: %w", err)
	}
	return nil
}

// DailyPrize mengambil hadiah juara bulanan Daily Quiz untuk 1 bulan: snapshot kalau sudah
// dibagikan (daily_quiz_prize_periods), selain itu pengaturan saat ini. [Native SQL, 2 primary key]
func (r *LeaderboardRepository) DailyPrize(ctx context.Context, period string) (leaderboardmodel.DailyPrize, error) {
	var rows []leaderboardmodel.DailyPrize
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT COALESCE(p.prize_rank_1, s.prize_rank_1) AS prize_rank1,
			       COALESCE(p.prize_rank_2, s.prize_rank_2) AS prize_rank2,
			       COALESCE(p.prize_rank_3, s.prize_rank_3) AS prize_rank3,
			       p.distributed_at
			FROM daily_quiz_settings s
			LEFT JOIN daily_quiz_prize_periods p ON p.period = ?
			WHERE s.id = 1`, period).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return leaderboardmodel.DailyPrize{}, err
	}
	return rows[0], nil
}
