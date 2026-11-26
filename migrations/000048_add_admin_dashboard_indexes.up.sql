ALTER TABLE learning_modules
    ADD COLUMN published_at TIMESTAMPTZ;

COMMENT ON COLUMN learning_modules.published_at IS 'Last transition from draft to published; NULL for legacy rows, which use created_at for dashboard estimates';

CREATE INDEX users_dashboard_metrics_idx
    ON users (role, created_at)
    WHERE deleted_at IS NULL AND role IN ('student', 'tutor', 'admin');

CREATE INDEX users_active_student_grade_idx
    ON users (grade_id)
    WHERE role = 'student' AND status = 'active' AND deleted_at IS NULL;

CREATE INDEX questions_dashboard_metrics_idx
    ON questions (education_level_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX learning_modules_dashboard_metrics_idx
    ON learning_modules (education_level_id, created_at, published_at)
    INCLUDE (status)
    WHERE deleted_at IS NULL;

CREATE INDEX learning_modules_published_dashboard_idx
    ON learning_modules ((COALESCE(published_at, created_at)))
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX quizzes_dashboard_schedule_idx
    ON quizzes (type, start_at, grade_id)
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX quizzes_name_trgm_idx
    ON quizzes USING gin (name gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX daily_quizzes_dashboard_trend_idx
    ON daily_quizzes (submitted_at, education_level_id)
    INCLUDE (score, student_id)
    WHERE submitted_at IS NOT NULL;

CREATE INDEX admin_tasks_dashboard_pending_idx
    ON admin_tasks (created_at DESC, id DESC)
    WHERE status = 'pending';

CREATE INDEX admin_tasks_dashboard_activity_idx
    ON admin_tasks (status_updated_at DESC, id DESC)
    WHERE status_updated_at IS NOT NULL;
