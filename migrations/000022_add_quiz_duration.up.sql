-- ============================================================================
-- Waktu pengerjaan quiz. (BRD FR-QUIZ-041, FR-QUIZ-043)
--
--   duration_seconds = lama mengerjakan total: submit - mulai (Uji Kom/Tryout: started_at,
--                      Daily Quiz: opened_at). Dihitung server saat submit.
--   Waktu per soal (answered_at, time_spent_seconds dari FE) disimpan di kolom JSONB answers.
-- ============================================================================

ALTER TABLE quiz_attempts
    ADD COLUMN duration_seconds INT CHECK (duration_seconds >= 0);

ALTER TABLE daily_quizzes
    ADD COLUMN duration_seconds INT CHECK (duration_seconds >= 0);

-- Pengerjaan yang sudah di-submit sebelum kolom ini ada.
UPDATE quiz_attempts
SET duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM (submitted_at - started_at))::int)
WHERE submitted_at IS NOT NULL;

UPDATE daily_quizzes
SET duration_seconds = GREATEST(0, EXTRACT(EPOCH FROM (submitted_at - opened_at))::int)
WHERE submitted_at IS NOT NULL;

COMMENT ON COLUMN quiz_attempts.duration_seconds IS 'Lama mengerjakan (detik) = submitted_at - started_at';
COMMENT ON COLUMN daily_quizzes.duration_seconds IS 'Lama mengerjakan (detik) = submitted_at - opened_at';
COMMENT ON COLUMN quiz_attempts.answers IS 'Jawaban siswa urut nomor soal: [{question_id, answer, is_correct, points, answered_at, time_spent_seconds}]';
COMMENT ON COLUMN daily_quizzes.answers IS 'Jawaban siswa urut nomor soal: [{question_id, answer, is_correct, points, answered_at, time_spent_seconds}]';
