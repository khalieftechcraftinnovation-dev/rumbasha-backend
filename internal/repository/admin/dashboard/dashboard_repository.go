// Package dashboard reads aggregated data for the Admin dashboard.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	dashboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/dashboard"
)

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

var searchLikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *DashboardRepository) Search(ctx context.Context, query string) (dashboardmodel.SearchResults, error) {
	const searchQuery = `
	WITH input AS (
		SELECT ?::text AS pattern
	),
	user_results AS (
		SELECT u.id, COALESCE(NULLIF(u.real_name, ''), u.username) AS name,
		       u.username, u.role, u.status, u.created_at
		FROM users u CROSS JOIN input i
		WHERE u.deleted_at IS NULL
		  AND (u.real_name ILIKE i.pattern OR u.username ILIKE i.pattern OR u.email ILIKE i.pattern)
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT 5
	),
	question_results AS (
		SELECT q.id, q.code, LEFT(q.question_text, 180) AS preview, q.type, q.created_at,
		       level.id AS level_id, level.name AS level_name,
		       grade.id AS grade_id, grade.name AS grade_name
		FROM questions q
		CROSS JOIN input i
		JOIN master_data_items level ON level.id = q.education_level_id
		JOIN master_data_items grade ON grade.id = q.grade_id
		WHERE q.deleted_at IS NULL AND (q.code ILIKE i.pattern OR q.question_text ILIKE i.pattern)
		ORDER BY q.created_at DESC, q.id DESC
		LIMIT 5
	),
	module_results AS (
		SELECT m.id, m.name, m.status, m.created_at,
		       level.id AS level_id, level.name AS level_name,
		       grade.id AS grade_id, grade.name AS grade_name
		FROM learning_modules m
		CROSS JOIN input i
		JOIN master_data_items level ON level.id = m.education_level_id
		JOIN master_data_items grade ON grade.id = m.grade_id
		WHERE m.deleted_at IS NULL AND m.name ILIKE i.pattern
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT 5
	),
	quiz_results AS (
		SELECT q.id, q.name, q.type, q.status, q.start_at, q.end_at, q.created_at,
		       level.id AS level_id, level.name AS level_name,
		       grade.id AS grade_id, grade.name AS grade_name
		FROM quizzes q
		CROSS JOIN input i
		JOIN master_data_items level ON level.id = q.education_level_id
		JOIN master_data_items grade ON grade.id = q.grade_id
		WHERE q.deleted_at IS NULL AND q.name ILIKE i.pattern
		ORDER BY q.created_at DESC, q.id DESC
		LIMIT 5
	)
	SELECT jsonb_build_object(
		'users', COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id', id, 'name', name, 'username', username, 'role', role, 'status', status
			) ORDER BY created_at DESC, id DESC) FROM user_results
		), '[]'::jsonb),
		'questions', COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id', id, 'code', code, 'preview', preview, 'type', type,
				'education_level', jsonb_build_object('id', level_id, 'name', level_name),
				'grade', jsonb_build_object('id', grade_id, 'name', grade_name)
			) ORDER BY created_at DESC, id DESC) FROM question_results
		), '[]'::jsonb),
		'learning_modules', COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id', id, 'name', name, 'status', status,
				'education_level', jsonb_build_object('id', level_id, 'name', level_name),
				'grade', jsonb_build_object('id', grade_id, 'name', grade_name)
			) ORDER BY created_at DESC, id DESC) FROM module_results
		), '[]'::jsonb),
		'quizzes', COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id', id, 'name', name, 'type', type, 'status', status,
				'start_at', start_at, 'end_at', end_at,
				'education_level', jsonb_build_object('id', level_id, 'name', level_name),
				'grade', jsonb_build_object('id', grade_id, 'name', grade_name)
			) ORDER BY created_at DESC, id DESC) FROM quiz_results
		), '[]'::jsonb)
	) AS results`

	var row struct {
		Results []byte `gorm:"column:results"`
	}
	pattern := "%" + searchLikeEscaper.Replace(query) + "%"
	if err := r.db.WithContext(ctx).Raw(searchQuery, pattern).Scan(&row).Error; err != nil {
		return dashboardmodel.SearchResults{}, err
	}

	var results dashboardmodel.SearchResults
	if err := json.Unmarshal(row.Results, &results); err != nil {
		return dashboardmodel.SearchResults{}, fmt.Errorf("decode Admin dashboard search results: %w", err)
	}
	return results, nil
}

