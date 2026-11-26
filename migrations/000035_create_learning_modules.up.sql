-- ============================================================================
-- Modul pembelajaran: materi belajar dalam file PDF. (BRD FR-LMS-026 s/d FR-LMS-030)
--
--   learning_modules       1 baris per modul: nama, deskripsi, letak (mapel → topik), file PDF
--   learning_module_plans  paket akses (subscription_plans) yang membuka modul ini (minimal 1)
--
-- File PDF disimpan di disk: STORAGE_DIR/modules/<tahun>/<bulan>/<uuid>.pdf. TIDAK disajikan
-- publik (konten berbayar); diunduh lewat API yang wajib login.
--
-- Letak: mapel & bidang materi wajib (bidang materi sudah terikat jenjang & kelas, jadi jenjang
-- & kelas disalin otomatis untuk filter); bab, sub bab, topik opsional dan berurutan.
-- ============================================================================

CREATE TABLE learning_modules (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(200)  NOT NULL CHECK (btrim(name) <> ''),
    description TEXT,

    education_level_id UUID NOT NULL REFERENCES master_data_items (id),
    grade_id           UUID NOT NULL REFERENCES master_data_items (id),
    subject_id         UUID NOT NULL REFERENCES master_data_items (id),
    subject_field_id   UUID NOT NULL REFERENCES master_data_items (id),
    chapter_id         UUID REFERENCES master_data_items (id),
    subchapter_id      UUID REFERENCES master_data_items (id),
    topic_id           UUID REFERENCES master_data_items (id),

    -- File PDF.
    file_name    VARCHAR(255) NOT NULL,  -- nama file asli saat diunggah
    storage_path TEXT         NOT NULL,  -- lokasi relatif di STORAGE_DIR, contoh modules/2026/11/<uuid>.pdf
    mime_type    VARCHAR(100) NOT NULL,
    size_bytes   BIGINT       NOT NULL CHECK (size_bytes > 0),

    page_count INT         CHECK (page_count > 0),  -- jumlah halaman (diisi admin / dihitung dari PDF); NULL = tidak diketahui

    -- published = tampil di menu Modul siswa; draft = belum tampil ("Simpan sebagai draf").
    status     VARCHAR(10) NOT NULL DEFAULT 'published' CHECK (status IN ('draft', 'published')),
    created_by UUID        NOT NULL REFERENCES users (id),
    updated_by UUID        REFERENCES users (id),
    deleted_by UUID        REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Hierarki diisi berurutan dan saling cocok (pengaman terakhir; service sudah mengecek).
    CONSTRAINT learning_modules_subchapter_needs_chapter CHECK (subchapter_id IS NULL OR chapter_id IS NOT NULL),
    CONSTRAINT learning_modules_topic_needs_subchapter   CHECK (topic_id IS NULL OR subchapter_id IS NOT NULL),
    CONSTRAINT learning_modules_grade_level_fkey         FOREIGN KEY (grade_id, education_level_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT learning_modules_field_subject_fkey       FOREIGN KEY (subject_field_id, subject_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT learning_modules_chapter_field_fkey       FOREIGN KEY (chapter_id, subject_field_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT learning_modules_subchapter_chapter_fkey  FOREIGN KEY (subchapter_id, chapter_id)    REFERENCES master_data_items (id, parent_id),
    CONSTRAINT learning_modules_topic_subchapter_fkey    FOREIGN KEY (topic_id, subchapter_id)      REFERENCES master_data_items (id, parent_id)
);

-- List admin: terbaru dulu; filter per letak.
CREATE INDEX learning_modules_list_idx      ON learning_modules (created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX learning_modules_placement_idx ON learning_modules (grade_id, subject_id, subject_field_id) WHERE deleted_at IS NULL;
CREATE INDEX learning_modules_name_trgm_idx ON learning_modules USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;

COMMENT ON TABLE  learning_modules              IS 'Modul pembelajaran (materi PDF). File tidak publik, diunduh lewat API';
COMMENT ON COLUMN learning_modules.storage_path IS 'Lokasi file relatif di STORAGE_DIR, contoh modules/2026/11/<uuid>.pdf';


CREATE TABLE learning_module_plans (
    module_id UUID NOT NULL REFERENCES learning_modules (id) ON DELETE CASCADE,
    plan_id   UUID NOT NULL REFERENCES subscription_plans (id),
    PRIMARY KEY (module_id, plan_id)
);

-- Modul yang tersedia di 1 paket (nanti dipakai untuk daftar modul siswa).
CREATE INDEX learning_module_plans_plan_idx ON learning_module_plans (plan_id);

COMMENT ON TABLE learning_module_plans IS 'Paket akses yang membuka modul (modul tersedia di paket-paket ini)';
