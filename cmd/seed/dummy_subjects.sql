-- ============================================================================
-- DATA DUMMY hierarki mapel: Matematika SD (Kelas 1-6) + contoh kecil SMP (Kelas 7-9).
-- JANGAN dijalankan di production.
--
--   Matematika (mapel, sama untuk semua jenjang)
--     → Bidang Materi (+ jenjang & kelas) → Bab → Sub Bab → Topik
--
-- Cara pakai: make seed   (atau buka file ini di DBeaver lalu Alt+X)
-- Aman dijalankan berulang kali: data yang sudah ada dilewati (ON CONFLICT DO NOTHING).
-- Membutuhkan migration 000007 (type subject s/d topic di master_data_items).
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Mapel (tanpa induk, dipakai semua jenjang). Hanya Matematika yang diisi sampai topik.
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, code, name, sort_order) VALUES
    ('subject', 'MTK', 'Matematika',   1),
    ('subject', 'IPA', 'IPA',          2),
    ('subject', 'BIN', 'B. Indonesia', 3),
    ('subject', 'BIG', 'B. Inggris',   4),
    ('subject', 'IPS', 'IPS',          5)
ON CONFLICT DO NOTHING;


-- ----------------------------------------------------------------------------
-- Daftar lengkap Matematika SD: satu baris = satu topik beserta kelas dan jalur induknya.
-- Untuk menambah data, cukup tambah baris di sini. Angka = urutan tampil (sort_order).
-- Tabel sementara (TEMP) otomatis hilang setelah transaksi selesai.
-- ----------------------------------------------------------------------------
CREATE TEMP TABLE tmp_math_sd (
    grade_code      TEXT,
    field_name      TEXT, field_sort      INT,
    chapter_name    TEXT, chapter_sort    INT,
    subchapter_name TEXT, subchapter_sort INT,
    topic_name      TEXT, topic_sort      INT
) ON COMMIT DROP;

