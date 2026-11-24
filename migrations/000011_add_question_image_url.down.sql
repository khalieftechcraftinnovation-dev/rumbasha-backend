UPDATE questions
SET answer = jsonb_set(answer, '{input_type}', '"string"')
WHERE type = 'free_text' AND answer->>'input_type' = 'text';

-- PERHATIAN: link gambar (image_url) soal ikut terhapus.
ALTER TABLE questions DROP CONSTRAINT questions_one_image_source;
ALTER TABLE questions DROP COLUMN image_url;