func (r *DashboardRepository) Summary(ctx context.Context, periods dashboardmodel.SummaryPeriods) (dashboardmodel.Summary, error) {
	const query = `
	WITH bounds AS (
		SELECT ?::timestamptz AS month_start,
		       ?::timestamptz AS month_end,
		       ?::timestamptz AS previous_month_start,
		       ?::timestamptz AS week_start,
		       ?::timestamptz AS quiz_window_start,
		       ?::timestamptz AS today_start,
		       ?::timestamptz AS tomorrow_start
	),
	user_metrics AS (
		SELECT COUNT(*)::bigint AS total,
		       COUNT(*) FILTER (WHERE role = 'student')::bigint AS students,
		       COUNT(*) FILTER (WHERE role = 'tutor')::bigint AS tutors,
		       COUNT(*) FILTER (WHERE role = 'admin')::bigint AS admins,
		       COUNT(*) FILTER (WHERE created_at >= b.month_start AND created_at < b.month_end)::bigint AS new_this_month,
		       COUNT(*) FILTER (WHERE created_at >= b.previous_month_start AND created_at < b.month_start)::bigint AS new_last_month
		FROM users CROSS JOIN bounds b
		WHERE deleted_at IS NULL AND role IN ('student', 'tutor', 'admin')
	),
	question_levels AS (
		SELECT l.id, l.name, l.sort_order,
		       COUNT(q.id)::bigint AS total,
		       COUNT(q.id) FILTER (WHERE q.created_at >= b.week_start AND q.created_at < b.tomorrow_start)::bigint AS new_this_week
		FROM master_data_items l
		CROSS JOIN bounds b
		LEFT JOIN questions q ON q.education_level_id = l.id AND q.deleted_at IS NULL
		WHERE l.type = 'education_level' AND l.code IN ('SD', 'SMP') AND l.deleted_at IS NULL
		GROUP BY l.id, l.name, l.sort_order
	),
	question_metrics AS (
		SELECT (SELECT COUNT(*)::bigint FROM questions WHERE deleted_at IS NULL) AS total,
		       (SELECT COUNT(*)::bigint FROM questions q CROSS JOIN bounds b
		        WHERE q.deleted_at IS NULL AND q.created_at >= b.week_start AND q.created_at < b.tomorrow_start) AS new_this_week,
		       COALESCE((
		           SELECT jsonb_agg(jsonb_build_object(
		               'education_level', jsonb_build_object('id', id, 'name', name),
		               'total', total, 'new_this_week', new_this_week
		           ) ORDER BY sort_order, name)
		           FROM question_levels
		       ), '[]'::jsonb) AS by_level
	),
	module_levels AS (
		SELECT l.id, l.name, l.sort_order,
		       COUNT(m.id)::bigint AS total,
		       COUNT(m.id) FILTER (WHERE m.status = 'published')::bigint AS published,
		       COUNT(m.id) FILTER (
		           WHERE m.status = 'published'
		             AND COALESCE(m.published_at, m.created_at) >= b.week_start
		             AND COALESCE(m.published_at, m.created_at) < b.tomorrow_start
		       )::bigint AS published_this_week
		FROM master_data_items l
		CROSS JOIN bounds b
		LEFT JOIN learning_modules m ON m.education_level_id = l.id AND m.deleted_at IS NULL
		WHERE l.type = 'education_level' AND l.code IN ('SD', 'SMP') AND l.deleted_at IS NULL
		GROUP BY l.id, l.name, l.sort_order
	),
	module_metrics AS (
		SELECT (SELECT COUNT(*)::bigint FROM learning_modules WHERE deleted_at IS NULL) AS total,
		       (SELECT COUNT(*)::bigint FROM learning_modules WHERE deleted_at IS NULL AND status = 'published') AS published,
		       (SELECT COUNT(*)::bigint FROM learning_modules m CROSS JOIN bounds b
		        WHERE m.deleted_at IS NULL AND m.status = 'published'
		          AND COALESCE(m.published_at, m.created_at) >= b.week_start
		          AND COALESCE(m.published_at, m.created_at) < b.tomorrow_start) AS published_this_week,
		       COALESCE((
		           SELECT jsonb_agg(jsonb_build_object(
		               'education_level', jsonb_build_object('id', id, 'name', name),
		               'total', total, 'published', published, 'published_this_week', published_this_week
		           ) ORDER BY sort_order, name)
		           FROM module_levels
		       ), '[]'::jsonb) AS by_level
	),
	target_quizzes AS (
		SELECT q.id, q.grade_id
		FROM quizzes q CROSS JOIN bounds b
		WHERE q.type IN ('uji_kom', 'tryout')
		  AND q.status = 'published' AND q.deleted_at IS NULL
		  AND q.start_at >= b.quiz_window_start AND q.start_at < b.tomorrow_start
	),
	targets_by_grade AS (
		SELECT grade_id, COUNT(*)::bigint AS quiz_count
		FROM target_quizzes
		GROUP BY grade_id
	),
	students_by_grade AS (
		SELECT grade_id, COUNT(*)::bigint AS student_count
		FROM users
		WHERE role = 'student' AND status = 'active' AND deleted_at IS NULL
		GROUP BY grade_id
	),
	completion_metrics AS (
		SELECT COALESCE((
			SELECT SUM(t.quiz_count * s.student_count)::bigint
			FROM targets_by_grade t
			JOIN students_by_grade s ON s.grade_id = t.grade_id
		), 0)::bigint AS total,
		(SELECT COUNT(*)::bigint
		 FROM target_quizzes q
		 JOIN quiz_attempts a ON a.quiz_id = q.id AND a.submitted_at IS NOT NULL
		 JOIN users u ON u.id = a.student_id AND u.grade_id = q.grade_id
		              AND u.role = 'student' AND u.status = 'active' AND u.deleted_at IS NULL
		) AS completed
	),
	task_metrics AS (
		SELECT COUNT(*) FILTER (WHERE status = 'pending')::bigint AS pending_count
		FROM admin_tasks
	),
	task_items AS (
		SELECT COALESCE(jsonb_agg(jsonb_build_object(
			'id', id, 'text', text, 'status', status, 'created_at', created_at,
			'created_by', jsonb_build_object('id', created_by_id, 'name', created_by_name)
		) ORDER BY created_at DESC, id DESC), '[]'::jsonb) AS items
		FROM (
			SELECT t.id, t.task_text AS text, t.status, t.created_at,
			       u.id AS created_by_id, COALESCE(NULLIF(u.real_name, ''), u.username) AS created_by_name
			FROM admin_tasks t
			JOIN users u ON u.id = t.created_by
			WHERE t.status = 'pending'
			ORDER BY t.created_at DESC, t.id DESC
			LIMIT 5
		) recent
	),
	task_activity AS (
		SELECT COALESCE(jsonb_agg(jsonb_build_object(
			'id', id, 'text', text, 'status', status, 'updated_at', updated_at,
			'updated_by', jsonb_build_object('id', updated_by_id, 'name', updated_by_name)
		) ORDER BY updated_at DESC), '[]'::jsonb) AS recent_activity
		FROM (
			SELECT t.id, t.task_text AS text, t.status, t.status_updated_at AS updated_at,
			       u.id AS updated_by_id, COALESCE(NULLIF(u.real_name, ''), u.username) AS updated_by_name
			FROM admin_tasks t
			JOIN users u ON u.id = t.status_updated_by
			WHERE t.status_updated_at IS NOT NULL
			ORDER BY t.status_updated_at DESC, t.id DESC
			LIMIT 5
		) recent
	),
	uji_kom_today AS (
		SELECT COUNT(*)::bigint AS total
		FROM quizzes q CROSS JOIN bounds b
		WHERE q.type = 'uji_kom' AND q.status = 'published' AND q.deleted_at IS NULL
		  AND q.start_at < b.tomorrow_start AND q.end_at >= b.today_start
	)
	SELECT jsonb_build_object(
		'users', jsonb_build_object(
			'total', u.total,
			'by_role', jsonb_build_object('students', u.students, 'tutors', u.tutors, 'admins', u.admins),
			'new_this_month', u.new_this_month,
			'new_last_month', u.new_last_month,
			'month_growth_percent', CASE
				WHEN u.new_last_month = 0 THEN NULL
				ELSE ROUND(((u.new_this_month - u.new_last_month)::numeric * 100 / u.new_last_month), 2)::float8
			END
		),
		'question_bank', jsonb_build_object(
			'total', q.total, 'new_this_week', q.new_this_week, 'by_level', q.by_level
		),
		'learning_modules', jsonb_build_object(
			'total', m.total, 'published', m.published, 'published_this_week', m.published_this_week, 'by_level', m.by_level
		),
		'quiz_completion', jsonb_build_object(
			'completed', c.completed, 'total', c.total, 'not_completed', GREATEST(c.total - c.completed, 0),
			'percent', CASE WHEN c.total = 0 THEN NULL
			                ELSE ROUND((c.completed::numeric * 100 / c.total), 2)::float8 END
		),
		'uji_kom_today', (SELECT total FROM uji_kom_today),
		'tasks', jsonb_build_object(
			'pending_count', t.pending_count,
			'items', i.items,
			'recent_activity', a.recent_activity
		)
	) AS dashboard
	FROM user_metrics u
	CROSS JOIN question_metrics q
	CROSS JOIN module_metrics m
	CROSS JOIN completion_metrics c
	CROSS JOIN task_metrics t
	CROSS JOIN task_items i
	CROSS JOIN task_activity a`

	var row struct {
		Dashboard []byte `gorm:"column:dashboard"`
	}
	err := r.db.WithContext(ctx).Raw(query,
		periods.MonthStart, periods.MonthEnd, periods.PreviousMonthStart,
		periods.WeekStart, periods.QuizWindowStart, periods.TodayStart, periods.TomorrowStart,
	).Scan(&row).Error
	if err != nil {
		return dashboardmodel.Summary{}, err
	}

	var summary dashboardmodel.Summary
	if err := json.Unmarshal(row.Dashboard, &summary); err != nil {
		return dashboardmodel.Summary{}, fmt.Errorf("decode Admin dashboard summary: %w", err)
	}
	return summary, nil
}

