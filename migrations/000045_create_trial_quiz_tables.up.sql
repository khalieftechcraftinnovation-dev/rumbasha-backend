-- ============================================================================
-- Quiz Trial publik (BRD 1.3, FR-AUTH-012, FR-LDP-001 tab Quiz Trial).
--
-- Tabel soal SENDIRI (bukan tabel questions), supaya API publik ringan: soal trial sedikit
-- (maks 10 per golongan) dan tidak bercampur dengan bank soal. Struktur isi soal SAMA PERSIS
-- dengan questions (type, answer JSONB, tabel, gambar, display_order), divalidasi dengan aturan
-- yang sama di service.
--
-- Golongan (grade_group):
--   pra_sd = Pra SD (Nol Kecil, Nol Besar)
--   sd_1_3 = SD Kelas 1-3
--   sd_4_6 = SD Kelas 4-6
--   smp    = SMP (Kelas 7-9)
--   sma    = SMA (Kelas 10-12)
-- ============================================================================

-- Nomor urut kode soal trial: TQ-1, TQ-2, ...
CREATE SEQUENCE trial_quiz_questions_code_seq;

CREATE TABLE trial_quiz_questions (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code          VARCHAR(20)  NOT NULL UNIQUE DEFAULT ('TQ-' || nextval('trial_quiz_questions_code_seq')),
    grade_group   VARCHAR(10)  NOT NULL CHECK (grade_group IN ('pra_sd', 'sd_1_3', 'sd_4_6', 'smp', 'sma')),
    subject_id    UUID         NOT NULL REFERENCES master_data_items (id),  -- mapel: Matematika, IPA, IPS, B. Inggris, ...
    sort_order    INT          NOT NULL DEFAULT 0,                          -- urutan soal di quiz (kecil dulu)

    -- Isi soal (sama dengan tabel questions)
    type          VARCHAR(20)  NOT NULL CHECK (type IN ('single_choice', 'multiple_choice', 'true_false', 'free_text')),
    question_text TEXT         NOT NULL,
    formula       TEXT,
    image_id      UUID         REFERENCES media_assets (id) ON DELETE SET NULL,
    image_url     TEXT,
    table_data    JSONB,
    explanation   TEXT         NOT NULL,
    display_order JSONB        NOT NULL,
    difficulty    VARCHAR(10)  NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    answer        JSONB        NOT NULL,

    created_by    UUID         NOT NULL REFERENCES users (id),
    updated_by    UUID         REFERENCES users (id),
    deleted_by    UUID         REFERENCES users (id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT trial_quiz_questions_answer_is_object    CHECK (jsonb_typeof(answer) = 'object'),
    CONSTRAINT trial_quiz_questions_display_order_array CHECK (jsonb_typeof(display_order) = 'array'),
    CONSTRAINT trial_quiz_questions_one_image_source    CHECK (image_id IS NULL OR image_url IS NULL)
);

-- API publik & list admin: soal 1 golongan, urut.
CREATE INDEX trial_quiz_questions_group_idx
    ON trial_quiz_questions (grade_group, sort_order, created_at)
    WHERE deleted_at IS NULL;


-- ----------------------------------------------------------------------------
-- Hasil pengerjaan trial (tanpa login). Data peserta wajib: nama, jenjang & kelas, no WhatsApp.
-- Benar/salah per soal dinilai FE; nilai dihitung BE dari jumlah benar (total 100).
-- ----------------------------------------------------------------------------
CREATE TABLE trial_quiz_attempts (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(100)  NOT NULL,
    whatsapp           VARCHAR(20)   NOT NULL,  -- disimpan rapi: 62xxxxxxxxxx
    education_level_id UUID          NOT NULL REFERENCES master_data_items (id),
    grade_id           UUID          NOT NULL REFERENCES master_data_items (id),
    grade_group        VARCHAR(10)   NOT NULL CHECK (grade_group IN ('pra_sd', 'sd_1_3', 'sd_4_6', 'smp', 'sma')),
    score              NUMERIC(5, 2) NOT NULL CHECK (score BETWEEN 0 AND 100),
    correct_count      INT           NOT NULL CHECK (correct_count >= 0),
    total_questions    INT           NOT NULL CHECK (total_questions > 0),
    duration_seconds   INT           CHECK (duration_seconds >= 0),  -- lama mengerjakan (dari FE), boleh kosong
    answers            JSONB         NOT NULL,  -- [{"question_id","answer","is_correct","score",...}] urut soal
    submitted_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT trial_quiz_attempts_answers_array CHECK (jsonb_typeof(answers) = 'array')
);

-- Report admin: urut terbaru, filter periode & golongan.
CREATE INDEX trial_quiz_attempts_submitted_idx ON trial_quiz_attempts (submitted_at DESC);
CREATE INDEX trial_quiz_attempts_group_idx     ON trial_quiz_attempts (grade_group, submitted_at DESC);

COMMENT ON TABLE  trial_quiz_questions             IS 'Soal Quiz Trial publik, maks 10 per golongan (struktur isi sama dengan questions)';
COMMENT ON COLUMN trial_quiz_questions.grade_group IS 'pra_sd, sd_1_3, sd_4_6, smp, sma';
COMMENT ON TABLE  trial_quiz_attempts              IS 'Hasil Quiz Trial publik + data peserta (nama, jenjang, kelas, WhatsApp)';
