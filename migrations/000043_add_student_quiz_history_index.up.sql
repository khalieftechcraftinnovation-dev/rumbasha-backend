-- ============================================================================
-- Halaman siswa "Sudah dikerjakan" Uji Kom & Tryout + statistik (BRD FR-TRY-017, FR-TRY-021):
-- pengerjaan 1 siswa yang sudah dikumpulkan, terbaru dulu. Index unik (quiz_id, student_id) tidak
-- bisa dipakai karena kolom pertamanya quiz_id. Riwayat Daily Quiz siswa memakai index
-- daily_quizzes_student_report_idx (migration 000034).
-- ============================================================================

CREATE INDEX quiz_attempts_student_history_idx
    ON quiz_attempts (student_id, submitted_at DESC)
    INCLUDE (quiz_id, score, points, duration_seconds, correct_count)
    WHERE submitted_at IS NOT NULL;
