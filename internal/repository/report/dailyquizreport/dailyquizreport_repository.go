// Package dailyquizreport (repository) menghitung report Daily Quiz per siswa dari daily_quizzes
// yang sudah di-submit.
//
// Akun yang sudah dihapus tidak ikut (sama dengan leaderboard).
//
// Di package lain, import dengan alias dailyquizreportrepo:
//
//	dailyquizreportrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/report/dailyquizreport"
package dailyquizreport

import (
	"context"
	"strings"

	"gorm.io/gorm"

	dailyquizreportmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/report/dailyquizreport"
)

// DailyQuizReportRepository membaca data report Daily Quiz.
type DailyQuizReportRepository struct {
	db *gorm.DB
}

// NewDailyQuizReportRepository membuat DailyQuizReportRepository. Dipanggil sekali di main.go.
func NewDailyQuizReportRepository(db *gorm.DB) *DailyQuizReportRepository {
	return &DailyQuizReportRepository{db: db}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// statsColumns = rekap Daily Quiz (alias d) yang dijumlahkan per kelompok (GROUP BY).
const statsColumns = `
	COUNT(*)::int                                           AS total_quizzes,
	COUNT(*) FILTER (WHERE d.difficulty = 'easy')::int      AS easy,
	COUNT(*) FILTER (WHERE d.difficulty = 'medium')::int    AS medium,
	COUNT(*) FILTER (WHERE d.difficulty = 'hard')::int      AS hard,
	COALESCE(ROUND(AVG(d.score)::numeric, 2), 0)::float8                 AS average_score,
	COALESCE(ROUND(AVG(d.duration_seconds)), 0)::int                     AS average_duration_seconds,
	COALESCE(SUM(d.total_questions), 0)::int                             AS total_questions,
	COALESCE(SUM(d.correct_count), 0)::int                               AS correct,
	COALESCE(SUM(d.points), 0)::int                                      AS total_points,
	COALESCE(SUM(d.duration_seconds), 0)::int                            AS total_duration_seconds`

// studentRow menampung 1 baris report (flat, untuk Scan).
type studentRow struct {
	Rank               int
	StudentID, Name    string
	StudentCode        *string
	LevelID, LevelName *string
	GradeID, GradeName *string
	dailyquizreportmodel.Stats
}

func ref(id, name *string) *dailyquizreportmodel.Ref {
	if id == nil || name == nil {
		return nil
	}
	return &dailyquizreportmodel.Ref{ID: *id, Name: *name}
}

// List mengambil 1 halaman siswa urut peringkat + total (setelah pencarian). [Native SQL]
//
//  1. agg: rekap per siswa dari Daily Quiz yang di-submit dalam periode (dan mapel kalau difilter),
//     dibaca dari index daily_quizzes_report_idx.
//  2. ranked: + data siswa (filter jenjang/kelas akun), peringkat dihitung di dalam filter:
//     total point tertinggi, lalu total waktu paling singkat; sama persis → peringkat sama
//     (aturan sama dengan leaderboard). Pencarian nama TIDAK mengubah peringkat.
func (r *DailyQuizReportRepository) List(ctx context.Context, f dailyquizreportmodel.ListFilter) ([]dailyquizreportmodel.StudentRow, int, error) {
	aggWhere := "d.submitted_at >= ? AND d.submitted_at < ?"
	args := []any{f.From, f.To}
	if f.SubjectID != "" {
		aggWhere += " AND d.subject_id = ?"
		args = append(args, f.SubjectID)
	}
	userWhere := "u.deleted_at IS NULL"
	if f.EducationLevelID != "" {
		userWhere += " AND u.education_level_id = ?"
		args = append(args, f.EducationLevelID)
	}
	if f.GradeID != "" {
		userWhere += " AND u.grade_id = ?"
		args = append(args, f.GradeID)
	}
	ranked := `
		WITH agg AS (
			SELECT d.student_id, ` + statsColumns + `
			FROM daily_quizzes d
			WHERE d.submitted_at IS NOT NULL AND ` + aggWhere + `
			GROUP BY d.student_id
		),
		ranked AS (
			SELECT a.*, u.real_name AS name, s.student_code,
			       u.education_level_id AS level_id, l.name AS level_name,
			       u.grade_id, g.name AS grade_name,
			       RANK() OVER (ORDER BY a.total_points DESC, a.total_duration_seconds ASC)::int AS rank
			FROM agg a
			JOIN users u                ON u.id = a.student_id AND ` + userWhere + `
			LEFT JOIN students s        ON s.user_id = a.student_id
			LEFT JOIN master_data_items l ON l.id = u.education_level_id
			LEFT JOIN master_data_items g ON g.id = u.grade_id
		)`

	where := "TRUE"
	if f.Search != "" {
		like := "%" + likeEscaper.Replace(f.Search) + "%"
		where = "(name ILIKE ? OR student_code ILIKE ?)"
		args = append(args, like, like)
	}

	var rows []studentRow
	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	err := r.db.WithContext(ctx).Raw(ranked+`
		SELECT * FROM ranked WHERE `+where+`
		ORDER BY rank, name, student_id
		LIMIT ? OFFSET ?`, pageArgs...).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]dailyquizreportmodel.StudentRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, dailyquizreportmodel.StudentRow{
			Rank: row.Rank, StudentID: row.StudentID, Name: row.Name, StudentCode: row.StudentCode,
			EducationLevel: ref(row.LevelID, row.LevelName), Grade: ref(row.GradeID, row.GradeName),
			Stats: row.Stats,
		})
	}

	if len(items) < f.PerPage && (len(items) > 0 || f.Page == 1) {
		return items, (f.Page-1)*f.PerPage + len(items), nil
	}
	var total int
	err = r.db.WithContext(ctx).Raw(ranked+` SELECT COUNT(*) FROM ranked WHERE `+where, args...).Scan(&total).Error
	return items, total, err
}

