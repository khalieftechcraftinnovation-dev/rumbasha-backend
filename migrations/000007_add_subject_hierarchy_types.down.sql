-- Kembalikan master_data_items ke type awal (jenjang & kelas saja).
-- PERHATIAN: semua data mapel → topik ikut terhapus.

-- Hapus dari tingkat paling bawah dulu, karena parent_id tidak boleh menunjuk ke data yang sudah dihapus.
DELETE FROM master_data_items WHERE type = 'topic';
DELETE FROM master_data_items WHERE type = 'subchapter';
DELETE FROM master_data_items WHERE type = 'chapter';
DELETE FROM master_data_items WHERE type = 'subject_field';
DELETE FROM master_data_items WHERE type = 'subject';

DROP INDEX master_data_items_curriculum_idx;
DROP INDEX master_data_items_type_parent_name_unique;
CREATE UNIQUE INDEX master_data_items_type_parent_name_unique
    ON master_data_items (type, parent_id, LOWER(name)) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_grade_level_fkey;
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_id_parent_unique;
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_curriculum_check;
ALTER TABLE master_data_items
    DROP COLUMN subject_id,
    DROP COLUMN education_level_id,
    DROP COLUMN grade_id;

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_parent_check;
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN ('education_level', 'grade'));

COMMENT ON COLUMN master_data_items.type IS
    'education_level = jenjang, grade = kelas. Type baru ditambah lewat migration';
COMMENT ON COLUMN master_data_items.parent_id IS
    'Item induk, contoh Kelas 7 → SMP';
