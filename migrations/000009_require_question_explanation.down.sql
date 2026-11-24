-- Pembahasan kembali opsional. Teks penanda dari migration up tidak dihapus.
ALTER TABLE questions DROP CONSTRAINT questions_explanation_not_blank;
ALTER TABLE questions ALTER COLUMN explanation DROP NOT NULL;
COMMENT ON COLUMN questions.explanation IS NULL;
