-- ============================================================================
-- DATA DUMMY TAMBAHAN: akun user. JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" SETELAH dummy_data.sql.
-- Aman dijalankan berkali-kali: username yang sudah ada dilewati (ON CONFLICT DO NOTHING).
--
-- Bersama dummy_data.sql, totalnya mengikuti contoh desain menu User (FR-STU-012):
--   Semua 55 = Siswa 47 + Tutor 4 + Admin 3 + Super Admin 1
--
-- Semua akun memakai password: password123
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- Admin & tutor tambahan (1 admin per cabang, tutor tersebar di semua cabang)
-- ----------------------------------------------------------------------------
INSERT INTO users (role, username, password_hash, real_name, branch_id, phone)
SELECT v.role, v.username,
       '$2a$10$CLjXDk4dqnL0gJHw5b6iW.FTDmBleeSo/Vy9soguhl6AF3LYB6Pc2',  -- password123
       v.real_name,
       (SELECT id FROM branches WHERE name = v.branch),
       v.phone
FROM (VALUES
    ('admin', 'admin.sbybarat', 'Admin Surabaya Barat', 'Surabaya Barat', '081200000004'),
    ('admin', 'admin.sidoarjo', 'Admin Sidoarjo',       'Sidoarjo',       '081200000005'),
    ('tutor', 'tutor.andi',     'Andi Firmansyah',      'Surabaya Barat', '081200000006'),
    ('tutor', 'tutor.sinta',    'Sinta Maharani',       'Sidoarjo',       '081200000007'),
    ('tutor', 'tutor.bayu',     'Bayu Prasetyo',        'Surabaya Timur', '081200000008')
) AS v (role, username, real_name, branch, phone)
ON CONFLICT (username) DO NOTHING;


-- ----------------------------------------------------------------------------
-- 43 siswa tambahan
--
-- Dibuat otomatis dari daftar nama di bawah. Untuk siswa ke-n:
--   nama     → nama depan ke-n + nama belakang bergiliran
--   username → siswa.<nama depan>, contoh siswa.adit
--   kelas    → bergiliran dari daftar kelas (lebih banyak SD & SMP);
--              jenjang diambil dari induk kelasnya, jadi pasti cocok
--   cabang   → bergiliran Surabaya Timur, Surabaya Barat, Sidoarjo
--   status   → setiap siswa ke-15 nonaktif
-- ----------------------------------------------------------------------------
WITH names AS (
    SELECT ARRAY[
        'Adit', 'Bagas', 'Cahya', 'Dewi', 'Fajar', 'Gita', 'Hana', 'Indra', 'Joko', 'Kirana',
        'Lukman', 'Maya', 'Nadia', 'Oki', 'Putra', 'Qori', 'Rizky', 'Sari', 'Teguh', 'Umi',
        'Vina', 'Wahyu', 'Yoga', 'Zahra', 'Agus', 'Bella', 'Dian', 'Farhan', 'Galih', 'Intan',
        'Lina', 'Nanda', 'Rafi', 'Salsa', 'Tari', 'Wulan', 'Yusuf', 'Aldi', 'Fitri', 'Hendra',
        'Kevin', 'Mega', 'Reza'
    ] AS first_names,
    ARRAY[  -- nama marga/keluarga yang netral untuk laki-laki maupun perempuan
        'Wijaya', 'Halim', 'Siregar', 'Nasution', 'Harahap', 'Tanjung',
        'Lubis', 'Sinaga', 'Hasibuan', 'Pane', 'Simatupang', 'Gunawan'
    ] AS last_names,
    ARRAY[
        'NOL_KECIL', 'NOL_BESAR',
        'GRADE_1', 'GRADE_2', 'GRADE_3', 'GRADE_4', 'GRADE_5', 'GRADE_6',
        'GRADE_7', 'GRADE_8', 'GRADE_9', 'GRADE_7', 'GRADE_8', 'GRADE_9',
        'GRADE_10', 'GRADE_11', 'GRADE_12'
    ] AS grade_codes,
    ARRAY['Surabaya Timur', 'Surabaya Barat', 'Sidoarjo'] AS branch_names
),
new_students AS (
    -- Satu baris per siswa, n = 1 sampai 43.
    -- Catatan: array di PostgreSQL dimulai dari indeks 1, jadi pakai ((n - 1) % panjang) + 1.
    SELECT n,
           first_names[n]                                                    AS first_name,
           last_names[((n - 1) % array_length(last_names, 1)) + 1]           AS last_name,
           grade_codes[((n - 1) % array_length(grade_codes, 1)) + 1]         AS grade_code,
           branch_names[((n - 1) % array_length(branch_names, 1)) + 1]       AS branch_name
    FROM names, generate_series(1, array_length(first_names, 1)) AS n
)
INSERT INTO users (role, username, password_hash, real_name, branch_id, parent_phone,
                   school_origin, education_level_id, grade_id, status)
