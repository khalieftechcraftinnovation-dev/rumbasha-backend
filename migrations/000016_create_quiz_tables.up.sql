-- ============================================================================
-- Quiz (Uji Kom & Tryout) + soal yang dipilih. (BRD 3.6, FR-TRY-022 s/d FR-TRY-028)
--
--   quizzes         1 baris per quiz (info quiz)
--   quiz_questions  soal yang dipilih per quiz, berurutan (sort_order)
--
-- quiz_questions juga dipakai list Bank Soal untuk menandai soal yang sudah pernah
-- dipakai quiz (index quiz_questions_question_idx), jadi tidak perlu kolom tambahan
-- di tabel questions yang harus selalu disinkronkan.
-- ============================================================================

CREATE TABLE quizzes (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200)  NOT NULL CHECK (btrim(name) <> ''),
    -- Kode quiz_type di master data (disimpan sebagai teks, bukan relasi).
    -- API create saat ini hanya menerima uji_kom & tryout; daily_quiz & rumath disiapkan untuk nanti.
    type       VARCHAR(20)   NOT NULL CHECK (type IN ('daily_quiz', 'uji_kom', 'tryout', 'rumath')),

    -- Letak quiz: jenjang, kelas, mapel wajib; bidang materi s/d topik opsional (berurutan).
    education_level_id UUID  NOT NULL REFERENCES master_data_items (id),
    grade_id           UUID  NOT NULL REFERENCES master_data_items (id),
    subject_id         UUID  NOT NULL REFERENCES master_data_items (id),
    subject_field_id   UUID  REFERENCES master_data_items (id),
    chapter_id         UUID  REFERENCES master_data_items (id),
    subchapter_id      UUID  REFERENCES master_data_items (id),
    topic_id           UUID  REFERENCES master_data_items (id),

    duration_minutes INT     NOT NULL CHECK (duration_minutes BETWEEN 1 AND 600),
    coin_cost        INT     NOT NULL DEFAULT 0 CHECK (coin_cost >= 0),  -- 0 = gratis
    total_questions  INT     NOT NULL CHECK (total_questions > 0),       -- = jumlah baris quiz_questions
    status           VARCHAR(10) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),

    created_by UUID          NOT NULL REFERENCES users (id),
    updated_by UUID          REFERENCES users (id),
    deleted_by UUID          REFERENCES users (id),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Hierarki diisi berurutan: bab butuh bidang materi, sub bab butuh bab, topik butuh sub bab.
    CONSTRAINT quizzes_chapter_needs_field      CHECK (chapter_id IS NULL OR subject_field_id IS NOT NULL),
    CONSTRAINT quizzes_subchapter_needs_chapter CHECK (subchapter_id IS NULL OR chapter_id IS NOT NULL),
    CONSTRAINT quizzes_topic_needs_subchapter   CHECK (topic_id IS NULL OR subchapter_id IS NOT NULL),

    -- Pengaman terakhir urutan hierarki (service sudah mengecek dengan pesan yang jelas).
    CONSTRAINT quizzes_grade_level_fkey          FOREIGN KEY (grade_id, education_level_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT quizzes_field_subject_fkey        FOREIGN KEY (subject_field_id, subject_id)  REFERENCES master_data_items (id, parent_id),
    CONSTRAINT quizzes_chapter_field_fkey        FOREIGN KEY (chapter_id, subject_field_id)  REFERENCES master_data_items (id, parent_id),
    CONSTRAINT quizzes_subchapter_chapter_fkey   FOREIGN KEY (subchapter_id, chapter_id)     REFERENCES master_data_items (id, parent_id),
    CONSTRAINT quizzes_topic_subchapter_fkey     FOREIGN KEY (topic_id, subchapter_id)       REFERENCES master_data_items (id, parent_id)
);

-- List quiz admin: terbaru dulu, filter tipe (tab) / mapel / jenjang.
CREATE INDEX quizzes_type_created_at_idx ON quizzes (type, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX quizzes_created_at_idx      ON quizzes (created_at DESC, id DESC)       WHERE deleted_at IS NULL;
CREATE INDEX quizzes_subject_grade_idx   ON quizzes (subject_id, grade_id)            WHERE deleted_at IS NULL;

COMMENT ON TABLE  quizzes                  IS 'Quiz (Uji Kom, Tryout, ...): info quiz. Soalnya di quiz_questions';
COMMENT ON COLUMN quizzes.type             IS 'Kode quiz_type master data: daily_quiz, uji_kom, tryout, rumath';
COMMENT ON COLUMN quizzes.total_questions  IS 'Jumlah soal, selalu sama dengan jumlah baris quiz_questions (disimpan supaya list quiz tidak perlu COUNT)';
COMMENT ON COLUMN quizzes.coin_cost        IS 'Biaya coin per pengerjaan, 0 = gratis';


CREATE TABLE quiz_questions (
    quiz_id     UUID NOT NULL REFERENCES quizzes (id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions (id),
    sort_order  INT  NOT NULL CHECK (sort_order > 0),  -- nomor soal di quiz, mulai 1

    PRIMARY KEY (quiz_id, question_id),                -- 1 soal tidak boleh dipilih 2x di quiz yang sama
    CONSTRAINT quiz_questions_order_unique UNIQUE (quiz_id, sort_order)
);

-- Penanda di list Bank Soal: "soal ini sudah dipakai di quiz mana saja".
CREATE INDEX quiz_questions_question_idx ON quiz_questions (question_id);

COMMENT ON TABLE quiz_questions IS 'Soal yang dipilih per quiz, berurutan. Juga dipakai untuk menandai soal yang sudah pernah dipakai quiz';


-- ----------------------------------------------------------------------------
-- Filter list Bank Soal untuk memilih soal quiz: peruntukan (used_for) & status.
-- Contoh: used_for @> '["uji_kom"]'. jsonb_path_ops = index lebih kecil, khusus operator @>.
-- ----------------------------------------------------------------------------
CREATE INDEX questions_used_for_idx ON questions USING GIN (used_for jsonb_path_ops) WHERE deleted_at IS NULL;
