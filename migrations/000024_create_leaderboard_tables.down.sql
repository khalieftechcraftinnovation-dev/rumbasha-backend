DROP TABLE IF EXISTS leaderboard_point_logs;
DROP TABLE IF EXISTS leaderboard_entries;

-- Kembalikan index leaderboard lama (migration 000023).
CREATE INDEX IF NOT EXISTS daily_quizzes_leaderboard_idx
    ON daily_quizzes (submitted_at)
    INCLUDE (student_id, points, duration_seconds)
    WHERE submitted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS quiz_attempts_leaderboard_idx
    ON quiz_attempts (quiz_id, points DESC, duration_seconds)
    WHERE submitted_at IS NOT NULL;
