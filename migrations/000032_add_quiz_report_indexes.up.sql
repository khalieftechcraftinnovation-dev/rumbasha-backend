-- ============================================================================
-- Index report quiz Uji Kom & Tryout. (BRD FR-TRY-040 s/d FR-TRY-042)
--
--   quiz_attempts_report_idx  pengerjaan yang sudah submit per quiz + nilai/point/waktu
--                             (INCLUDE): jumlah peserta, rata-rata, tertinggi, terendah dihitung
--                             dari index saja (index-only scan), tanpa membaca tabel.
--   quizzes_report_idx        list report: quiz publish (belum dihapus) urut tanggal event terbaru.
-- ============================================================================

CREATE INDEX quiz_attempts_report_idx
    ON quiz_attempts (quiz_id)
    INCLUDE (score, points, duration_seconds, student_id)
    WHERE submitted_at IS NOT NULL;

CREATE INDEX quizzes_report_idx
    ON quizzes (start_at DESC, id DESC)
    WHERE deleted_at IS NULL AND status = 'published';
