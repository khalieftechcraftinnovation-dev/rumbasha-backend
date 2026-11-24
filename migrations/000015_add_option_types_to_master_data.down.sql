DELETE FROM master_data_items WHERE type IN ('question_type', 'quiz_type', 'difficulty');

DROP INDEX IF EXISTS master_data_items_dropdown_idx;

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_code_required_check;

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_parent_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_parent_check
    CHECK ((type IN ('education_level', 'subject')) = (parent_id IS NULL));

ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN ('education_level', 'grade', 'subject', 'subject_field', 'chapter', 'subchapter', 'topic'));

COMMENT ON COLUMN master_data_items.type IS
    'education_level = jenjang, grade = kelas, subject = mapel, subject_field = bidang materi, chapter = bab, subchapter = sub bab, topic = topik. Type baru ditambah lewat migration';
COMMENT ON COLUMN master_data_items.code IS 'Kode tetap untuk program, contoh PRA_SD, SD, NOL_KECIL, GRADE_7';
