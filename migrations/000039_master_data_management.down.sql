ALTER TABLE branches
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS master_data_items_name_trgm_idx;
ALTER TABLE master_data_items
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS is_system;

-- Data sekolah ikut terhapus supaya constraint type lama bisa dipasang lagi.
DELETE FROM master_data_items WHERE type = 'school';
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN (
        'education_level', 'grade', 'subject', 'subject_field', 'chapter', 'subchapter', 'topic',
        'question_type', 'quiz_type', 'difficulty'
    ));
