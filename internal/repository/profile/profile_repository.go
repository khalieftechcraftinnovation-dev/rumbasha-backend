// Package profile (repository) mengakses data halaman Profil siswa (users + students + ringkasan
// dari leaderboard_entries, daily_quizzes, quiz_attempts).
//
// Di package lain, import dengan alias profilerepo:
//
//	profilerepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/profile"
package profile

import (
	"context"
	"time"

	"gorm.io/gorm"

	profilemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/profile"
)

// ProfileRepository mengakses data profil siswa.
type ProfileRepository struct {
	db *gorm.DB
}

// NewProfileRepository membuat ProfileRepository. Dipanggil sekali di main.go.
func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

type profileRow struct {
	ID, Username, RealName     string
	StudentCode                *string
	Email, Phone, ParentPhone  *string
	SchoolOrigin, TargetSchool *string
	DateOfBirth                *time.Time
	LevelID, LevelName         *string
	GradeID, GradeName         *string
	BranchID, BranchName       *string
	AvatarPath, AvatarURL      *string
	CoinBalance                int
	JoinDate                   time.Time
	EndDate                    *time.Time
}

func ref(id, name *string) *profilemodel.Ref {
	if id == nil {
		return nil
	}
	r := &profilemodel.Ref{ID: *id}
	if name != nil {
		r.Name = *name
	}
	return r
}

// Find mengambil profil 1 siswa (role student, belum dihapus). Tidak ada → ErrNotFound.
// [Native SQL, primary key; JOIN jenjang, kelas, cabang lewat primary key]
func (r *ProfileRepository) Find(ctx context.Context, userID string) (profilemodel.Profile, error) {
	var rows []profileRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.username, u.real_name, s.student_code, u.email, u.phone, u.parent_phone,
		       u.school_origin, s.target_school, s.date_of_birth,
		       u.education_level_id AS level_id, l.name AS level_name,
		       u.grade_id, g.name AS grade_name, u.branch_id, b.name AS branch_name,
		       u.avatar_path, u.avatar_url, COALESCE(s.coin_balance, 0) AS coin_balance, u.join_date, u.end_date
		FROM users u
		LEFT JOIN students s          ON s.user_id = u.id
		LEFT JOIN master_data_items l ON l.id = u.education_level_id
		LEFT JOIN master_data_items g ON g.id = u.grade_id
		LEFT JOIN branches b          ON b.id = u.branch_id
		WHERE u.id = ? AND u.role = 'student' AND u.deleted_at IS NULL`, userID).Scan(&rows).Error
	if err != nil {
		return profilemodel.Profile{}, err
	}
	if len(rows) == 0 {
		return profilemodel.Profile{}, profilemodel.ErrNotFound
	}
	row := rows[0]
	return profilemodel.Profile{
		ID: row.ID, Username: row.Username, RealName: row.RealName, StudentCode: row.StudentCode,
		Email: row.Email, Phone: row.Phone, ParentPhone: row.ParentPhone,
		SchoolOrigin: row.SchoolOrigin, TargetSchool: row.TargetSchool, DateOfBirth: row.DateOfBirth,
		EducationLevel: ref(row.LevelID, row.LevelName), Grade: ref(row.GradeID, row.GradeName), Branch: ref(row.BranchID, row.BranchName),
		AvatarPath: row.AvatarPath, AvatarURL: row.AvatarURL, CoinBalance: row.CoinBalance,
		JoinDate: row.JoinDate, EndDate: row.EndDate,
	}, nil
}

// Stats menghitung ringkasan Profil, 1 query:
//   - peringkat Daily Quiz bulan ini di antara siswa dengan kelas yang sama (aturan leaderboard:
//     point tertinggi, lalu waktu tersingkat; index leaderboard_entries_daily_idx);
//   - point Daily Quiz bulan ini & total semua point (leaderboard_entries milik siswa);
//   - quiz yang dikumpulkan bulan ini (Daily Quiz + Uji Kom + Tryout).
//
// [Native SQL]
func (r *ProfileRepository) Stats(ctx context.Context, userID string, gradeID *string, month string, from, to time.Time) (profilemodel.Stats, error) {
	var row struct {
		DailyRank                     *int
		DailyRankTotal                int
		PointsThisMonth, TotalPoints  int
		DailyThisMonth, QuizThisMonth int
	}
	err := r.db.WithContext(ctx).Raw(`
		WITH board AS (
			SELECT e.student_id,
			       RANK() OVER (ORDER BY e.points DESC, e.duration_seconds ASC)::int AS rank
			FROM leaderboard_entries e
			JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
			WHERE e.board = 'daily_quiz' AND e.period = ? AND e.grade_id IS NOT DISTINCT FROM ?::uuid
		)
		SELECT (SELECT rank FROM board WHERE student_id = ?)                                         AS daily_rank,
		       (SELECT COUNT(*)::int FROM board)                                                     AS daily_rank_total,
		       (SELECT COALESCE(SUM(points), 0)::int FROM leaderboard_entries
		         WHERE student_id = ? AND board = 'daily_quiz' AND period = ?)                       AS points_this_month,
		       (SELECT COALESCE(SUM(points), 0)::int FROM leaderboard_entries WHERE student_id = ?)  AS total_points,
		       (SELECT COUNT(*)::int FROM daily_quizzes
		         WHERE student_id = ? AND submitted_at >= ? AND submitted_at < ?)                    AS daily_this_month,
		       (SELECT COUNT(*)::int FROM quiz_attempts
		         WHERE student_id = ? AND submitted_at >= ? AND submitted_at < ?)                    AS quiz_this_month`,
		month, gradeID, userID, userID, month, userID, userID, from, to, userID, from, to).
		Scan(&row).Error
	return profilemodel.Stats{
		Month: month, DailyRank: row.DailyRank, DailyRankTotal: row.DailyRankTotal,
		PointsThisMonth: row.PointsThisMonth, TotalPoints: row.TotalPoints,
		QuizzesThisMonth: row.DailyThisMonth + row.QuizThisMonth,
	}, err
}

// Update mengganti data diri yang boleh diubah siswa (nama, email, no HP, tanggal lahir), 1 transaksi.
// [Native SQL, primary key]
func (r *ProfileRepository) Update(ctx context.Context, userID string, u profilemodel.Update, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			UPDATE users SET real_name = ?, email = ?, phone = ?, updated_at = ?
			WHERE id = ? AND role = 'student' AND deleted_at IS NULL`, u.RealName, u.Email, u.Phone, now, userID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return profilemodel.ErrNotFound
		}
		return tx.Exec(`UPDATE students SET date_of_birth = ?, updated_at = ? WHERE user_id = ?`, u.DateOfBirth, now, userID).Error
	})
}

// UpdateParentPhone mengganti no HP orang tua / wali (nil = dikosongkan). [Native SQL, primary key]
func (r *ProfileRepository) UpdateParentPhone(ctx context.Context, userID string, phone *string, now time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE users SET parent_phone = ?, updated_at = ?
		WHERE id = ? AND role = 'student' AND deleted_at IS NULL`, phone, now, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return profilemodel.ErrNotFound
	}
	return nil
}

// UpdateAvatar mengganti (atau mengosongkan, kalau path nil) foto profil. [Native SQL, primary key]
func (r *ProfileRepository) UpdateAvatar(ctx context.Context, userID string, path, url *string, now time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE users SET avatar_path = ?, avatar_url = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`, path, url, now, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return profilemodel.ErrNotFound
	}
	return nil
}
