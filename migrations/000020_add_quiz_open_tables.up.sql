-- ============================================================================
-- Buka quiz oleh siswa. (BRD FR-QUIZ-021, FR-TRY-034, FR-TRY-035)
--
--   Daily Quiz  : dicatat kapan pertama dibuka & coin yang dipotong (jatah gratis per hari).
--   Uji Kom/Tryout: quiz_attempts = siswa yang sudah membuka quiz dengan kode akses.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Daily Quiz: dibuka pertama kali kapan, dan berapa coin yang dipotong (0 = jatah gratis).
--    Jatah gratis dihitung dari jumlah Daily Quiz yang DIBUKA hari ini (WIB).
-- ----------------------------------------------------------------------------
ALTER TABLE daily_quizzes
    ADD COLUMN opened_at    TIMESTAMPTZ,
    ADD COLUMN coin_charged INT NOT NULL DEFAULT 0 CHECK (coin_charged >= 0);

-- Hitung Daily Quiz yang sudah dibuka siswa hari ini.
CREATE INDEX daily_quizzes_student_opened_idx
    ON daily_quizzes (student_id, opened_at)
    WHERE opened_at IS NOT NULL;

COMMENT ON COLUMN daily_quizzes.opened_at    IS 'Pertama kali dibuka siswa (NULL = baru di-generate). Membuka ulang tidak dihitung lagi';
COMMENT ON COLUMN daily_quizzes.coin_charged IS 'Coin yang dipotong saat dibuka (0 = masih jatah gratis hari itu)';


-- ----------------------------------------------------------------------------
-- 2. Uji Kom & Tryout: 1 baris per siswa per quiz, dibuat saat siswa pertama kali membuka
--    quiz dengan kode akses yang benar. Membuka ulang (contoh setelah refresh halaman, saat
--    kode akses sudah berganti) tidak perlu kode lagi. started_at = awal timer pengerjaan.
-- ----------------------------------------------------------------------------
CREATE TABLE quiz_attempts (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id    UUID        NOT NULL REFERENCES quizzes (id),
    student_id UUID        NOT NULL REFERENCES users (id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT quiz_attempts_quiz_student_unique UNIQUE (quiz_id, student_id)
);

COMMENT ON TABLE  quiz_attempts            IS 'Siswa yang sudah membuka quiz Uji Kom / Tryout (dengan kode akses)';
COMMENT ON COLUMN quiz_attempts.started_at IS 'Mulai mengerjakan; batas = min(started_at + durasi, end_at quiz)';
