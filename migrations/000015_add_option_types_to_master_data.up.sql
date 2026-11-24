-- ============================================================================
-- Master data untuk dropdown: tipe soal, tipe quiz, dan level (tingkat kesulitan).
--
-- Kolom "code" berisi nilai yang DISIMPAN di tabel lain apa adanya (bukan relasi id):
--   questions.type       ← question_type.code  (single_choice, ...)
--   questions.used_for   ← quiz_type.code      (daily_quiz, ...)
--   questions.difficulty ← difficulty.code     (easy, medium, hard)
-- Jadi kode di sini TIDAK BOLEH diubah; yang boleh diubah hanya nama tampilan & urutan.
-- Data dibutuhkan aplikasi, jadi diisi di migration (sama seperti jenjang & kelas).
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Type baru
-- ----------------------------------------------------------------------------
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_type_check;

ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_type_check
    CHECK (type IN (
        'education_level',  -- jenjang          (tanpa induk)
        'grade',            -- kelas            (induk: education_level)
        'subject',          -- mata pelajaran   (tanpa induk, sama untuk semua jenjang)
        'subject_field',    -- bidang materi    (induk: subject)
        'chapter',          -- bab              (induk: subject_field)
        'subchapter',       -- sub bab          (induk: chapter)
        'topic',            -- topik            (induk: subchapter)
        'question_type',    -- tipe soal        (tanpa induk, code = questions.type)
        'quiz_type',        -- tipe quiz        (tanpa induk, code = isi questions.used_for)
        'difficulty'        -- level            (tanpa induk, code = questions.difficulty)
    ));

-- Hanya type hierarki yang wajib punya induk; type lain wajib tanpa induk.
ALTER TABLE master_data_items DROP CONSTRAINT master_data_items_parent_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_parent_check
    CHECK ((type IN ('grade', 'subject_field', 'chapter', 'subchapter', 'topic')) = (parent_id IS NOT NULL));

-- Type yang nilainya disimpan sebagai teks wajib punya code (code itulah yang disimpan).
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_code_required_check
    CHECK (type NOT IN ('question_type', 'quiz_type', 'difficulty') OR code IS NOT NULL);


-- ----------------------------------------------------------------------------
-- 2. Index dropdown: semua query dropdown berbentuk
--      WHERE type = ? AND parent_id = ? (atau IS NULL) ORDER BY sort_order
-- ----------------------------------------------------------------------------
CREATE INDEX master_data_items_dropdown_idx
    ON master_data_items (type, parent_id, sort_order)
    WHERE deleted_at IS NULL;


-- ----------------------------------------------------------------------------
-- 3. Data
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, code, name, sort_order) VALUES
    -- Nama mengikuti tab Bank Soal.
    ('question_type', 'single_choice',   'Pilihan ganda', 1),
    ('question_type', 'multiple_choice', 'PG kompleks',   2),
    ('question_type', 'true_false',      'Benar / Salah', 3),
    ('question_type', 'free_text',       'Isian',         4),

    ('quiz_type', 'daily_quiz', 'Daily Quiz', 1),
    ('quiz_type', 'uji_kom',    'Uji Kom',    2),
    ('quiz_type', 'tryout',     'Tryout',     3),
    ('quiz_type', 'rumath',     'RUMATH',     4),

    ('difficulty', 'easy',   'Mudah',  1),
    ('difficulty', 'medium', 'Sedang', 2),
    ('difficulty', 'hard',   'Sulit',  3);

COMMENT ON COLUMN master_data_items.type IS
    'education_level = jenjang, grade = kelas, subject = mapel, subject_field = bidang materi, chapter = bab, subchapter = sub bab, topic = topik, question_type = tipe soal, quiz_type = tipe quiz, difficulty = level. Type baru ditambah lewat migration';
COMMENT ON COLUMN master_data_items.code IS
    'Kode tetap untuk program, contoh SD, GRADE_7. Untuk question_type, quiz_type, difficulty: nilai yang disimpan di tabel lain (contoh questions.difficulty = easy), jangan diubah';