func (r *DashboardRepository) DailyQuizTrendRows(
	ctx context.Context,
	from, lastBucket, to time.Time,
	period string,
) ([]dashboardmodel.TrendRow, error) {
	bucketUnit := "day"
	step := "1 day"
	if period == "12w" {
		bucketUnit = "week"
		step = "1 week"
	}

	query := fmt.Sprintf(`
	WITH buckets AS (
		SELECT generate_series(?::date, ?::date, INTERVAL '%s')::date AS bucket_start
	),
	levels AS (
		SELECT id, name, sort_order
		FROM master_data_items
		WHERE type = 'education_level' AND code IN ('SD', 'SMP') AND deleted_at IS NULL
	),
	averages AS (
		SELECT date_trunc('%s', d.submitted_at AT TIME ZONE 'Asia/Jakarta')::date AS bucket_start,
		       d.education_level_id,
		       COUNT(*)::bigint AS sessions,
		       ROUND(AVG(d.score)::numeric, 2)::float8 AS average
		FROM daily_quizzes d
		JOIN users u ON u.id = d.student_id AND u.deleted_at IS NULL
		WHERE d.submitted_at IS NOT NULL AND d.submitted_at >= ? AND d.submitted_at < ?
		GROUP BY 1, d.education_level_id
	)
	SELECT b.bucket_start,
	       l.id AS education_level_id, l.name AS education_level_name, l.sort_order,
	       COALESCE(a.sessions, 0)::bigint AS sessions, a.average
	FROM buckets b
	CROSS JOIN levels l
	LEFT JOIN averages a ON a.bucket_start = b.bucket_start AND a.education_level_id = l.id
	ORDER BY b.bucket_start, l.sort_order, l.name`, step, bucketUnit)

	var rows []struct {
		BucketStart        time.Time
		EducationLevelID   string
		EducationLevelName string
		SortOrder          int
		Sessions           int64
		Average            *float64
	}
	err := r.db.WithContext(ctx).Raw(query, from.Format(time.DateOnly), lastBucket.Format(time.DateOnly), from, to).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]dashboardmodel.TrendRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, dashboardmodel.TrendRow{
			BucketStart: row.BucketStart,
			EducationLevel: dashboardmodel.EducationLevel{
				ID: row.EducationLevelID, Name: row.EducationLevelName,
			},
			SortOrder: row.SortOrder,
			Sessions:  row.Sessions,
			Average:   row.Average,
		})
	}
	return result, nil
}
