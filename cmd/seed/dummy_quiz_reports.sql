-- ============================================================================
-- DATA DUMMY report quiz: 1 event Tryout yang SUDAH SELESAI beserta hasil pengerjaan siswa
-- Kelas 6. JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" SETELAH dummy_questions.sql (butuh soal Matematika Kelas 6).
-- Aman dijalankan berkali-kali: kalau quiz demo sudah ada, tidak ada yang ditambah.
--
-- Isi:
--   - Quiz "Tryout Demo Matematika Kelas 6": publish, 10 soal Kelas 6, event 7 hari lalu 08:00-10:00 WIB.
--   - Setiap siswa aktif Kelas 6 sudah submit. Benar/salah per soal dibuat dari hash (selalu sama
--     setiap seed): peluang benar easy 85%, medium 60%, hard 35%. Waktu pengerjaan 15-55 menit.
--   - Nilai (100 / 10 soal), point (easy 10, medium 15, hard 20), dan leaderboard Tryout-nya ikut
--     diisi, sama seperti hasil submit asli.
--
-- Hasilnya bisa dilihat di GET /api/v1/quiz-reports dan GET /api/v1/leaderboards/quizzes/:id.
-- ============================================================================

BEGIN;

-- Jadwal: 7 hari lalu 08:00-10:00 WIB. (jam WIB tanpa zona) AT TIME ZONE 'Asia/Jakarta' = timestamptz.
INSERT INTO quizzes (name, type, education_level_id, grade_id, subject_id,
                     start_at, end_at, duration_minutes, coin_cost, total_questions, status,
                     access_code, access_code_updated_at, created_by)
