-- ============================================================================
-- Bank soal + tabel media minimal. (BRD 3.5, 3.17)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Image Library (versi minimal). Dipakai soal & opsi jawaban lewat image_id.
-- API upload gambar dibuat terpisah nanti; sekarang tabelnya dulu supaya
-- image_id di soal bisa dicek benar-benar ada.
-- ----------------------------------------------------------------------------
CREATE TABLE media_assets (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name    VARCHAR(255) NOT NULL,  -- nama file asli saat diunggah
    storage_path TEXT         NOT NULL,  -- lokasi file di storage (path / key)
    mime_type    VARCHAR(100) NOT NULL,  -- contoh: image/png
    size_bytes   BIGINT       NOT NULL CHECK (size_bytes > 0),
    uploaded_by  UUID         REFERENCES users (id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

COMMENT ON TABLE media_assets IS 'Image library: gambar untuk soal, materi, dan landing page';


-- ----------------------------------------------------------------------------
-- Bank soal.
--
-- Isi jawaban berbeda per tipe soal, jadi disimpan di kolom JSONB "answer"
-- (sudah divalidasi di service sebelum disimpan). Membaca 1 soal = 1 baris,
-- tanpa JOIN ke tabel opsi.
-- ----------------------------------------------------------------------------

-- Nomor urut untuk kode soal: Q-1, Q-2, ...
CREATE SEQUENCE questions_code_seq;

CREATE TABLE questions (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code          VARCHAR(20)  NOT NULL UNIQUE DEFAULT ('Q-' || nextval('questions_code_seq')),
    type          VARCHAR(20)  NOT NULL CHECK (type IN ('single_choice', 'multiple_choice', 'true_false', 'free_text')),

    -- Isi soal
    question_text TEXT         NOT NULL,  -- boleh berisi LaTeX, contoh: $\frac{1}{2}$
    formula       TEXT,                   -- rumus LaTeX terpisah (opsional)
    image_id      UUID         REFERENCES media_assets (id),
    table_data    JSONB,                  -- {"headers": [...], "rows": [[...]]} (opsional)
    explanation   TEXT,                   -- pembahasan (opsional)
    display_order JSONB        NOT NULL,  -- urutan tampil, contoh ["question", "table", "image"]

    -- Letak soal (placement). Bab wajib, sub bab & topik opsional.
    education_level_id UUID    NOT NULL REFERENCES master_data_items (id),
    grade_id           UUID    NOT NULL REFERENCES master_data_items (id),
    subject_id         UUID    NOT NULL REFERENCES master_data_items (id),
    subject_field_id   UUID    NOT NULL REFERENCES master_data_items (id),
    chapter_id         UUID    NOT NULL REFERENCES master_data_items (id),
    subchapter_id      UUID    REFERENCES master_data_items (id),
    topic_id           UUID    REFERENCES master_data_items (id),

    difficulty    VARCHAR(10)  NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    used_for      JSONB        NOT NULL,  -- contoh ["daily_quiz", "uji_kom"]
    status        VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    answer        JSONB        NOT NULL,  -- isi berbeda per type, lihat service/question

    created_by    UUID         NOT NULL REFERENCES users (id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT questions_answer_is_object      CHECK (jsonb_typeof(answer) = 'object'),
    CONSTRAINT questions_used_for_not_empty    CHECK (jsonb_typeof(used_for) = 'array' AND jsonb_array_length(used_for) > 0),
    CONSTRAINT questions_display_order_array   CHECK (jsonb_typeof(display_order) = 'array'),
    CONSTRAINT questions_topic_needs_subchapter CHECK (topic_id IS NULL OR subchapter_id IS NOT NULL),

    -- Pengaman terakhir urutan hierarki (service sudah mengecek lebih dulu dengan pesan yang jelas).
    -- Pasangan (anak, induk) harus sama dengan (id, parent_id) di master_data_items.
    -- Kolom yang NULL (sub bab / topik kosong) otomatis tidak dicek.
    CONSTRAINT questions_grade_level_fkey      FOREIGN KEY (grade_id, education_level_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT questions_field_subject_fkey    FOREIGN KEY (subject_field_id, subject_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT questions_chapter_field_fkey    FOREIGN KEY (chapter_id, subject_field_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT questions_subchapter_chapter_fkey FOREIGN KEY (subchapter_id, chapter_id)  REFERENCES master_data_items (id, parent_id),
    CONSTRAINT questions_topic_subchapter_fkey FOREIGN KEY (topic_id, subchapter_id)      REFERENCES master_data_items (id, parent_id)
);

-- Menyusun Daily Quiz: soal aktif per bab & tingkat kesulitan.
CREATE INDEX questions_chapter_difficulty_idx
    ON questions (chapter_id, difficulty)
    WHERE deleted_at IS NULL AND status = 'active';

-- Filter Bank Soal: mapel, jenjang, kelas.
CREATE INDEX questions_subject_level_grade_idx
    ON questions (subject_id, education_level_id, grade_id)
    WHERE deleted_at IS NULL;

-- Daftar Bank Soal urut terbaru.
CREATE INDEX questions_created_at_idx
    ON questions (created_at DESC)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE  questions            IS 'Bank soal (single_choice, multiple_choice, true_false, free_text)';
COMMENT ON COLUMN questions.code       IS 'Kode soal otomatis, contoh Q-1256';
COMMENT ON COLUMN questions.answer     IS 'Opsi & kunci jawaban, bentuknya tergantung type (divalidasi di service)';
COMMENT ON COLUMN questions.used_for   IS 'Peruntukan: daily_quiz, uji_kom, tryout, rumath';
COMMENT ON COLUMN questions.display_order IS 'Urutan tampil bagian soal: question, formula, image, table';
