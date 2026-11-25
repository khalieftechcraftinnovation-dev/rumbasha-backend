-- ============================================================================
-- Tabel leaderboard tersendiri (lebih mudah dilacak & cepat dibaca). (BRD FR-RNK-014 s/d FR-RNK-018)
--
--   leaderboard_entries    = 1 baris per siswa per papan:
--                            Daily Quiz → per bulan (period 'YYYY-MM', WIB)
--                            Uji Kom / Tryout → per quiz (quiz_id)
--                            berisi total point, total waktu, jumlah quiz, snapshot jenjang & kelas.
--   leaderboard_point_logs = riwayat setiap point yang masuk: dari pengerjaan mana (source),
--                            berapa point & waktunya, kapan. 1 pengerjaan hanya bisa tercatat 1x.
--
-- Keduanya diisi di transaksi yang sama dengan submit jawaban.
-- Peringkat dibaca langsung dari leaderboard_entries (tanpa menjumlah ulang hasil quiz).
-- ============================================================================

CREATE TABLE leaderboard_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    board      VARCHAR(20) NOT NULL CHECK (board IN ('daily_quiz', 'uji_kom', 'tryout')),
    period     VARCHAR(7)  CHECK (period ~ '^\d{4}-\d{2}$'),     -- Daily Quiz: bulan WIB, contoh 2026-11
    quiz_id    UUID        REFERENCES quizzes (id),             -- Uji Kom / Tryout: quiz-nya
    student_id UUID        NOT NULL REFERENCES users (id),

    -- Snapshot jenjang & kelas akun siswa saat point terakhir masuk (filter leaderboard).
    -- Disimpan supaya peringkat bulan lalu tidak berubah kalau siswa naik kelas.
    education_level_id UUID REFERENCES master_data_items (id),
    grade_id           UUID REFERENCES master_data_items (id),

    points            INT         NOT NULL DEFAULT 0 CHECK (points >= 0),
    duration_seconds  INT         NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    quiz_count        INT         NOT NULL DEFAULT 0 CHECK (quiz_count >= 0),
    score             NUMERIC(5, 2) CHECK (score BETWEEN 0 AND 100),  -- Uji Kom / Tryout: nilai (info)
    last_submitted_at TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Daily Quiz memakai period (tanpa quiz_id); Uji Kom / Tryout memakai quiz_id (tanpa period).
    CONSTRAINT leaderboard_entries_scope CHECK (
        CASE WHEN board = 'daily_quiz' THEN period IS NOT NULL AND quiz_id IS NULL
             ELSE period IS NULL AND quiz_id IS NOT NULL END
    ),
    -- 1 siswa hanya 1 baris per papan (dipakai INSERT ... ON CONFLICT saat submit).
    CONSTRAINT leaderboard_entries_unique UNIQUE NULLS NOT DISTINCT (board, period, quiz_id, student_id)
);

-- Peringkat Daily Quiz 1 bulan (urutan = point DESC, waktu ASC), dengan / tanpa filter jenjang & kelas.
CREATE INDEX leaderboard_entries_daily_idx
    ON leaderboard_entries (period, points DESC, duration_seconds)
    WHERE board = 'daily_quiz';
CREATE INDEX leaderboard_entries_daily_grade_idx
    ON leaderboard_entries (period, grade_id)
    WHERE board = 'daily_quiz';
-- Peringkat 1 quiz Uji Kom / Tryout.
CREATE INDEX leaderboard_entries_quiz_idx
    ON leaderboard_entries (quiz_id, points DESC, duration_seconds)
    WHERE quiz_id IS NOT NULL;

COMMENT ON TABLE  leaderboard_entries        IS 'Total point siswa per papan peringkat (Daily Quiz per bulan, Uji Kom/Tryout per quiz)';
COMMENT ON COLUMN leaderboard_entries.period IS 'Daily Quiz: bulan WIB (YYYY-MM) dari waktu submit';
COMMENT ON COLUMN leaderboard_entries.points IS 'Total point: soal benar easy 10 / medium 15 / hard 20, salah 0';