SELECT 'Tryout Demo Matematika Kelas 6', 'tryout', g.parent_id, g.id, s.id,
       (date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta') - INTERVAL '7 days' + INTERVAL '8 hours') AT TIME ZONE 'Asia/Jakarta',
       (date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta') - INTERVAL '7 days' + INTERVAL '10 hours') AT TIME ZONE 'Asia/Jakarta',
       60, 0, 10, 'published', 'DEMO26', NOW(),
       (SELECT id FROM users WHERE username = 'admin.sbytimur')
FROM master_data_items g, master_data_items s
WHERE g.type = 'grade' AND g.code = 'GRADE_6' AND s.type = 'subject' AND s.code = 'MTK'
  AND NOT EXISTS (SELECT 1 FROM quizzes WHERE name = 'Tryout Demo Matematika Kelas 6' AND deleted_at IS NULL);

-- 10 soal Kelas 6 bertanda Tryout (urut kode soal), 3-4 per level.
INSERT INTO quiz_questions (quiz_id, question_id, sort_order)
SELECT z.id, q.id, ROW_NUMBER() OVER (ORDER BY q.difficulty, q.code)
FROM quizzes z
JOIN LATERAL (
    SELECT q.id, q.code, q.difficulty,
           ROW_NUMBER() OVER (PARTITION BY q.difficulty ORDER BY q.code) AS n
    FROM questions q
    WHERE q.grade_id = z.grade_id AND q.deleted_at IS NULL AND q.status = 'active'
      AND q.used_for @> '["tryout"]'
) q ON (q.difficulty = 'easy' AND q.n <= 4) OR (q.difficulty = 'medium' AND q.n <= 3) OR (q.difficulty = 'hard' AND q.n <= 3)
WHERE z.name = 'Tryout Demo Matematika Kelas 6' AND z.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM quiz_questions qq WHERE qq.quiz_id = z.id);

UPDATE quizzes z SET total_questions = (SELECT COUNT(*) FROM quiz_questions qq WHERE qq.quiz_id = z.id)
WHERE z.name = 'Tryout Demo Matematika Kelas 6' AND z.deleted_at IS NULL;

-- Hasil pengerjaan setiap siswa aktif Kelas 6.
WITH quiz AS (
    SELECT id, start_at, total_questions FROM quizzes
    WHERE name = 'Tryout Demo Matematika Kelas 6' AND deleted_at IS NULL
),
students AS (
    SELECT u.id AS student_id
    FROM users u, quiz z
    WHERE u.role = 'student' AND u.status = 'active' AND u.deleted_at IS NULL
      AND u.grade_id = (SELECT grade_id FROM quizzes WHERE id = z.id)
      AND NOT EXISTS (SELECT 1 FROM quiz_attempts a WHERE a.quiz_id = z.id AND a.student_id = u.id)
),
per_question AS (
    SELECT s.student_id, qq.sort_order, q.id AS question_id, q.difficulty,
           (ABS(hashtext(s.student_id::text || q.id::text)) % 100) <
               CASE q.difficulty WHEN 'easy' THEN 85 WHEN 'medium' THEN 60 ELSE 35 END AS is_correct
    FROM students s
    CROSS JOIN quiz z
    JOIN quiz_questions qq ON qq.quiz_id = z.id
    JOIN questions q       ON q.id = qq.question_id
),
graded AS (
    SELECT p.student_id,
           COUNT(*) FILTER (WHERE p.is_correct)::int AS correct_count,
           SUM(CASE WHEN p.is_correct THEN CASE p.difficulty WHEN 'easy' THEN 10 WHEN 'medium' THEN 15 ELSE 20 END ELSE 0 END)::int AS points,
           jsonb_agg(jsonb_build_object(
               'question_id', p.question_id,
               'answer', CASE WHEN p.is_correct THEN '"B"'::jsonb ELSE '"A"'::jsonb END,
               'is_correct', p.is_correct,
               'score', CASE WHEN p.is_correct THEN ROUND(100.0 / (SELECT total_questions FROM quiz), 2) ELSE 0 END,
               'points', CASE WHEN p.is_correct THEN CASE p.difficulty WHEN 'easy' THEN 10 WHEN 'medium' THEN 15 ELSE 20 END ELSE 0 END,
               'answered_at', NULL,
               'time_spent_seconds', NULL) ORDER BY p.sort_order) AS answers
    FROM per_question p
    GROUP BY p.student_id
)
INSERT INTO quiz_attempts (quiz_id, student_id, started_at, submitted_at, score, correct_count, answers, duration_seconds, points)
SELECT z.id, g.student_id,
       z.start_at + make_interval(mins => ABS(hashtext(g.student_id::text)) % 30),
       z.start_at + make_interval(mins => ABS(hashtext(g.student_id::text)) % 30)
                  + make_interval(secs => 900 + ABS(hashtext(g.student_id::text || 'durasi')) % 2400),
       ROUND(g.correct_count * 100.0 / z.total_questions, 2), g.correct_count, g.answers,
       900 + ABS(hashtext(g.student_id::text || 'durasi')) % 2400, g.points
FROM graded g, quiz z;

-- Leaderboard Tryout untuk quiz demo (sama seperti yang dicatat saat submit).
WITH entries AS (
    INSERT INTO leaderboard_entries (board, quiz_id, student_id, education_level_id, grade_id,
                                     points, duration_seconds, quiz_count, score, last_submitted_at)
    SELECT 'tryout', a.quiz_id, a.student_id, u.education_level_id, u.grade_id,
           a.points, a.duration_seconds, 1, a.score, a.submitted_at
    FROM quiz_attempts a
    JOIN quizzes z ON z.id = a.quiz_id AND z.name = 'Tryout Demo Matematika Kelas 6' AND z.deleted_at IS NULL
    JOIN users u   ON u.id = a.student_id
    WHERE a.submitted_at IS NOT NULL
    ON CONFLICT ON CONSTRAINT leaderboard_entries_unique DO NOTHING
    RETURNING id, quiz_id, student_id, points, duration_seconds, last_submitted_at
)
INSERT INTO leaderboard_point_logs (entry_id, student_id, source_type, source_id, points, duration_seconds, created_at)
SELECT e.id, e.student_id, 'quiz_attempt', a.id, e.points, e.duration_seconds, e.last_submitted_at
FROM entries e
JOIN quiz_attempts a ON a.quiz_id = e.quiz_id AND a.student_id = e.student_id;

COMMIT;
