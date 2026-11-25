-- ============================================================================
-- Index report Daily Quiz per siswa. (BRD FR-RNK-023 s/d FR-RNK-025)
--
--   daily_quizzes_report_idx          Daily Quiz yang sudah submit per rentang tanggal submit, berisi
--                                     semua kolom yang dijumlahkan (INCLUDE) → agregat per siswa
--                                     dibaca dari index saja (index-only scan).
--   daily_quizzes_student_report_idx  rincian 1 siswa (per mapel) dalam rentang tanggal.
-- ============================================================================

CREATE INDEX daily_quizzes_report_idx
    ON daily_quizzes (submitted_at)
    INCLUDE (student_id, subject_id, difficulty, total_questions, correct_count, score, duration_seconds, points)
    WHERE submitted_at IS NOT NULL;

CREATE INDEX daily_quizzes_student_report_idx
    ON daily_quizzes (student_id, submitted_at)
    INCLUDE (subject_id, difficulty, total_questions, correct_count, score, duration_seconds, points)
    WHERE submitted_at IS NOT NULL;
