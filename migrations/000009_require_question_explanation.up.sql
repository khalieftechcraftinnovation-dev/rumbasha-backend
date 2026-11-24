-- Pembahasan (explanation) wajib diisi untuk semua tipe soal. (BRD FR-QUIZ-033)
--
-- Soal lama yang belum punya pembahasan diberi teks penanda supaya constraint bisa dipasang.
-- Cari dan lengkapi lewat:  SELECT code FROM questions WHERE explanation = '(pembahasan belum diisi)';
UPDATE questions
SET explanation = '(pembahasan belum diisi)', updated_at = NOW()
WHERE explanation IS NULL OR btrim(explanation) = '';

ALTER TABLE questions ALTER COLUMN explanation SET NOT NULL;

-- Tidak boleh hanya berisi spasi.
ALTER TABLE questions ADD CONSTRAINT questions_explanation_not_blank
    CHECK (btrim(explanation) <> '');

COMMENT ON COLUMN questions.explanation IS 'Pembahasan soal, wajib untuk semua tipe';
