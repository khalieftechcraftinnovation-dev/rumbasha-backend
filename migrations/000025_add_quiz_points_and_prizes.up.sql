-- ============================================================================
-- Point per level soal & hadiah coin juara 1-3 per quiz (Uji Kom & Tryout).
-- (BRD FR-TRY-036 s/d FR-TRY-039)
--
--   quizzes.points_easy/medium/hard   point 1 soal BENAR per level, diatur admin per quiz
--                                     (default 10 / 15 / 20 = aturan lama, jadi quiz lama tidak berubah)
--   quizzes.prize_rank_1/2/3          hadiah coin juara 1, 2, 3 leaderboard quiz (0 = tanpa hadiah)
--   quizzes.prizes_distributed_at     kapan hadiah dibagikan scheduler (NULL = belum)
--   quiz_prize_awards                 siapa saja yang menerima hadiah (riwayat, untuk pelacakan)
--   coin_transactions.type 'reward'   coin masuk dari hadiah
--
-- Hadiah dibagikan otomatis oleh scheduler setiap hari jam 23:59 WIB untuk quiz publish yang
-- event-nya sudah selesai (end_at <= saat itu) dan belum dibagikan.
-- Daily Quiz memakai pengaturan global (migration 000026, daily_quiz_settings).
-- ============================================================================

ALTER TABLE quizzes
    ADD COLUMN points_easy   INT NOT NULL DEFAULT 10 CHECK (points_easy   BETWEEN 0 AND 1000),
    ADD COLUMN points_medium INT NOT NULL DEFAULT 15 CHECK (points_medium BETWEEN 0 AND 1000),
    ADD COLUMN points_hard   INT NOT NULL DEFAULT 20 CHECK (points_hard   BETWEEN 0 AND 1000),
    ADD COLUMN prize_rank_1  INT NOT NULL DEFAULT 0  CHECK (prize_rank_1  BETWEEN 0 AND 1000000),
    ADD COLUMN prize_rank_2  INT NOT NULL DEFAULT 0  CHECK (prize_rank_2  BETWEEN 0 AND 1000000),
    ADD COLUMN prize_rank_3  INT NOT NULL DEFAULT 0  CHECK (prize_rank_3  BETWEEN 0 AND 1000000),
    ADD COLUMN prizes_distributed_at TIMESTAMPTZ;

COMMENT ON COLUMN quizzes.points_easy           IS 'Point 1 soal level easy yang dijawab benar (leaderboard)';
COMMENT ON COLUMN quizzes.points_medium         IS 'Point 1 soal level medium yang dijawab benar (leaderboard)';
COMMENT ON COLUMN quizzes.points_hard           IS 'Point 1 soal level hard yang dijawab benar (leaderboard)';
COMMENT ON COLUMN quizzes.prize_rank_1          IS 'Hadiah coin juara 1 leaderboard quiz, 0 = tanpa hadiah';
COMMENT ON COLUMN quizzes.prize_rank_2          IS 'Hadiah coin juara 2 leaderboard quiz, 0 = tanpa hadiah';
COMMENT ON COLUMN quizzes.prize_rank_3          IS 'Hadiah coin juara 3 leaderboard quiz, 0 = tanpa hadiah';
COMMENT ON COLUMN quizzes.prizes_distributed_at IS 'Kapan hadiah dibagikan scheduler (23:59 WIB setelah event selesai). NULL = belum';

-- Scheduler hadiah: quiz publish yang belum dibagikan hadiahnya, urut end_at.
-- Partial index → hanya berisi quiz yang masih menunggu (kecil, cepat).
CREATE INDEX quizzes_prize_pending_idx ON quizzes (end_at)
    WHERE prizes_distributed_at IS NULL AND deleted_at IS NULL AND status = 'published';


-- Coin hadiah dicatat sebagai type 'reward' (reference_type quiz_prize, reference_id = quiz).
ALTER TABLE coin_transactions DROP CONSTRAINT coin_transactions_type_check;
ALTER TABLE coin_transactions ADD CONSTRAINT coin_transactions_type_check
    CHECK (type IN ('plan_grant', 'purchase', 'usage', 'admin_adjustment', 'refund', 'reward'));
ALTER TABLE coin_transactions ADD CONSTRAINT coin_transactions_reward_has_reference
    CHECK (type <> 'reward' OR (reference_type IS NOT NULL AND reference_id IS NOT NULL));
COMMENT ON COLUMN coin_transactions.type IS 'plan_grant = bonus paket, purchase = beli, usage = dipakai, admin_adjustment = diubah admin, refund, reward = hadiah juara quiz';


-- Penerima hadiah per quiz. 1 siswa hanya menerima 1x per quiz.
-- Juara dengan point & waktu sama persis mendapat peringkat (dan hadiah) yang sama.
CREATE TABLE quiz_prize_awards (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id             UUID        NOT NULL REFERENCES quizzes (id),
    student_id          UUID        NOT NULL REFERENCES students (user_id),
    rank                INT         NOT NULL CHECK (rank BETWEEN 1 AND 3),
    points              INT         NOT NULL,  -- point leaderboard saat hadiah dibagikan
    duration_seconds    INT         NOT NULL,  -- total waktu saat hadiah dibagikan
    coin_amount         INT         NOT NULL CHECK (coin_amount > 0),
    coin_transaction_id UUID        NOT NULL REFERENCES coin_transactions (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT quiz_prize_awards_unique UNIQUE (quiz_id, student_id)
);

CREATE INDEX quiz_prize_awards_student_idx ON quiz_prize_awards (student_id, created_at DESC);

COMMENT ON TABLE quiz_prize_awards IS 'Penerima hadiah coin juara 1-3 leaderboard Uji Kom / Tryout (dibagikan scheduler)';
