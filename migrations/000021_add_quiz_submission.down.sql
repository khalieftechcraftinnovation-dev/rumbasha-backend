ALTER TABLE daily_quizzes
    DROP CONSTRAINT IF EXISTS daily_quizzes_submit_after_open,
    DROP CONSTRAINT IF EXISTS daily_quizzes_submission_complete,
    DROP COLUMN IF EXISTS answers,
    DROP COLUMN IF EXISTS correct_count,
    DROP COLUMN IF EXISTS score,
    DROP COLUMN IF EXISTS submitted_at;

ALTER TABLE quiz_attempts
    DROP CONSTRAINT IF EXISTS quiz_attempts_submission_complete,
    DROP COLUMN IF EXISTS answers,
    DROP COLUMN IF EXISTS correct_count,
    DROP COLUMN IF EXISTS score,
    DROP COLUMN IF EXISTS submitted_at;
