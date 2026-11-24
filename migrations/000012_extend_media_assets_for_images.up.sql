-- ============================================================================
-- Image Library: media_assets dilengkapi untuk API upload gambar. (BRD 3.17)
--
-- File gambar disimpan di disk: STORAGE_DIR/images/<category>/<tahun>/<bulan>/<uuid>.<ext>
--   storage_path = lokasi relatif di STORAGE_DIR (dipakai saat hapus file)
--   url          = link lengkap untuk FE, contoh http://localhost:8080/storage/images/question/2026/11/xxx.png
-- ============================================================================

ALTER TABLE media_assets
    ADD COLUMN category         VARCHAR(20)  NOT NULL DEFAULT 'question'
        CHECK (category IN ('question', 'banner', 'landing')),
    ADD COLUMN name             VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN url              TEXT         NOT NULL DEFAULT '',
    ADD COLUMN width            INT,
    ADD COLUMN height           INT,
    -- Hierarki mapel (opsional, hanya untuk kategori question).
    ADD COLUMN subject_id       UUID REFERENCES master_data_items (id),
    ADD COLUMN subject_field_id UUID REFERENCES master_data_items (id),
    ADD COLUMN chapter_id       UUID REFERENCES master_data_items (id),
    ADD COLUMN subchapter_id    UUID REFERENCES master_data_items (id),
    ADD COLUMN topic_id         UUID REFERENCES master_data_items (id);

-- DEFAULT di atas hanya untuk mengisi baris lama; data baru wajib diisi aplikasi.
ALTER TABLE media_assets
    ALTER COLUMN category DROP DEFAULT,
    ALTER COLUMN name     DROP DEFAULT,
    ALTER COLUMN url      DROP DEFAULT;

-- Hierarki hanya untuk gambar soal, dan diisi berurutan dari atas
-- (bidang materi butuh mapel, bab butuh bidang materi, dst.).
ALTER TABLE media_assets ADD CONSTRAINT media_assets_hierarchy_check CHECK (
    (category = 'question' OR (subject_id IS NULL AND subject_field_id IS NULL AND chapter_id IS NULL
                                AND subchapter_id IS NULL AND topic_id IS NULL))
    AND (subject_field_id IS NULL OR subject_id IS NOT NULL)
    AND (chapter_id       IS NULL OR subject_field_id IS NOT NULL)
    AND (subchapter_id    IS NULL OR chapter_id IS NOT NULL)
    AND (topic_id         IS NULL OR subchapter_id IS NOT NULL)
);

-- Daftar gambar per kategori (terbaru dulu) dan filter per hierarki mapel.
CREATE INDEX media_assets_category_created_idx ON media_assets (category, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX media_assets_hierarchy_idx ON media_assets (subject_id, subject_field_id, chapter_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN media_assets.category     IS 'question = gambar soal, banner, landing = landing page';
COMMENT ON COLUMN media_assets.name         IS 'Nama gambar (upload banyak sekaligus: nama + nomor urut, contoh "Segitiga 2")';
COMMENT ON COLUMN media_assets.url          IS 'Link gambar lengkap untuk FE';
COMMENT ON COLUMN media_assets.storage_path IS 'Lokasi file relatif di STORAGE_DIR, contoh images/question/2026/11/<uuid>.png';

-- Gambar dihapus permanen (hard delete). Soal AKTIF yang memakai gambar dicek aplikasi
-- (hapus ditolak). Soal yang sudah di-soft delete cukup dilepas: image_id-nya jadi NULL.
ALTER TABLE questions DROP CONSTRAINT questions_image_id_fkey;
ALTER TABLE questions ADD CONSTRAINT questions_image_id_fkey
    FOREIGN KEY (image_id) REFERENCES media_assets (id) ON DELETE SET NULL;
