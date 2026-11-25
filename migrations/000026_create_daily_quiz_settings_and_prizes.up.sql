-- ============================================================================
-- Pengaturan global Daily Quiz + hadiah juara bulanan. (BRD FR-RNK-019 s/d FR-RNK-022)
--
--   daily_quiz_settings          1 baris: point 1 soal benar per level & hadiah coin juara 1-3
--                                bulanan. Berlaku untuk SELURUH Daily Quiz, diubah admin lewat API.
--   daily_quiz_prize_periods     1 baris per bulan yang hadiahnya sudah dibagikan (+ snapshot hadiah)
--   daily_quiz_prize_awards      penerima hadiah: juara 1-3 PER KELAS per bulan
--
-- Hadiah dibagikan scheduler (job 23:59 WIB) pada hari terakhir bulan, untuk bulan itu dan bulan
-- sebelumnya yang terlewat. Coin dicatat di coin_transactions type 'reward'.
-- ============================================================================

CREATE TABLE daily_quiz_settings (
    id            SMALLINT    PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- selalu 1 baris
    points_easy   INT         NOT NULL DEFAULT 10 CHECK (points_easy   BETWEEN 0 AND 1000),
    points_medium INT         NOT NULL DEFAULT 15 CHECK (points_medium BETWEEN 0 AND 1000),
    points_hard   INT         NOT NULL DEFAULT 20 CHECK (points_hard   BETWEEN 0 AND 1000),
    prize_rank_1  INT         NOT NULL DEFAULT 0  CHECK (prize_rank_1  BETWEEN 0 AND 1000000),
    prize_rank_2  INT         NOT NULL DEFAULT 0  CHECK (prize_rank_2  BETWEEN 0 AND 1000000),
    prize_rank_3  INT         NOT NULL DEFAULT 0  CHECK (prize_rank_3  BETWEEN 0 AND 1000000),
    updated_by    UUID        REFERENCES users (id),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Nilai awal = aturan lama (10 / 15 / 20, tanpa hadiah).
INSERT INTO daily_quiz_settings (id) VALUES (1);

COMMENT ON TABLE  daily_quiz_settings              IS 'Pengaturan global Daily Quiz (selalu 1 baris, id = 1)';
COMMENT ON COLUMN daily_quiz_settings.points_easy  IS 'Point 1 soal level easy yang dijawab benar (semua Daily Quiz)';
COMMENT ON COLUMN daily_quiz_settings.prize_rank_1 IS 'Hadiah coin juara 1 bulanan per kelas, 0 = tanpa hadiah';


CREATE TABLE daily_quiz_prize_periods (
    period         VARCHAR(7)  PRIMARY KEY CHECK (period ~ '^[0-9]{4}-[0-9]{2}$'),  -- bulan WIB, contoh 2026-11
    prize_rank_1   INT         NOT NULL,  -- snapshot hadiah saat dibagikan
    prize_rank_2   INT         NOT NULL,
    prize_rank_3   INT         NOT NULL,
    distributed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE daily_quiz_prize_periods IS 'Bulan Daily Quiz yang hadiah juaranya sudah dibagikan (1 bulan hanya sekali)';


CREATE TABLE daily_quiz_prize_awards (
    id                  UUID        PRIMARY KEY,  -- diisi aplikasi (juga dipakai sebagai reference_id coin)
    period              VARCHAR(7)  NOT NULL REFERENCES daily_quiz_prize_periods (period),
    grade_id            UUID        NOT NULL REFERENCES master_data_items (id),  -- kelas (snapshot leaderboard)
    student_id          UUID        NOT NULL REFERENCES students (user_id),
    rank                INT         NOT NULL CHECK (rank BETWEEN 1 AND 3),
    points              INT         NOT NULL,
    duration_seconds    INT         NOT NULL,
    coin_amount         INT         NOT NULL CHECK (coin_amount > 0),
    coin_transaction_id UUID        NOT NULL REFERENCES coin_transactions (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT daily_quiz_prize_awards_unique UNIQUE (period, student_id)
);

CREATE INDEX daily_quiz_prize_awards_period_idx  ON daily_quiz_prize_awards (period, grade_id, rank);
CREATE INDEX daily_quiz_prize_awards_student_idx ON daily_quiz_prize_awards (student_id, created_at DESC);

COMMENT ON TABLE daily_quiz_prize_awards IS 'Penerima hadiah coin juara 1-3 Daily Quiz per kelas per bulan';
