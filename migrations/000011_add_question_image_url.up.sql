-- 1. Gambar soal boleh berupa link luar (image_url), selain gambar dari Image Library (image_id).
--    Satu soal hanya boleh memakai salah satu. Gambar opsi jawaban disimpan di JSON answer
--    (options[].image_url), jadi tidak perlu kolom baru.
ALTER TABLE questions ADD COLUMN image_url TEXT;

ALTER TABLE questions ADD CONSTRAINT questions_one_image_source
    CHECK (image_id IS NULL OR image_url IS NULL);

COMMENT ON COLUMN questions.image_url IS 'Link gambar soal dari luar (http/https). Tidak boleh bersamaan dengan image_id';

-- 2. Soal free_text: input_type "string" diganti namanya menjadi "text" (sama dengan template Excel).
UPDATE questions
SET answer = jsonb_set(answer, '{input_type}', '"text"')
WHERE type = 'free_text' AND answer->>'input_type' = 'string';
