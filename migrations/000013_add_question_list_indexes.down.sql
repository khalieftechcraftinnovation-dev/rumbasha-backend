DROP INDEX IF EXISTS questions_topic_idx;
DROP INDEX IF EXISTS questions_subchapter_idx;
DROP INDEX IF EXISTS questions_chapter_idx;
DROP INDEX IF EXISTS questions_subject_field_idx;

DROP INDEX IF EXISTS questions_created_at_idx;
CREATE INDEX questions_created_at_idx
    ON questions (created_at DESC)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS questions_code_trgm_idx;
DROP INDEX IF EXISTS questions_text_trgm_idx;

-- Hanya dipakai index di atas, jadi ikut dihapus.
DROP EXTENSION IF EXISTS pg_trgm;
