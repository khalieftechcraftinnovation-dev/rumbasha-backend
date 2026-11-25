-- ============================================================================
-- Point quiz & leaderboard. (BRD FR-QUIZ-044, FR-LDB-*)
--
-- Nilai != point:
--   nilai (score)  = 0-100, per soal = 100 / total soal (tampil di hasil quiz).
--   point (points) = untuk leaderboard, per soal BENAR sesuai level soal:
--                    easy = 10, medium = 15, hard = 20; salah / tidak dijawab = 0.
-- Berlaku untuk semua quiz (Daily Quiz, Uji Kom, Tryout).
--
-- Isi kolom JSONB answers per soal menjadi:
--   {question_id, answer, is_correct, score, points, answered_at, time_spent_seconds}
--   (sebelumnya key "points" berisi nilai per soal → diganti nama menjadi "score").
-- ============================================================================

ALTER TABLE quiz_attempts ADD COLUMN points INT CHECK (points >= 0);
ALTER TABLE daily_quizzes ADD COLUMN points INT CHECK (points >= 0);

-- Rapikan jawaban yang sudah tersimpan: "points" (nilai) → "score", lalu isi "points" (point)
-- dari level soal, dan total point per pengerjaan.
CREATE FUNCTION pg_temp.answers_with_points(answers JSONB) RETURNS JSONB AS $$
    SELECT COALESCE(jsonb_agg(
        (e - 'points')
        || jsonb_build_object('score', COALESCE(e -> 'score', e -> 'points', '0'::jsonb))
        || jsonb_build_object('points',
               CASE WHEN (e ->> 'is_correct')::boolean THEN
                   CASE q.difficulty WHEN 'easy' THEN 10 WHEN 'medium' THEN 15 WHEN 'hard' THEN 20 ELSE 0 END
               ELSE 0 END)
        ORDER BY ordinality), '[]'::jsonb)
    FROM jsonb_array_elements(answers) WITH ORDINALITY AS t (e, ordinality)
    LEFT JOIN questions q ON q.id = (e ->> 'question_id')::uuid
$$ LANGUAGE sql;

UPDATE quiz_attempts SET answers = pg_temp.answers_with_points(answers) WHERE answers IS NOT NULL;
UPDATE daily_quizzes SET answers = pg_temp.answers_with_points(answers) WHERE answers IS NOT NULL;

UPDATE quiz_attempts
SET points = (SELECT COALESCE(SUM((e ->> 'points')::int), 0) FROM jsonb_array_elements(answers) e)
WHERE answers IS NOT NULL;
UPDATE daily_quizzes
SET points = (SELECT COALESCE(SUM((e ->> 'points')::int), 0) FROM jsonb_array_elements(answers) e)
WHERE answers IS NOT NULL;

-- Point wajib terisi bersama hasil submit.
ALTER TABLE quiz_attempts ADD CONSTRAINT quiz_attempts_points_complete CHECK ((submitted_at IS NULL) = (points IS NULL));
ALTER TABLE daily_quizzes ADD CONSTRAINT daily_quizzes_points_complete CHECK ((submitted_at IS NULL) = (points IS NULL));

-- Leaderboard Daily Quiz per bulan: Daily Quiz yang di-submit dalam rentang 1 bulan.
-- INCLUDE = kolom yang dijumlahkan ikut di index (index-only scan, tidak perlu baca tabel).
CREATE INDEX daily_quizzes_leaderboard_idx
    ON daily_quizzes (submitted_at)
    INCLUDE (student_id, points, duration_seconds)
    WHERE submitted_at IS NOT NULL;

-- Leaderboard Uji Kom / Tryout per quiz: pengerjaan yang sudah di-submit di 1 quiz.
CREATE INDEX quiz_attempts_leaderboard_idx
    ON quiz_attempts (quiz_id, points DESC, duration_seconds)
    WHERE submitted_at IS NOT NULL;

COMMENT ON COLUMN quiz_attempts.points IS 'Point leaderboard: soal benar easy 10 / medium 15 / hard 20, salah 0';
COMMENT ON COLUMN daily_quizzes.points IS 'Point leaderboard: soal benar easy 10 / medium 15 / hard 20, salah 0';
