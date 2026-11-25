-- ============================================================================
-- Submit jawaban quiz. (BRD FR-QUIZ-041, FR-QUIZ-042)
--
-- Penilaian dilakukan di FE (FE sudah menerima kunci jawaban saat quiz dibuka); server
-- menyimpan jawaban + benar/salah per soal, lalu menghitung nilai:
--   nilai total 100, nilai per soal = 100 / total soal, nilai = jumlah benar x nilai per soal.
--
-- Jawaban disimpan sebagai 1 kolom JSONB per pengerjaan (bukan 1 baris per soal): submit = 1 UPDATE,
-- review = baca 1 baris. Isi:
--   [{"question_id": "...", "answer": <jawaban siswa>, "is_correct": true, "points": 20}, ...]
-- urut sesuai nomor soal.
-- ============================================================================

-- Uji Kom & Tryout: hasil per siswa per quiz.
ALTER TABLE quiz_attempts
    ADD COLUMN submitted_at  TIMESTAMPTZ,
    ADD COLUMN score         NUMERIC(5, 2) CHECK (score BETWEEN 0 AND 100),
    ADD COLUMN correct_count INT CHECK (correct_count >= 0),
    ADD COLUMN answers       JSONB,
    ADD CONSTRAINT quiz_attempts_submission_complete CHECK (
        (submitted_at IS NULL) = (score IS NULL) AND
        (submitted_at IS NULL) = (correct_count IS NULL) AND
        (submitted_at IS NULL) = (answers IS NULL)
    );

-- Daily Quiz: hasil langsung di baris Daily Quiz (1 Daily Quiz = 1 pengerjaan).
ALTER TABLE daily_quizzes
    ADD COLUMN submitted_at  TIMESTAMPTZ,
    ADD COLUMN score         NUMERIC(5, 2) CHECK (score BETWEEN 0 AND 100),
    ADD COLUMN correct_count INT CHECK (correct_count >= 0),
    ADD COLUMN answers       JSONB,
    ADD CONSTRAINT daily_quizzes_submission_complete CHECK (
        (submitted_at IS NULL) = (score IS NULL) AND
        (submitted_at IS NULL) = (correct_count IS NULL) AND
        (submitted_at IS NULL) = (answers IS NULL)
    ),
    ADD CONSTRAINT daily_quizzes_submit_after_open CHECK (submitted_at IS NULL OR opened_at IS NOT NULL);

COMMENT ON COLUMN quiz_attempts.answers IS 'Jawaban siswa urut nomor soal: [{question_id, answer, is_correct, points}]';
COMMENT ON COLUMN quiz_attempts.score   IS 'Nilai 0-100 = jumlah benar x (100 / total soal)';
COMMENT ON COLUMN daily_quizzes.answers IS 'Jawaban siswa urut nomor soal: [{question_id, answer, is_correct, points}]';
COMMENT ON COLUMN daily_quizzes.score   IS 'Nilai 0-100 = jumlah benar x (100 / total soal)';