SELECT 'student',
       'siswa.' || LOWER(s.first_name),
       '$2a$10$CLjXDk4dqnL0gJHw5b6iW.FTDmBleeSo/Vy9soguhl6AF3LYB6Pc2',  -- password123
       s.first_name || ' ' || s.last_name,
       b.id,
       '0813' || LPAD((10000000 + s.n * 7919)::TEXT, 8, '0'),            -- nomor HP ortu unik
       CASE level.code
           WHEN 'PRA_SD' THEN 'TK Tunas Bangsa ' || (s.n % 5 + 1)
           WHEN 'SD'     THEN 'SDN ' || (s.n % 10 + 1) || ' ' || split_part(s.branch_name, ' ', 1)
           WHEN 'SMP'    THEN 'SMPN ' || (s.n % 10 + 1) || ' ' || split_part(s.branch_name, ' ', 1)
           ELSE               'SMAN ' || (s.n % 10 + 1) || ' ' || split_part(s.branch_name, ' ', 1)
       END,
       level.id,
       grade.id,
       CASE WHEN s.n % 15 = 0 THEN 'inactive' ELSE 'active' END
FROM new_students s
JOIN branches b                  ON b.name = s.branch_name
JOIN master_data_items AS grade  ON grade.type = 'grade' AND grade.code = s.grade_code
JOIN master_data_items AS level  ON level.id = grade.parent_id
ON CONFLICT (username) DO NOTHING;


-- Data tambahan siswa untuk SEMUA user ber-role student yang belum punya baris di students.
-- Tanggal lahir diperkirakan dari kelasnya: umur ≈ kelas + 6 tahun (Nol Kecil 4, Nol Besar 5).
--
-- Sengaja memakai NOT EXISTS, bukan ON CONFLICT: kolom student_code diisi dari
-- sequence, dan ON CONFLICT tetap "memakai" nomor urut walau barisnya dilewati,
-- sehingga kode siswa jadi melompat setiap kali seed diulang.
-- ORDER BY membuat kode siswa urut berdasarkan kelas, lalu username.
INSERT INTO students (user_id, date_of_birth, target_school)
SELECT u.id,
       make_date(
           EXTRACT(YEAR FROM NOW())::INT - CASE grade.code
               WHEN 'NOL_KECIL' THEN 4
               WHEN 'NOL_BESAR' THEN 5
               ELSE REPLACE(grade.code, 'GRADE_', '')::INT + 6
           END,
           (ABS(hashtext(u.username)) % 12) + 1,   -- bulan "acak" tapi tetap sama tiap dijalankan
           (ABS(hashtext(u.username)) % 28) + 1),
       CASE level.code
           WHEN 'SD'  THEN 'SMPN 1 Surabaya'
           WHEN 'SMP' THEN 'SMAN 5 Surabaya'
           WHEN 'SMA' THEN 'Universitas Airlangga'
       END                                          -- PraSD: belum punya target sekolah
FROM users u
JOIN master_data_items AS grade ON grade.id = u.grade_id
JOIN master_data_items AS level ON level.id = grade.parent_id
WHERE u.role = 'student'
  AND NOT EXISTS (SELECT 1 FROM students s WHERE s.user_id = u.id)
ORDER BY grade.sort_order, u.username;

COMMIT;
