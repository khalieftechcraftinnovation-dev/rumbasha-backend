-- ============================================================================
-- Hierarki mata pelajaran di master_data_items. (BRD 3.4.1, 3.16)
--
--   subject (Matematika)                         ← mapel SAMA untuk semua jenjang, tanpa induk
--     └─ subject_field  Bidang Materi  + jenjang & kelas, contoh: Bilangan (SD, Kelas 4)
--          └─ chapter      Bab          contoh: KPK dan FPB
--               └─ subchapter  Sub Bab  contoh: Menentukan KPK dan FPB
--                    └─ topic   Topik   contoh: Menentukan KPK
--
-- Mapel hanya satu (Matematika), dipakai SD, SMP, dan SMA.
-- Mulai Bidang Materi ke bawah, setiap item terhubung ke jenjang & kelas.
-- Contoh: "Bilangan" Matematika Kelas 1 dan "Bilangan" Matematika Kelas 4 adalah 2 item berbeda.
--
-- Data mapel diisi admin lewat menu Master Data (data contoh: make seed),
-- jadi migration ini hanya menambah type dan kolomnya.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Type baru
-- ----------------------------------------------------------------------------
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;

ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN (
        'education_level',  -- jenjang          (tanpa induk)
        'grade',            -- kelas            (induk: education_level)
        'subject',          -- mata pelajaran   (tanpa induk, sama untuk semua jenjang)
        'subject_field',    -- bidang materi    (induk: subject)
        'chapter',          -- bab              (induk: subject_field)
        'subchapter',       -- sub bab          (induk: chapter)
        'topic'             -- topik            (induk: subchapter)
    ));

-- Jenjang dan mapel tidak punya induk, type lain wajib punya induk.
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_parent_check
    CHECK ((type IN ('education_level', 'subject')) = (parent_id IS NULL));


-- ----------------------------------------------------------------------------
-- 2. Kolom bantu untuk Bidang Materi ke bawah
--
-- Kolom ini disalin dari induknya saat data dibuat (diisi oleh service/seed),
-- supaya query daftar bab/topik cukup 1 tabel tanpa JOIN berlapis. Contoh:
--   SELECT * FROM master_data_items
--   WHERE type = 'chapter' AND subject_id = ? AND grade_id = ?
-- ----------------------------------------------------------------------------
ALTER TABLE master_data_items
    ADD COLUMN subject_id         UUID REFERENCES master_data_items (id),
    ADD COLUMN education_level_id UUID REFERENCES master_data_items (id),
    ADD COLUMN grade_id           UUID REFERENCES master_data_items (id);

-- Bidang materi s/d topik WAJIB mengisi ketiga kolom; type lain WAJIB kosong.
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_curriculum_check
    CHECK (
        CASE WHEN type IN ('subject_field', 'chapter', 'subchapter', 'topic')
             THEN subject_id IS NOT NULL AND education_level_id IS NOT NULL AND grade_id IS NOT NULL
             ELSE subject_id IS NULL     AND education_level_id IS NULL     AND grade_id IS NULL
        END
    );

-- Kelas harus milik jenjang yang diisi (Kelas 4 + SMP ditolak).
-- Caranya: pasangan (grade_id, education_level_id) harus sama dengan (id, parent_id) milik kelas tersebut.
-- Foreign key ke 2 kolom butuh UNIQUE di 2 kolom tujuan (id sudah unik, jadi pasti aman).
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_id_parent_unique
    UNIQUE (id, parent_id);

ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_grade_level_fkey
    FOREIGN KEY (grade_id, education_level_id) REFERENCES master_data_items (id, parent_id);


-- ----------------------------------------------------------------------------
-- 3. Index
-- ----------------------------------------------------------------------------

-- Nama tidak boleh kembar dalam satu type, induk, dan kelas yang sama.
-- grade_id ikut dicek supaya "Bilangan" boleh ada di Kelas 1 dan Kelas 4 dengan induk yang sama (Matematika).
DROP INDEX master_data_items_type_parent_name_unique;
CREATE UNIQUE INDEX master_data_items_type_parent_name_unique
    ON master_data_items (type, parent_id, grade_id, LOWER(name)) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;

-- Untuk dropdown/filter: "semua bab Matematika Kelas 4" urut sort_order.
CREATE INDEX master_data_items_curriculum_idx
    ON master_data_items (subject_id, grade_id, type, sort_order)
    WHERE deleted_at IS NULL AND subject_id IS NOT NULL;


-- ----------------------------------------------------------------------------
-- 4. Keterangan kolom
-- ----------------------------------------------------------------------------
COMMENT ON COLUMN master_data_items.type IS
    'education_level = jenjang, grade = kelas, subject = mapel, subject_field = bidang materi, chapter = bab, subchapter = sub bab, topic = topik. Type baru ditambah lewat migration';
COMMENT ON COLUMN master_data_items.parent_id IS
    'Item induk, contoh Kelas 7 → SMP, Bab Pecahan → Bidang Materi Bilangan. Kosong untuk jenjang dan mapel';
COMMENT ON COLUMN master_data_items.subject_id IS
    'Mapel (bidang materi s/d topik). Disalin dari induk supaya filter tidak perlu JOIN';
COMMENT ON COLUMN master_data_items.education_level_id IS
    'Jenjang (bidang materi s/d topik). Disalin dari induk';
COMMENT ON COLUMN master_data_items.grade_id IS
    'Kelas (bidang materi s/d topik). Disalin dari induk, harus milik education_level_id';
