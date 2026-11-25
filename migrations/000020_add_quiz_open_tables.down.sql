DROP TABLE IF EXISTS quiz_attempts;
DROP INDEX IF EXISTS daily_quizzes_student_opened_idx;
ALTER TABLE daily_quizzes
    DROP COLUMN IF EXISTS coin_charged,
    DROP COLUMN IF EXISTS opened_at;