CREATE TABLE leaderboard_point_logs (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id         UUID        NOT NULL REFERENCES leaderboard_entries (id) ON DELETE CASCADE,
    student_id       UUID        NOT NULL REFERENCES users (id),
    source_type      VARCHAR(20) NOT NULL CHECK (source_type IN ('daily_quiz', 'quiz_attempt')),
    source_id        UUID        NOT NULL,  -- daily_quizzes.id / quiz_attempts.id
    points           INT         NOT NULL CHECK (points >= 0),
    duration_seconds INT         NOT NULL CHECK (duration_seconds >= 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 1 pengerjaan hanya boleh menambah point 1x.
    CONSTRAINT leaderboard_point_logs_source_unique UNIQUE (source_type, source_id)
);

CREATE INDEX leaderboard_point_logs_entry_idx   ON leaderboard_point_logs (entry_id, created_at);
CREATE INDEX leaderboard_point_logs_student_idx ON leaderboard_point_logs (student_id, created_at DESC);

COMMENT ON TABLE leaderboard_point_logs IS 'Riwayat point leaderboard per pengerjaan quiz (untuk pelacakan)';


-- ----------------------------------------------------------------------------
-- Isi dari hasil quiz yang sudah di-submit sebelum tabel ini ada.
-- ----------------------------------------------------------------------------
INSERT INTO leaderboard_entries (board, period, student_id, education_level_id, grade_id,
                                 points, duration_seconds, quiz_count, last_submitted_at)
SELECT 'daily_quiz', to_char(d.submitted_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM'), d.student_id,
       u.education_level_id, u.grade_id,
       SUM(d.points), SUM(COALESCE(d.duration_seconds, 0)), COUNT(*), MAX(d.submitted_at)
FROM daily_quizzes d
JOIN users u ON u.id = d.student_id
WHERE d.submitted_at IS NOT NULL
GROUP BY 2, d.student_id, u.education_level_id, u.grade_id;

INSERT INTO leaderboard_entries (board, quiz_id, student_id, education_level_id, grade_id,
                                 points, duration_seconds, quiz_count, score, last_submitted_at)
SELECT z.type, a.quiz_id, a.student_id, u.education_level_id, u.grade_id,
       a.points, COALESCE(a.duration_seconds, 0), 1, a.score, a.submitted_at
FROM quiz_attempts a
JOIN quizzes z ON z.id = a.quiz_id
JOIN users u   ON u.id = a.student_id
WHERE a.submitted_at IS NOT NULL AND z.type IN ('uji_kom', 'tryout');

INSERT INTO leaderboard_point_logs (entry_id, student_id, source_type, source_id, points, duration_seconds, created_at)
SELECT e.id, d.student_id, 'daily_quiz', d.id, d.points, COALESCE(d.duration_seconds, 0), d.submitted_at
FROM daily_quizzes d
JOIN leaderboard_entries e ON e.board = 'daily_quiz' AND e.student_id = d.student_id
                          AND e.period = to_char(d.submitted_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM')
WHERE d.submitted_at IS NOT NULL;

INSERT INTO leaderboard_point_logs (entry_id, student_id, source_type, source_id, points, duration_seconds, created_at)
SELECT e.id, a.student_id, 'quiz_attempt', a.id, a.points, COALESCE(a.duration_seconds, 0), a.submitted_at
FROM quiz_attempts a
JOIN leaderboard_entries e ON e.quiz_id = a.quiz_id AND e.student_id = a.student_id
WHERE a.submitted_at IS NOT NULL;


-- Index leaderboard lama (menjumlah langsung dari hasil quiz) tidak dipakai lagi.
DROP INDEX IF EXISTS daily_quizzes_leaderboard_idx;
DROP INDEX IF EXISTS quiz_attempts_leaderboard_idx;
