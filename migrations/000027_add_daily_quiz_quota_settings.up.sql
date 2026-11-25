-- ============================================================================
-- Jatah gratis & biaya coin Daily Quiz pindah dari .env (DAILY_QUIZ_FREE_QUOTA,
-- DAILY_QUIZ_COIN_COST) ke pengaturan global daily_quiz_settings, supaya bisa diubah admin
-- lewat API tanpa restart server. (BRD FR-TOK-008, FR-TOK-009, FR-RNK-019)
--
-- Nilai awal = default .env sebelumnya: 2 Daily Quiz gratis per hari, lalu 10 coin per Daily Quiz.
-- ============================================================================

ALTER TABLE daily_quiz_settings
    ADD COLUMN free_per_day INT NOT NULL DEFAULT 2  CHECK (free_per_day BETWEEN 0 AND 100),
    ADD COLUMN coin_cost    INT NOT NULL DEFAULT 10 CHECK (coin_cost    BETWEEN 0 AND 1000000);

COMMENT ON COLUMN daily_quiz_settings.free_per_day IS 'Jumlah Daily Quiz gratis per siswa per hari (WIB)';
COMMENT ON COLUMN daily_quiz_settings.coin_cost    IS 'Biaya coin per Daily Quiz setelah jatah gratis hari itu habis (0 = selalu gratis)';
