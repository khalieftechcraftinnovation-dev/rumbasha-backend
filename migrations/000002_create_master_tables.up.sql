-- ============================================================================
-- Data acuan (master data) yang dibutuhkan akun user. (BRD 3.16)
-- ============================================================================

-- Cabang Rumbasha (contoh dari desain: Surabaya Timur, Surabaya Barat, Sidoarjo).
-- Menjadi atribut akun admin, tutor, dan siswa. (FR-MST-007, FR-STU-017)
CREATE TABLE branches (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    status     VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ  -- terisi = dihapus (soft delete), data lama tetap aman
);

-- Nama cabang tidak boleh kembar (huruf besar/kecil dianggap sama).
-- WHERE deleted_at IS NULL: cabang yang sudah dihapus tidak ikut dihitung.
CREATE UNIQUE INDEX branches_name_unique
    ON branches (LOWER(name))
    WHERE deleted_at IS NULL;

COMMENT ON TABLE  branches            IS 'Cabang Rumbasha';
COMMENT ON COLUMN branches.deleted_at IS 'Terisi = sudah dihapus (soft delete)';


-- Satu tabel untuk semua jenis data acuan, dibedakan lewat kolom "type".
-- Mirip menu Master Data di desain (tab Kelas, Mapel, Bidang materi, ...).
--
-- parent_id menunjuk ke item induknya, contoh:
--   Kelas 7 (type grade) → parent: SMP (type education_level)
CREATE TABLE master_data_items (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    type       VARCHAR(30)  NOT NULL CHECK (type IN ('education_level', 'grade')),
    code       VARCHAR(30),  -- kode tetap untuk dipakai di program, contoh: SD, GRADE_7
    name       VARCHAR(150) NOT NULL,
    parent_id  UUID         REFERENCES master_data_items (id),
    sort_order INT          NOT NULL DEFAULT 0,  -- urutan tampil di dropdown
    status     VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Kode tidak boleh kembar dalam satu type (kode boleh kosong untuk item buatan admin).
CREATE UNIQUE INDEX master_data_items_type_code_unique
    ON master_data_items (type, code);

-- Nama tidak boleh kembar dalam satu type dan induk yang sama.
-- NULLS NOT DISTINCT: item tanpa induk (parent_id kosong) juga ikut dicek.
CREATE UNIQUE INDEX master_data_items_type_parent_name_unique
    ON master_data_items (type, parent_id, LOWER(name)) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;

CREATE INDEX master_data_items_parent_id_idx ON master_data_items (parent_id);

COMMENT ON TABLE  master_data_items           IS 'Data acuan generik (jenjang, kelas, ...), dibedakan oleh kolom type';
COMMENT ON COLUMN master_data_items.type      IS 'education_level = jenjang, grade = kelas. Type baru ditambah lewat migration';
COMMENT ON COLUMN master_data_items.code      IS 'Kode tetap untuk program, contoh PRA_SD, SD, NOL_KECIL, GRADE_7';
COMMENT ON COLUMN master_data_items.parent_id IS 'Item induk, contoh Kelas 7 → SMP';


-- ----------------------------------------------------------------------------
-- Data awal jenjang dan kelas. Data ini dibutuhkan aplikasi, jadi diisi di migration.
--
--   PraSD → Nol Kecil, Nol Besar
--   SD    → Kelas 1-6
--   SMP   → Kelas 7-9
--   SMA   → Kelas 10-12
-- ----------------------------------------------------------------------------

-- Jenjang
INSERT INTO master_data_items (type, code, name, sort_order) VALUES
    ('education_level', 'PRA_SD', 'PraSD', 1),
    ('education_level', 'SD',     'SD',    2),
    ('education_level', 'SMP',    'SMP',   3),
    ('education_level', 'SMA',    'SMA',   4);

-- Kelas PraSD: Nol Kecil dan Nol Besar
INSERT INTO master_data_items (type, code, name, parent_id, sort_order)
SELECT 'grade', v.code, v.name, level.id, v.sort_order
FROM (VALUES
        ('NOL_KECIL', 'Nol Kecil', 1),
        ('NOL_BESAR', 'Nol Besar', 2)
     ) AS v (code, name, sort_order)
JOIN master_data_items AS level
  ON level.type = 'education_level'
 AND level.code = 'PRA_SD';

-- Kelas 1-12, masing-masing berinduk ke jenjangnya.
-- generate_series(1, 12) menghasilkan angka 1 sampai 12.
-- sort_order = g + 2, supaya Kelas 1 tampil setelah Nol Kecil (1) dan Nol Besar (2).
INSERT INTO master_data_items (type, code, name, parent_id, sort_order)
SELECT 'grade',
       'GRADE_' || g,
       'Kelas ' || g,
       level.id,
       g + 2
FROM generate_series(1, 12) AS g
JOIN master_data_items AS level
  ON level.type = 'education_level'
 AND level.code = CASE
                      WHEN g <= 6 THEN 'SD'
                      WHEN g <= 9 THEN 'SMP'
                      ELSE             'SMA'
                  END;
