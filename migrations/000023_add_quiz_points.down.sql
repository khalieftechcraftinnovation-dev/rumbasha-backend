DROP INDEX IF EXISTS quiz_attempts_leaderboard_idx;
DROP INDEX IF EXISTS daily_quizzes_leaderboard_idx;

ALTER TABLE daily_quizzes DROP CONSTRAINT IF EXISTS daily_quizzes_points_complete;
ALTER TABLE quiz_attempts DROP CONSTRAINT IF EXISTS quiz_attempts_points_complete;

-- Kembalikan bentuk jawaban lama: "points" berisi nilai per soal.
CREATE FUNCTION pg_temp.answers_without_points(answers JSONB) RETURNS JSONB AS $$
    SELECT COALESCE(jsonb_agg((e - 'points' - 'score') || jsonb_build_object('points', COALESCE(e -> 'score', '0'::jsonb)) ORDER BY ordinality), '[]'::jsonb)
    FROM jsonb_array_elements(answers) WITH ORDINALITY AS t (e, ordinality)
$$ LANGUAGE sql;

UPDATE quiz_attempts SET answers = pg_temp.answers_without_points(answers) WHERE answers IS NOT NULL;
UPDATE daily_quizzes SET answers = pg_temp.answers_without_points(answers) WHERE answers IS NOT NULL;

ALTER TABLE daily_quizzes DROP COLUMN IF EXISTS points;
ALTER TABLE quiz_attempts DROP COLUMN IF EXISTS points;
