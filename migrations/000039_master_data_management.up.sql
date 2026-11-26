-- ============================================================================
-- CRUD Master Data & Cabang oleh admin (BRD 3.16, FR-MST-001 s/d 008, FR-LMS-001 s/d 007).
--
--   master_data_items  + type 'school' (sekolah, tanpa induk)
--                      + is_system: data bawaan aplikasi (jenjang, kelas bawaan, tipe soal, tipe quiz,
--                        level) yang dipakai program lewat code-nya → tidak bisa dihapus, code / induk /
--                        status tidak bisa diubah (hanya nama & urutan, FR-MST-009)
--                      + kolom audit created_by / updated_by / deleted_by
--   branches           + kolom audit created_by / updated_by / deleted_by
-- ============================================================================

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN (
        'education_level',  -- jenjang          (tanpa induk)
        'grade',            -- kelas            (induk: education_level)
        'subject',          -- mata pelajaran   (tanpa induk, sama untuk semua jenjang)
        'subject_field',    -- bidang materi    (induk: subject)
        'chapter',          -- bab              (induk: subject_field)
        'subchapter',       -- sub bab          (induk: chapter)
        'topic',            -- topik            (induk: subchapter)
        'question_type',    -- tipe soal        (tanpa induk, code = questions.type)
        'quiz_type',        -- tipe quiz        (tanpa induk, code = isi questions.used_for)
        'difficulty',       -- level            (tanpa induk, code = questions.difficulty)
        'school'            -- sekolah          (tanpa induk; asal sekolah & filter peringkat)
    ));

ALTER TABLE master_data_items
    ADD COLUMN is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID;

-- Kolom audit sengaja TANPA foreign key ke users: master_data_items berisi data awal yang tidak
-- dikosongkan saat test, sedangkan PostgreSQL menolak TRUNCATE users selama ada foreign key dari
-- tabel yang tidak ikut dikosongkan. Isi kolom tetap id user (diisi API).

-- Data bawaan migration (000002 jenjang & kelas, 000015 tipe soal / tipe quiz / level).
UPDATE master_data_items SET is_system = TRUE
WHERE type IN ('education_level', 'grade', 'question_type', 'quiz_type', 'difficulty');

-- Daftar Master Data: cari nama (sebagian kata, tanpa peduli huruf besar/kecil).
CREATE INDEX master_data_items_name_trgm_idx
    ON master_data_items USING gin (name gin_trgm_ops)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN master_data_items.is_system  IS 'TRUE = data bawaan aplikasi: tidak bisa dihapus; hanya nama & urutan yang bisa diubah';
COMMENT ON COLUMN master_data_items.created_by IS 'Admin yang membuat (NULL = data awal / seed)';
COMMENT ON COLUMN master_data_items.updated_by IS 'Admin yang terakhir mengubah';
COMMENT ON COLUMN master_data_items.deleted_by IS 'Admin yang menghapus (soft delete)';

ALTER TABLE branches
    ADD COLUMN created_by UUID REFERENCES users (id),
    ADD COLUMN updated_by UUID REFERENCES users (id),
    ADD COLUMN deleted_by UUID REFERENCES users (id);

COMMENT ON COLUMN branches.created_by IS 'Admin yang membuat cabang (NULL = data awal / seed)';
COMMENT ON COLUMN branches.updated_by IS 'Admin yang terakhir mengubah';
COMMENT ON COLUMN branches.deleted_by IS 'Admin yang menghapus cabang (soft delete)';
