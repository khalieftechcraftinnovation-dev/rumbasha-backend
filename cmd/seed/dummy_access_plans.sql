-- ============================================================================
-- DATA DUMMY paket akses Basic / Pro / Ultra + akun siswa untuk mencoba skenario akses modul.
-- JANGAN dijalankan di production.
--
-- Dijalankan otomatis oleh "make seed" (setelah dummy_users.sql). Aman dijalankan berkali-kali.
--
-- Paket (berlaku semua jenjang; akses modul TETAP dibatasi jenjang akun siswa):
--   Basic  → modul tingkat "Ringkasan Materi" saja              + Daily Quiz
--   Pro    → + "Latihan Soal"                                    + Daily Quiz, Uji Kom
--   Ultra  → + "Pembahasan Lengkap" (semua modul di jenjangnya)   + semua fitur
--   (modul mana yang ada di paket mana diatur di learning_module_plans, lihat seed modul)
--
-- Akun skenario (password: password123), paket aktif 10 hari lalu s/d 20 hari lagi:
--   siswa.basic      SD Kelas 5   - Basic
--   siswa.pro        SD Kelas 5   - Pro
--   siswa.ultra      SD Kelas 5   - Ultra
--   siswa.ultra.smp  SMP Kelas 7  - Ultra  (hanya melihat modul SMP)
-- ============================================================================

BEGIN;

INSERT INTO subscription_plans
    (name, description, highlights, education_level_id, monthly_coin, is_featured,
     price_monthly, price_semester, price_yearly, strike_price_monthly,
     has_module, has_daily_quiz, has_uji_kom, has_tryout, has_rumath, sort_order)
SELECT v.name, v.description, v.highlights, NULL, v.monthly_coin, v.is_featured,
       v.price_monthly, v.price_semester, v.price_yearly, v.strike_price_monthly,
       TRUE, TRUE, v.has_uji_kom, v.has_tryout, v.has_rumath, v.sort_order
FROM (VALUES
    ('Basic', 'Belajar mandiri dengan ringkasan materi', E'Modul ringkasan materi\nDaily Quiz',
     0, FALSE, 99000::BIGINT, 540000::BIGINT, NULL::BIGINT, NULL::BIGINT, FALSE, FALSE, FALSE, 10),
    ('Pro', 'Ringkasan materi + latihan soal', E'Modul ringkasan + latihan soal\nDaily Quiz & Uji Kom\n50 coin tiap bulan',
     50, TRUE, 199000, 1080000, 2040000, 249000, TRUE, FALSE, FALSE, 11),
    ('Ultra', 'Semua modul + semua fitur', E'Semua modul termasuk pembahasan lengkap\nSemua fitur + RUMATH\n150 coin tiap bulan',
     150, FALSE, 349000, 1890000, 3600000, 399000, TRUE, TRUE, TRUE, 12)
) AS v (name, description, highlights, monthly_coin, is_featured, price_monthly, price_semester, price_yearly,
        strike_price_monthly, has_uji_kom, has_tryout, has_rumath, sort_order)
WHERE NOT EXISTS (SELECT 1 FROM subscription_plans p WHERE LOWER(p.name) = LOWER(v.name) AND p.deleted_at IS NULL);

-- Akun skenario.
INSERT INTO users (role, username, password_hash, real_name, branch_id, parent_phone, school_origin,
                   education_level_id, grade_id, status)
SELECT 'student', v.username,
       '$2a$10$CLjXDk4dqnL0gJHw5b6iW.FTDmBleeSo/Vy9soguhl6AF3LYB6Pc2',  -- password123
       v.real_name, (SELECT id FROM branches WHERE name = 'Surabaya Timur'), v.parent_phone, v.school,
       g.parent_id, g.id, 'active'
FROM (VALUES
    ('siswa.basic',     'Bima Basic',    '081377770001', 'SDN 1 Surabaya',  'GRADE_5'),
    ('siswa.pro',       'Putri Pro',     '081377770002', 'SDN 2 Surabaya',  'GRADE_5'),
    ('siswa.ultra',     'Umar Ultra',    '081377770003', 'SDN 3 Surabaya',  'GRADE_5'),
    ('siswa.ultra.smp', 'Sinta Ultra',   '081377770004', 'SMPN 1 Surabaya', 'GRADE_7')
) AS v (username, real_name, parent_phone, school, grade_code)
JOIN master_data_items g ON g.type = 'grade' AND g.code = v.grade_code
ON CONFLICT (username) DO NOTHING;

INSERT INTO students (user_id, target_school)
SELECT u.id, 'SMAN 5 Surabaya'
FROM users u
WHERE u.username IN ('siswa.basic', 'siswa.pro', 'siswa.ultra', 'siswa.ultra.smp')
  AND NOT EXISTS (SELECT 1 FROM students s WHERE s.user_id = u.id);

-- Paket aktif (hanya kalau belum ada paket yang aktif hari ini; seed ulang bulan depan → paket baru).
INSERT INTO student_subscriptions (student_id, plan_id, channel, period, price_paid, start_date, end_date, created_by)
SELECT u.id, p.id, 'online', 'monthly', p.price_monthly,
       (NOW() AT TIME ZONE 'Asia/Jakarta')::date - 10, (NOW() AT TIME ZONE 'Asia/Jakarta')::date + 20,
       (SELECT id FROM users WHERE username = 'admin.sbytimur')
FROM (VALUES
    ('siswa.basic', 'Basic'), ('siswa.pro', 'Pro'), ('siswa.ultra', 'Ultra'), ('siswa.ultra.smp', 'Ultra')
) AS v (username, plan_name)
JOIN users u              ON u.username = v.username
JOIN subscription_plans p ON LOWER(p.name) = LOWER(v.plan_name) AND p.deleted_at IS NULL
WHERE NOT EXISTS (
    SELECT 1 FROM student_subscriptions ss
    WHERE ss.student_id = u.id AND ss.cancelled_at IS NULL
      AND (NOW() AT TIME ZONE 'Asia/Jakarta')::date BETWEEN ss.start_date AND ss.end_date
);

COMMIT;
