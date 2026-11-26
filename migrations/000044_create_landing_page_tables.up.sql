-- ============================================================================
-- Landing Page (BRD 3.19, BR-025, BR-028).
--
-- Alur: admin mengubah section (isi, tampil/sembunyi, urutan) = DRAFT di landing_sections.
-- Setiap perubahan dicatat di landing_changes (panel "Belum dipublish"). Tombol Publish menyalin
-- section yang tampil ke landing_pages.published_content (1 kolom JSON = snapshot halaman publik),
-- menandai perubahan sudah dipublish, dan menulis landing_publish_logs.
--
-- Halaman publik cukup membaca 1 baris landing_pages (primary key), tanpa JOIN.
--
-- Kolom user (updated_by, changed_by, published_by) sengaja TANPA foreign key, sama seperti kolom
-- audit master_data_items: tabel ini berisi data awal dari migration yang tidak dikosongkan test.
-- ============================================================================

-- Halaman (tab) Landing Page. Data awal di bawah; halaman baru ditambah lewat migration.
CREATE TABLE landing_pages (
    code              VARCHAR(30)  PRIMARY KEY,  -- home, pricing, achievers
    name              VARCHAR(100) NOT NULL,     -- nama tab: Beranda, Paket & Biaya, Siswa Berprestasi
    sort_order        INT          NOT NULL DEFAULT 0,
    published_content JSONB,                     -- snapshot section yang tampil saat publish; NULL = belum pernah publish
    published_at      TIMESTAMPTZ,
    published_by      UUID,

    CONSTRAINT landing_pages_published_is_array
        CHECK (published_content IS NULL OR jsonb_typeof(published_content) = 'array')
);

-- Section setiap halaman (versi DRAFT yang diubah admin).
CREATE TABLE landing_sections (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    page_code   VARCHAR(30)  NOT NULL REFERENCES landing_pages (code),
    key         VARCHAR(30)  NOT NULL,                 -- kode tetap untuk FE, contoh hero, faq
    name        VARCHAR(100) NOT NULL,                 -- nama tampilan di panel admin
    content     JSONB        NOT NULL DEFAULT '{}',    -- isi section (bentuk bebas, ditentukan FE)
    is_visible  BOOLEAN      NOT NULL DEFAULT TRUE,    -- saklar "Tampil"
    sort_order  INT          NOT NULL,
    updated_by  UUID,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT landing_sections_page_key_unique UNIQUE (page_code, key),
    CONSTRAINT landing_sections_content_is_object CHECK (jsonb_typeof(content) = 'object')
);

-- Perubahan yang belum dipublish (panel "Belum dipublish"): published_at NULL.
CREATE TABLE landing_changes (
    id           BIGSERIAL    PRIMARY KEY,
    page_code    VARCHAR(30)  NOT NULL REFERENCES landing_pages (code),
    section_key  VARCHAR(30),                -- NULL untuk perubahan halaman (contoh urutan section)
    description  VARCHAR(200) NOT NULL,      -- contoh "Hero · isi diubah", "Urutan section diubah"
    changed_by   UUID,
    changed_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ                 -- diisi saat Publish
);

CREATE INDEX landing_changes_pending_idx
    ON landing_changes (page_code, changed_at)
    WHERE published_at IS NULL;

-- Riwayat publish.
CREATE TABLE landing_publish_logs (
    id           BIGSERIAL    PRIMARY KEY,
    page_code    VARCHAR(30)  NOT NULL REFERENCES landing_pages (code),
    published_by UUID,
    published_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    changes      JSONB        NOT NULL DEFAULT '[]'  -- daftar description yang ikut dipublish
);

CREATE INDEX landing_publish_logs_page_idx ON landing_publish_logs (page_code, published_at DESC);


-- ----------------------------------------------------------------------------
-- Data awal (FR-LDP-001, FR-LDP-002). Isi section kosong; diisi admin lewat panel.
-- Tab "Quiz Trial" tidak ada di sini: soalnya di tabel trial_quiz_questions.
-- ----------------------------------------------------------------------------
INSERT INTO landing_pages (code, name, sort_order) VALUES
    ('home',      'Beranda',           1),
    ('pricing',   'Paket & Biaya',     2),
    ('achievers', 'Siswa Berprestasi', 3);

INSERT INTO landing_sections (page_code, key, name, sort_order) VALUES
    ('home', 'hero',          'Hero',           1),
    ('home', 'video_profile', 'Video profil',   2),
    ('home', 'achievements',  'Bukti prestasi', 3),
    ('home', 'programs',      'Program',        4),
    ('home', 'facilities',    'Fasilitas',      5),
    ('home', 'testimonials',  'Testimoni',      6),
    ('home', 'promo',         'Promo',          7),
    ('home', 'faq',           'FAQ',            8),

    ('pricing',   'packages', 'Paket & Biaya',     1),
    ('achievers', 'students', 'Siswa Berprestasi', 1);

COMMENT ON TABLE  landing_pages                   IS 'Halaman (tab) Landing Page + snapshot yang sudah dipublish';
COMMENT ON COLUMN landing_pages.published_content IS 'Section yang tampil saat publish terakhir: [{"key","name","content"}] urut tampil';
COMMENT ON TABLE  landing_sections                IS 'Section Landing Page versi draft (diubah admin, tayang setelah Publish)';
COMMENT ON TABLE  landing_changes                 IS 'Perubahan Landing Page; published_at NULL = belum dipublish';
COMMENT ON TABLE  landing_publish_logs            IS 'Riwayat publish Landing Page';
