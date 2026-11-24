-- ============================================================================
-- Index untuk GET /api/v1/questions (list Bank Soal): filter, pencarian, urutan.
-- ============================================================================

-- pg_trgm membuat pencarian ILIKE '%kata%' bisa memakai index (tanpa ini PostgreSQL
-- harus membaca semua soal satu per satu). Extension ini "trusted" (PostgreSQL 13+),
-- jadi cukup pemilik database, tidak perlu superuser.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Pencarian berdasarkan teks soal & kode soal (contoh "garis bilangan", "Q-126").
CREATE INDEX questions_text_trgm_idx
    ON questions USING GIN (question_text gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX questions_code_trgm_idx
    ON questions USING GIN (code gin_trgm_ops)
    WHERE deleted_at IS NULL;

-- Urutan list: terbaru dulu. id ikut di index sebagai pemecah seri, karena soal yang
-- dibuat dalam 1 request (bulk create) punya created_at yang sama persis.
DROP INDEX questions_created_at_idx;
CREATE INDEX questions_created_at_idx
    ON questions (created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Filter per tingkat hierarki. Mapel/jenjang/kelas sudah ada di
-- questions_subject_level_grade_idx; questions_chapter_difficulty_idx hanya untuk soal
-- aktif, sedangkan list juga menampilkan soal nonaktif.
CREATE INDEX questions_subject_field_idx ON questions (subject_field_id) WHERE deleted_at IS NULL;
CREATE INDEX questions_chapter_idx       ON questions (chapter_id)       WHERE deleted_at IS NULL;
CREATE INDEX questions_subchapter_idx    ON questions (subchapter_id)    WHERE deleted_at IS NULL;
CREATE INDEX questions_topic_idx         ON questions (topic_id)         WHERE deleted_at IS NULL;