// FindStudent mengambil data siswa (akun role student yang belum dihapus). Tidak ada →
// ErrStudentNotFound. [Native SQL, primary key]
func (r *DailyQuizReportRepository) FindStudent(ctx context.Context, id string) (dailyquizreportmodel.Student, error) {
	var rows []struct {
		ID, Name           string
		StudentCode        *string
		LevelID, LevelName *string
		GradeID, GradeName *string
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.real_name AS name, s.student_code,
		       u.education_level_id AS level_id, l.name AS level_name, u.grade_id, g.name AS grade_name
		FROM users u
		LEFT JOIN students s          ON s.user_id = u.id
		LEFT JOIN master_data_items l ON l.id = u.education_level_id
		LEFT JOIN master_data_items g ON g.id = u.grade_id
		WHERE u.id = ? AND u.role = 'student' AND u.deleted_at IS NULL`, id).
		Scan(&rows).Error
	if err != nil {
		return dailyquizreportmodel.Student{}, err
	}
	if len(rows) == 0 {
		return dailyquizreportmodel.Student{}, dailyquizreportmodel.ErrStudentNotFound
	}
	row := rows[0]
	return dailyquizreportmodel.Student{
		ID: row.ID, Name: row.Name, StudentCode: row.StudentCode,
		EducationLevel: ref(row.LevelID, row.LevelName), Grade: ref(row.GradeID, row.GradeName),
	}, nil
}

// StudentSubjects merekap Daily Quiz 1 siswa dalam periode PER MAPEL (urut jumlah quiz terbanyak),
// index daily_quizzes_student_report_idx. [Native SQL, 1 query]
func (r *DailyQuizReportRepository) StudentSubjects(ctx context.Context, studentID string, p dailyquizreportmodel.Period) ([]dailyquizreportmodel.SubjectRow, error) {
	var rows []struct {
		SubjectID, SubjectName string
		dailyquizreportmodel.Stats
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT d.subject_id, sub.name AS subject_name, `+statsColumns+`
		FROM daily_quizzes d
		JOIN master_data_items sub ON sub.id = d.subject_id
		WHERE d.student_id = ? AND d.submitted_at IS NOT NULL AND d.submitted_at >= ? AND d.submitted_at < ?
		GROUP BY d.subject_id, sub.name, sub.sort_order
		ORDER BY total_quizzes DESC, sub.sort_order, sub.name`, studentID, p.From, p.To).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]dailyquizreportmodel.SubjectRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, dailyquizreportmodel.SubjectRow{
			Subject: dailyquizreportmodel.Ref{ID: row.SubjectID, Name: row.SubjectName}, Stats: row.Stats,
		})
	}
	return result, nil
}

// StudentTotal merekap SEMUA Daily Quiz 1 siswa dalam periode (gabungan semua mapel). [Native SQL, 1 query]
func (r *DailyQuizReportRepository) StudentTotal(ctx context.Context, studentID string, p dailyquizreportmodel.Period) (dailyquizreportmodel.Stats, error) {
	var rows []dailyquizreportmodel.Stats
	err := r.db.WithContext(ctx).Raw(`
		SELECT `+statsColumns+`
		FROM daily_quizzes d
		WHERE d.student_id = ? AND d.submitted_at IS NOT NULL AND d.submitted_at >= ? AND d.submitted_at < ?`,
		studentID, p.From, p.To).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return dailyquizreportmodel.Stats{}, err
	}
	return rows[0], nil
}
