-- ============================================================================
-- DATA DUMMY Daily Quiz bulan berjalan untuk siswa SD (Kelas 1-6). JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" SETELAH dummy_questions.sql (butuh soal Matematika SD).
-- Aman dijalankan berkali-kali: ID Daily Quiz dibuat dari hash (siswa + bulan + nomor), jadi yang
-- sudah ada dilewati. Dijalankan di bulan berikutnya → menambah data bulan itu.
--
-- Isi (per siswa aktif SD):
--   - 2-6 Daily Quiz, tersebar dari tanggal 1 sampai hari ini, sudah dibuka & di-submit.
--   - Level bergiliran (easy / medium / hard), 5 atau 10 soal dari bidang materi kelasnya.
--   - Benar/salah per soal dari hash: peluang benar easy 85%, medium 65%, hard 40%.
--   - Nilai, point (easy 10 / medium 15 / hard 20), waktu, dan leaderboard bulanan ikut diisi
--     seperti hasil submit asli.
--
-- Hasilnya bisa dilihat di GET /api/v1/daily-quiz-reports dan GET /api/v1/leaderboards/daily-quiz.
-- ============================================================================

BEGIN;

-- Rencana Daily Quiz: 1 baris per Daily Quiz (ID tetap dari hash supaya aman diulang).
CREATE TEMP TABLE tmp_daily_plan ON COMMIT DROP AS
WITH month_info AS (
    SELECT date_trunc('month', NOW() AT TIME ZONE 'Asia/Jakarta') AS month_start,
           (NOW() AT TIME ZONE 'Asia/Jakarta') AS now_wib,
           to_char(NOW() AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM') AS period
),
students AS (
    SELECT u.id AS student_id, u.education_level_id, u.grade_id
    FROM users u
    JOIN master_data_items g ON g.id = u.grade_id AND g.code IN ('GRADE_1', 'GRADE_2', 'GRADE_3', 'GRADE_4', 'GRADE_5', 'GRADE_6')
    WHERE u.role = 'student' AND u.status = 'active' AND u.deleted_at IS NULL
),
plan AS (
    SELECT s.*, m.period, n,
           md5('seed-daily-quiz|' || s.student_id || '|' || m.period || '|' || n)::uuid AS id,
           (ARRAY['easy', 'medium', 'hard'])[((ABS(hashtext(s.student_id::text)) + n) % 3) + 1] AS difficulty,
           CASE WHEN (ABS(hashtext(s.student_id::text || n::text)) % 2) = 0 THEN 5 ELSE 10 END AS total_questions,
           -- Waktu submit tersebar dari tanggal 1 sampai sekarang (jam WIB → timestamptz).
           (m.month_start + (m.now_wib - m.month_start) * ((ABS(hashtext(s.student_id::text || 'hari' || n)) % 1000) / 1000.0))
               AT TIME ZONE 'Asia/Jakarta' AS submitted_at,
           120 + ABS(hashtext(s.student_id::text || 'durasi' || n)) % 780 AS duration_seconds
    FROM students s
    CROSS JOIN month_info m
    CROSS JOIN LATERAL generate_series(1, 2 + ABS(hashtext(s.student_id::text || m.period)) % 5) AS n
)
SELECT p.*,
       -- Bidang materi kelas siswa (bergiliran per Daily Quiz).
       (SELECT f.id FROM master_data_items f
        WHERE f.type = 'subject_field' AND f.grade_id = p.grade_id AND f.deleted_at IS NULL
        ORDER BY f.sort_order, f.id
        OFFSET (p.n % GREATEST(1, (SELECT COUNT(*) FROM master_data_items f2
                                   WHERE f2.type = 'subject_field' AND f2.grade_id = p.grade_id AND f2.deleted_at IS NULL)))
        LIMIT 1) AS subject_field_id
FROM plan p
WHERE NOT EXISTS (SELECT 1 FROM daily_quizzes d WHERE d.id = p.id);

DELETE FROM tmp_daily_plan WHERE subject_field_id IS NULL;

-- 1. Daily Quiz (sudah dibuka; hasil diisi di langkah 3).
INSERT INTO daily_quizzes (id, student_id, education_level_id, grade_id, subject_id, subject_field_id,
                           difficulty, total_questions, opened_at, coin_charged, created_at)
SELECT p.id, p.student_id, p.education_level_id, p.grade_id, f.subject_id, p.subject_field_id,
       p.difficulty, p.total_questions,
       p.submitted_at - make_interval(secs => p.duration_seconds), 0,
       p.submitted_at - make_interval(secs => p.duration_seconds + 30)
FROM tmp_daily_plan p
JOIN master_data_items f ON f.id = p.subject_field_id;

-- 2. Soal: diambil dari bidang materi & level yang sama, urutan "acak" tetap (hash).
INSERT INTO daily_quiz_questions (daily_quiz_id, question_id, sort_order)
SELECT p.id, q.id, q.n
FROM tmp_daily_plan p
JOIN LATERAL (
    SELECT q.id, ROW_NUMBER() OVER (ORDER BY md5(p.id::text || q.id::text)) AS n
    FROM questions q
    WHERE q.subject_field_id = p.subject_field_id AND q.difficulty = p.difficulty
      AND q.status = 'active' AND q.deleted_at IS NULL AND q.used_for @> '["daily_quiz"]'
) q ON q.n <= p.total_questions;

-- Jumlah soal mengikuti soal yang benar-benar tersedia.
UPDATE daily_quizzes d SET total_questions = c.total
FROM (SELECT daily_quiz_id, COUNT(*)::int AS total FROM daily_quiz_questions
      WHERE daily_quiz_id IN (SELECT id FROM tmp_daily_plan) GROUP BY daily_quiz_id) c
WHERE d.id = c.daily_quiz_id AND d.total_questions <> c.total AND c.total IN (5, 10, 15);

-- 3. Jawaban & hasil submit.
WITH answers AS (
    SELECT dq.daily_quiz_id, dq.sort_order, dq.question_id, q.difficulty,
           (ABS(hashtext(dq.daily_quiz_id::text || dq.question_id::text)) % 100) <
               CASE q.difficulty WHEN 'easy' THEN 85 WHEN 'medium' THEN 65 ELSE 40 END AS is_correct
    FROM daily_quiz_questions dq
    JOIN questions q ON q.id = dq.question_id
    WHERE dq.daily_quiz_id IN (SELECT id FROM tmp_daily_plan)
),
graded AS (
    SELECT a.daily_quiz_id,
           COUNT(*)::int AS total,
           COUNT(*) FILTER (WHERE a.is_correct)::int AS correct_count,
           SUM(CASE WHEN a.is_correct THEN CASE a.difficulty WHEN 'easy' THEN 10 WHEN 'medium' THEN 15 ELSE 20 END ELSE 0 END)::int AS points,
           jsonb_agg(jsonb_build_object(
               'question_id', a.question_id,
               'answer', CASE WHEN a.is_correct THEN '"B"'::jsonb ELSE '"A"'::jsonb END,
               'is_correct', a.is_correct,
               'score', 0, 'points', 0, 'answered_at', NULL, 'time_spent_seconds', NULL) ORDER BY a.sort_order) AS answers
    FROM answers a
    GROUP BY a.daily_quiz_id
)
UPDATE daily_quizzes d
SET submitted_at     = p.submitted_at,
    score            = ROUND(g.correct_count * 100.0 / g.total, 2),
    correct_count    = g.correct_count,
    answers          = g.answers,
    duration_seconds = p.duration_seconds,
    points           = g.points
FROM graded g
JOIN tmp_daily_plan p ON p.id = g.daily_quiz_id
WHERE d.id = g.daily_quiz_id;

-- Daily Quiz yang tidak mendapat soal (bidang materi tanpa soal level itu) dibuang.
DELETE FROM daily_quizzes WHERE id IN (SELECT id FROM tmp_daily_plan) AND submitted_at IS NULL;

-- 4. Leaderboard bulanan: tambah ke baris siswa bulan ini (+ riwayat per Daily Quiz).
INSERT INTO leaderboard_entries (board, period, student_id, education_level_id, grade_id,
                                 points, duration_seconds, quiz_count, last_submitted_at)
SELECT 'daily_quiz', p.period, d.student_id, MIN(u.education_level_id::text)::uuid, MIN(u.grade_id::text)::uuid,
       SUM(d.points), SUM(d.duration_seconds), COUNT(*), MAX(d.submitted_at)
FROM daily_quizzes d
JOIN tmp_daily_plan p ON p.id = d.id
JOIN users u          ON u.id = d.student_id
GROUP BY p.period, d.student_id
ON CONFLICT ON CONSTRAINT leaderboard_entries_unique DO UPDATE SET
    points            = leaderboard_entries.points + EXCLUDED.points,
    duration_seconds  = leaderboard_entries.duration_seconds + EXCLUDED.duration_seconds,
    quiz_count        = leaderboard_entries.quiz_count + EXCLUDED.quiz_count,
    last_submitted_at = GREATEST(leaderboard_entries.last_submitted_at, EXCLUDED.last_submitted_at),
    updated_at        = NOW();

INSERT INTO leaderboard_point_logs (entry_id, student_id, source_type, source_id, points, duration_seconds, created_at)
SELECT e.id, d.student_id, 'daily_quiz', d.id, d.points, d.duration_seconds, d.submitted_at
FROM daily_quizzes d
JOIN tmp_daily_plan p      ON p.id = d.id
JOIN leaderboard_entries e ON e.board = 'daily_quiz' AND e.period = p.period AND e.student_id = d.student_id
ON CONFLICT ON CONSTRAINT leaderboard_point_logs_source_unique DO NOTHING;

COMMIT;
