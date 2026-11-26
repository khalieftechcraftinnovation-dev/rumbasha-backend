-- ============================================================================
-- DATA DUMMY soal (Bank Soal) Matematika SD Kelas 1-6 + contoh SMP. JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" SETELAH dummy_subjects.sql (butuh topik Matematika SD).
-- Aman dijalankan berkali-kali: soal dengan topik & teks yang sama dilewati.
--
-- Isi:
--   - Setiap topik mendapat soal di 3 level (easy, medium, hard). Jumlahnya dihitung supaya setiap
--     bidang materi punya MINIMAL 15 soal per level → Daily Quiz 5 / 10 / 15 soal selalu bisa
--     di-generate untuk semua bidang materi & level. Total ±730 soal.
--   - 4 tipe bergiliran: single choice, multiple choice, benar/salah, isian (angka).
--   - Soal hitungan: easy = penjumlahan, medium = perkalian, hard = perkalian lalu pengurangan.
--     Angka dibuat dari nomor soal, jadi hasilnya selalu sama setiap seed dijalankan.
--   - Peruntukan: Daily Quiz & Tryout; Kelas 6, 9, 12 juga Uji Kom (Uji Kom hanya kelas itu).
--   - Dibuat oleh admin.sbytimur, tanggal dibuat tersebar 0-59 hari terakhir.
-- ============================================================================

BEGIN;

WITH topics AS (
    -- Semua topik Matematika + letak lengkapnya. topics_in_field = jumlah topik di bidang materinya.
    SELECT t.id AS topic_id, t.name AS topic_name, t.parent_id AS subchapter_id,
           sc.parent_id AS chapter_id, c.parent_id AS field_id,
           t.subject_id, t.education_level_id, t.grade_id, g.code AS grade_code,
           COUNT(*) OVER (PARTITION BY c.parent_id) AS topics_in_field,
           ROW_NUMBER() OVER (ORDER BY g.sort_order, c.parent_id, sc.sort_order, t.sort_order, t.id) AS topic_seq
    FROM master_data_items t
    JOIN master_data_items sc ON sc.id = t.parent_id
    JOIN master_data_items c  ON c.id  = sc.parent_id
    JOIN master_data_items g  ON g.id  = t.grade_id
    JOIN master_data_items s  ON s.id  = t.subject_id AND s.type = 'subject' AND s.code = 'MTK'
    WHERE t.type = 'topic' AND t.deleted_at IS NULL
),
items AS (
    -- k = nomor soal di topik & level itu. Per topik per level minimal 3, dan cukup untuk
    -- 15 soal per bidang materi per level.
    SELECT tp.*, d.difficulty, d.level_no, k
    FROM topics tp
    CROSS JOIN (VALUES ('easy', 1), ('medium', 2), ('hard', 3)) AS d (difficulty, level_no)
    CROSS JOIN LATERAL generate_series(1, GREATEST(3, CEIL(15.0 / tp.topics_in_field)::INT)) AS k
),
numbers AS (
    -- a selalu berbeda untuk k berbeda (a = basis + k), jadi teks soal di 1 topik tidak kembar.
    SELECT i.*,
           CASE i.difficulty WHEN 'easy' THEN 4 + i.k + (i.topic_seq % 10)
                             WHEN 'medium' THEN 11 + i.k + (i.topic_seq % 9)
                             ELSE 21 + i.k + (i.topic_seq % 15) END                         AS a,
           CASE i.difficulty WHEN 'easy' THEN 3 + ((i.topic_seq + i.k * 7) % 25)
                             ELSE 2 + ((i.topic_seq + i.k) % 8) END                         AS b,
           1 + ((i.topic_seq * 3 + i.k) % 9)                                                AS c,
           ((i.topic_seq + i.k + i.level_no) % 4)::INT                                      AS type_no,
           ((i.topic_seq * 5 + i.k) % 4)::INT                                               AS correct_pos
    FROM items i
),
exprs AS (
    SELECT n.*,
           CASE n.difficulty WHEN 'easy'   THEN n.a || ' + ' || n.b
                             WHEN 'medium' THEN n.a || ' × ' || n.b
                             ELSE n.a || ' × ' || n.b || ' − ' || n.c END                   AS expr,
           CASE n.difficulty WHEN 'easy'   THEN n.a + n.b
                             WHEN 'medium' THEN n.a * n.b
                             ELSE n.a * n.b - n.c END                                       AS result
    FROM numbers n
),
built AS (
    SELECT e.*,
           (ARRAY['single_choice', 'multiple_choice', 'true_false', 'free_text'])[e.type_no + 1] AS question_type,
           CASE e.type_no
               WHEN 0 THEN format('Latihan %s: hasil dari %s adalah ...', e.topic_name, e.expr)
               WHEN 1 THEN format('Latihan %s: pilih semua bilangan yang LEBIH BESAR dari hasil %s.', e.topic_name, e.expr)
               WHEN 2 THEN format('Latihan %s: tentukan benar atau salah setiap pernyataan tentang %s.', e.topic_name, e.expr)
               ELSE        format('Latihan %s: tuliskan hasil dari %s.', e.topic_name, e.expr)
           END AS question_text,
           CASE e.type_no
               -- Single choice: 4 pilihan (hasil, +1, +2, -1), kunci di posisi correct_pos.
               WHEN 0 THEN jsonb_build_object(
                   'options', (SELECT jsonb_agg(jsonb_build_object(
                                       'key', chr(65 + p),
                                       'content', (e.result + (ARRAY[0, 1, 2, -1])[((p - e.correct_pos + 4) % 4) + 1])::TEXT,
                                       'image_id', NULL) ORDER BY p)
                               FROM generate_series(0, 3) AS p),
                   'correct_key', chr(65 + e.correct_pos))
               -- Multiple choice: B & C lebih besar dari hasil.
               WHEN 1 THEN jsonb_build_object(
                   'options', jsonb_build_array(
                       jsonb_build_object('key', 'A', 'content', (e.result - 2)::TEXT, 'image_id', NULL),
                       jsonb_build_object('key', 'B', 'content', (e.result + 3)::TEXT, 'image_id', NULL),
                       jsonb_build_object('key', 'C', 'content', (e.result + 1)::TEXT, 'image_id', NULL),
                       jsonb_build_object('key', 'D', 'content', (e.result - 5)::TEXT, 'image_id', NULL)),
                   'correct_keys', jsonb_build_array('B', 'C'))
               -- Benar/salah: A benar, B salah (dibalik untuk correct_pos ganjil).
               WHEN 2 THEN jsonb_build_object(
                   'options', jsonb_build_array(
                       jsonb_build_object('key', 'A', 'content', e.expr || ' = ' || (e.result + CASE WHEN e.correct_pos % 2 = 0 THEN 0 ELSE 1 END), 'image_id', NULL),
                       jsonb_build_object('key', 'B', 'content', e.expr || ' = ' || (e.result + CASE WHEN e.correct_pos % 2 = 0 THEN 1 ELSE 0 END), 'image_id', NULL)),
                   'correct_keys', jsonb_build_array(
                       jsonb_build_object('key', 'A', 'correct', e.correct_pos % 2 = 0),
                       jsonb_build_object('key', 'B', 'correct', e.correct_pos % 2 = 1)))
               -- Isian angka.
               ELSE jsonb_build_object('input_type', 'number', 'accepted_answers', jsonb_build_array(e.result::TEXT))
           END AS answer,
           CASE e.type_no
               WHEN 1 THEN format('Pembahasan: %s = %s. Bilangan yang lebih besar dari %s adalah %s dan %s.',
                                  e.expr, e.result, e.result, e.result + 3, e.result + 1)
               ELSE        format('Pembahasan: %s = %s.', e.expr, e.result)
           END AS explanation
    FROM exprs e
)
INSERT INTO questions (type, question_text, explanation, display_order,
                       education_level_id, grade_id, subject_id, subject_field_id, chapter_id, subchapter_id, topic_id,
                       difficulty, used_for, status, answer, created_by, created_at, updated_at)
