DROP INDEX IF EXISTS quizzes_access_code_start_at_idx;
DROP INDEX IF EXISTS quizzes_access_code_unique;
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_access_code_required;
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_access_code_format;
ALTER TABLE quizzes
    DROP COLUMN IF EXISTS access_code_updated_at,
    DROP COLUMN IF EXISTS access_code,
    DROP COLUMN IF EXISTS start_at;