INSERT INTO tmp_math_sd VALUES
    -- Kelas,   Bidang Materi,      Bab,                          Sub Bab,                          Topik
    ('GRADE_1', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Nilai Tempat', 1,                'Satuan dan puluhan', 1),
    ('GRADE_1', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Nilai Tempat', 1,                'Membaca dan menulis bilangan', 2),
    ('GRADE_1', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Nilai Tempat', 1,                'Membandingkan dan mengurutkan bilangan', 3),
    ('GRADE_1', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Penjumlahan dan Pengurangan', 2, 'Penjumlahan', 1),
    ('GRADE_1', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Penjumlahan dan Pengurangan', 2, 'Pengurangan', 2),
    ('GRADE_1', 'Geometri', 2,      'Bangun Datar', 1,            'Mengenal Bangun Datar', 1,       'Persegi, segitiga, dan lingkaran', 1),

    ('GRADE_2', 'Bilangan', 1,      'Bilangan Cacah', 1,          'Nilai Tempat', 1,                'Ratusan', 1),
    ('GRADE_2', 'Bilangan', 1,      'Perkalian dan Pembagian', 2, 'Perkalian', 1,                   'Perkalian 1 sampai 10', 1),
    ('GRADE_2', 'Bilangan', 1,      'Perkalian dan Pembagian', 2, 'Pembagian', 2,                   'Pembagian dasar', 1),
    ('GRADE_2', 'Pengukuran', 2,    'Satuan Waktu', 1,            'Membaca Jam', 1,                 'Jam dan menit', 1),
    ('GRADE_2', 'Pengukuran', 2,    'Satuan Waktu', 1,            'Kalender', 2,                    'Hari, minggu, bulan, dan tahun', 1),

    ('GRADE_3', 'Bilangan', 1,      'Pecahan', 1,                 'Mengenal Pecahan', 1,            'Pecahan sederhana', 1),
    ('GRADE_3', 'Bilangan', 1,      'Pecahan', 1,                 'Mengenal Pecahan', 1,            'Membandingkan pecahan', 2),
    ('GRADE_3', 'Geometri', 2,      'Bangun Datar', 1,            'Keliling Bangun Datar', 1,       'Keliling persegi dan persegi panjang', 1),
    ('GRADE_3', 'Geometri', 2,      'Bangun Datar', 1,            'Keliling Bangun Datar', 1,       'Keliling segitiga', 2),
    ('GRADE_3', 'Pengukuran', 3,    'Satuan Pengukuran', 1,       'Satuan Panjang dan Berat', 1,    'Konversi satuan panjang', 1),
    ('GRADE_3', 'Pengukuran', 3,    'Satuan Pengukuran', 1,       'Satuan Panjang dan Berat', 1,    'Konversi satuan berat', 2),

    ('GRADE_4', 'Bilangan', 1,      'KPK dan FPB', 1,             'Faktor dan Kelipatan', 1,        'Bilangan prima', 1),
    ('GRADE_4', 'Bilangan', 1,      'KPK dan FPB', 1,             'Faktor dan Kelipatan', 1,        'Faktorisasi prima', 2),
    ('GRADE_4', 'Bilangan', 1,      'KPK dan FPB', 1,             'Menentukan KPK dan FPB', 2,      'Menentukan KPK', 1),
    ('GRADE_4', 'Bilangan', 1,      'KPK dan FPB', 1,             'Menentukan KPK dan FPB', 2,      'Menentukan FPB', 2),
    ('GRADE_4', 'Bilangan', 1,      'KPK dan FPB', 1,             'Menentukan KPK dan FPB', 2,      'Soal cerita KPK dan FPB', 3),
    ('GRADE_4', 'Bilangan', 1,      'Pecahan', 2,                 'Pecahan Senilai', 1,             'Pecahan senilai', 1),
    ('GRADE_4', 'Bilangan', 1,      'Pecahan', 2,                 'Pecahan Senilai', 1,             'Menyederhanakan pecahan', 2),
    ('GRADE_4', 'Geometri', 2,      'Bangun Datar', 1,            'Luas Bangun Datar', 1,           'Luas persegi dan persegi panjang', 1),
    ('GRADE_4', 'Geometri', 2,      'Bangun Datar', 1,            'Luas Bangun Datar', 1,           'Luas segitiga', 2),
    ('GRADE_4', 'Pengukuran', 3,    'Sudut', 1,                   'Mengenal Sudut', 1,              'Jenis-jenis sudut', 1),
    ('GRADE_4', 'Pengukuran', 3,    'Sudut', 1,                   'Mengenal Sudut', 1,              'Mengukur sudut dengan busur', 2),

    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Operasi Pecahan', 1,             'Penjumlahan pecahan', 1),
    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Operasi Pecahan', 1,             'Pengurangan pecahan', 2),
    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Operasi Pecahan', 1,             'Perkalian pecahan', 3),
    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Operasi Pecahan', 1,             'Pembagian pecahan', 4),
    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Desimal dan Persen', 2,          'Mengubah pecahan ke desimal', 1),
    ('GRADE_5', 'Bilangan', 1,      'Pecahan', 1,                 'Desimal dan Persen', 2,          'Mengubah pecahan ke persen', 2),
    ('GRADE_5', 'Geometri', 2,      'Bangun Ruang', 1,            'Volume Bangun Ruang', 1,         'Volume kubus', 1),
    ('GRADE_5', 'Geometri', 2,      'Bangun Ruang', 1,            'Volume Bangun Ruang', 1,         'Volume balok', 2),

    ('GRADE_6', 'Bilangan', 1,      'Bilangan Bulat', 1,          'Operasi Bilangan Bulat', 1,      'Penjumlahan dan pengurangan bilangan bulat', 1),
    ('GRADE_6', 'Bilangan', 1,      'Bilangan Bulat', 1,          'Operasi Bilangan Bulat', 1,      'Operasi hitung campuran', 2),
    ('GRADE_6', 'Geometri', 2,      'Bangun Datar', 1,            'Lingkaran', 1,                   'Keliling lingkaran', 1),
    ('GRADE_6', 'Geometri', 2,      'Bangun Datar', 1,            'Lingkaran', 1,                   'Luas lingkaran', 2),
    ('GRADE_6', 'Geometri', 2,      'Bangun Ruang', 2,            'Mengenal Bangun Ruang', 1,       'Sisi, rusuk, dan titik sudut', 1),
    ('GRADE_6', 'Geometri', 2,      'Bangun Ruang', 2,            'Mengenal Bangun Ruang', 1,       'Jaring-jaring bangun ruang', 2),
    ('GRADE_6', 'Analisis Data dan Peluang', 3, 'Pengolahan Data', 1, 'Penyajian Data', 1,          'Tabel', 1),
    ('GRADE_6', 'Analisis Data dan Peluang', 3, 'Pengolahan Data', 1, 'Penyajian Data', 1,          'Diagram batang', 2),
    ('GRADE_6', 'Analisis Data dan Peluang', 3, 'Pengolahan Data', 1, 'Penyajian Data', 1,          'Diagram lingkaran', 3),
    ('GRADE_6', 'Analisis Data dan Peluang', 3, 'Pengolahan Data', 1, 'Ukuran Pemusatan Data', 2,   'Rata-rata (mean)', 1),
    ('GRADE_6', 'Analisis Data dan Peluang', 3, 'Pengolahan Data', 1, 'Ukuran Pemusatan Data', 2,   'Modus dan median', 2),

    -- SMP (contoh kecil, untuk mencoba batas jenjang di modul & soal)
    ('GRADE_7', 'Bilangan', 1,      'Bilangan Bulat', 1,          'Operasi Bilangan Bulat', 1,      'Penjumlahan dan pengurangan bilangan bulat', 1),
    ('GRADE_7', 'Bilangan', 1,      'Bilangan Bulat', 1,          'Operasi Bilangan Bulat', 1,      'Perkalian dan pembagian bilangan bulat', 2),
    ('GRADE_7', 'Aljabar', 2,       'Bentuk Aljabar', 1,          'Operasi Bentuk Aljabar', 1,      'Suku sejenis', 1),
    ('GRADE_8', 'Geometri', 1,      'Teorema Pythagoras', 1,      'Tripel Pythagoras', 1,           'Menentukan sisi segitiga siku-siku', 1),
    ('GRADE_9', 'Aljabar', 1,       'Persamaan Kuadrat', 1,       'Akar Persamaan Kuadrat', 1,      'Pemfaktoran', 1);


-- ----------------------------------------------------------------------------
-- 2. Bidang Materi (induk: Matematika). Jenjang diambil dari induk kelasnya (Kelas 4 → SD).
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id, sort_order)
SELECT DISTINCT 'subject_field', t.field_name, subject.id, subject.id, grade.parent_id, grade.id, t.field_sort
FROM tmp_math_sd AS t
JOIN master_data_items AS subject ON subject.type = 'subject' AND subject.code = 'MTK'
JOIN master_data_items AS grade   ON grade.type = 'grade' AND grade.code = t.grade_code
ON CONFLICT DO NOTHING;


-- ----------------------------------------------------------------------------
-- 3. Bab (induk: Bidang Materi). Mapel, jenjang, kelas disalin dari induk.
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id, sort_order)
SELECT DISTINCT 'chapter', t.chapter_name, field.id, field.subject_id, field.education_level_id, field.grade_id, t.chapter_sort
FROM tmp_math_sd AS t
JOIN master_data_items AS subject ON subject.type = 'subject' AND subject.code = 'MTK'
JOIN master_data_items AS grade   ON grade.type = 'grade' AND grade.code = t.grade_code
JOIN master_data_items AS field   ON field.type = 'subject_field' AND field.subject_id = subject.id
                                 AND field.grade_id = grade.id AND field.name = t.field_name
ON CONFLICT DO NOTHING;


-- ----------------------------------------------------------------------------
-- 4. Sub Bab (induk: Bab)
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id, sort_order)
SELECT DISTINCT 'subchapter', t.subchapter_name, chapter.id, chapter.subject_id, chapter.education_level_id, chapter.grade_id, t.subchapter_sort
FROM tmp_math_sd AS t
JOIN master_data_items AS subject ON subject.type = 'subject' AND subject.code = 'MTK'
JOIN master_data_items AS grade   ON grade.type = 'grade' AND grade.code = t.grade_code
JOIN master_data_items AS field   ON field.type = 'subject_field' AND field.subject_id = subject.id
                                 AND field.grade_id = grade.id AND field.name = t.field_name
JOIN master_data_items AS chapter ON chapter.type = 'chapter' AND chapter.parent_id = field.id AND chapter.name = t.chapter_name
ON CONFLICT DO NOTHING;


-- ----------------------------------------------------------------------------
-- 5. Topik (induk: Sub Bab)
-- ----------------------------------------------------------------------------
INSERT INTO master_data_items (type, name, parent_id, subject_id, education_level_id, grade_id, sort_order)
SELECT 'topic', t.topic_name, subchapter.id, subchapter.subject_id, subchapter.education_level_id, subchapter.grade_id, t.topic_sort
FROM tmp_math_sd AS t
JOIN master_data_items AS subject    ON subject.type = 'subject' AND subject.code = 'MTK'
JOIN master_data_items AS grade      ON grade.type = 'grade' AND grade.code = t.grade_code
JOIN master_data_items AS field      ON field.type = 'subject_field' AND field.subject_id = subject.id
                                    AND field.grade_id = grade.id AND field.name = t.field_name
JOIN master_data_items AS chapter    ON chapter.type = 'chapter' AND chapter.parent_id = field.id AND chapter.name = t.chapter_name
JOIN master_data_items AS subchapter ON subchapter.type = 'subchapter' AND subchapter.parent_id = chapter.id AND subchapter.name = t.subchapter_name
ON CONFLICT DO NOTHING;

COMMIT;


-- ----------------------------------------------------------------------------
-- Cek hasilnya (jalankan terpisah di DBeaver):
--
-- 1) Daftar bab Matematika Kelas 4: cukup 1 tabel, tanpa JOIN (pakai kolom bantu).
-- SELECT c.name
-- FROM master_data_items c
-- WHERE c.type = 'chapter' AND c.deleted_at IS NULL
--   AND c.subject_id = (SELECT id FROM master_data_items WHERE type = 'subject' AND code = 'MTK')
--   AND c.grade_id   = (SELECT id FROM master_data_items WHERE type = 'grade'   AND code = 'GRADE_4')
-- ORDER BY c.sort_order;
--
-- 2) Pohon lengkap Matematika SD per kelas.
-- SELECT g.name AS kelas, f.name AS bidang, c.name AS bab, sc.name AS sub_bab, t.name AS topik
-- FROM master_data_items t
-- JOIN master_data_items sc ON sc.id = t.parent_id
-- JOIN master_data_items c  ON c.id  = sc.parent_id
-- JOIN master_data_items f  ON f.id  = c.parent_id
-- JOIN master_data_items g  ON g.id  = t.grade_id
-- WHERE t.type = 'topic'
--   AND t.subject_id = (SELECT id FROM master_data_items WHERE type = 'subject' AND code = 'MTK')
-- ORDER BY g.sort_order, f.sort_order, c.sort_order, sc.sort_order, t.sort_order;
-- ----------------------------------------------------------------------------