SELECT b.question_type, b.question_text, b.explanation, '["question"]'::JSONB,
       b.education_level_id, b.grade_id, b.subject_id, b.field_id, b.chapter_id, b.subchapter_id, b.topic_id,
       b.difficulty,
       CASE WHEN b.grade_code IN ('GRADE_6', 'GRADE_9', 'GRADE_12') THEN '["daily_quiz", "uji_kom", "tryout"]'::JSONB
            ELSE '["daily_quiz", "tryout"]'::JSONB END,
       'active', b.answer,
       (SELECT id FROM users WHERE username = 'admin.sbytimur'),
       NOW() - make_interval(days => (b.topic_seq * 7 + b.k * 3 + b.level_no)::INT % 60,
                             mins => (b.topic_seq * 13 + b.k)::INT % 1440),
       NOW() - make_interval(days => (b.topic_seq * 7 + b.k * 3 + b.level_no)::INT % 60,
                             mins => (b.topic_seq * 13 + b.k)::INT % 1440)
FROM built b
WHERE NOT EXISTS (
    SELECT 1 FROM questions q WHERE q.topic_id = b.topic_id AND q.question_text = b.question_text
)
ORDER BY b.topic_seq, b.level_no, b.k;  -- kode soal (Q-...) urut topik

COMMIT;


-- ----------------------------------------------------------------------------
-- Cek hasilnya (jalankan terpisah di DBeaver): jumlah soal aktif per bidang materi & level.
-- Daily Quiz butuh minimal 15 per baris untuk pilihan 15 soal.
--
-- SELECT g.name AS kelas, f.name AS bidang, q.difficulty, COUNT(*) AS soal
-- FROM questions q
-- JOIN master_data_items f ON f.id = q.subject_field_id
-- JOIN master_data_items g ON g.id = q.grade_id
-- WHERE q.deleted_at IS NULL AND q.status = 'active'
-- GROUP BY g.sort_order, g.name, f.name, q.difficulty
-- ORDER BY g.sort_order, f.name, q.difficulty;
-- ----------------------------------------------------------------------------
