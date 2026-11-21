-- ============================================================================
-- DATA DUMMY untuk development & demo. JANGAN dijalankan di production.
--
-- Cara pakai: make seed   (atau buka file ini di DBeaver lalu Alt+X)
-- Jalankan sekali di database yang masih kosong.
--
-- Semua akun memakai password: password123
-- (hash bcrypt di bawah adalah hasil dari "password123")
--
-- Tanggal paket dihitung dari hari ini, jadi datanya selalu relevan kapan pun dijalankan.
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- Cabang
-- ----------------------------------------------------------------------------
INSERT INTO branches (name) VALUES
    ('Surabaya Timur'),
    ('Surabaya Barat'),
    ('Sidoarjo');


-- ----------------------------------------------------------------------------
-- Akun: 1 per role, ditambah beberapa siswa dengan kondisi paket berbeda
-- ----------------------------------------------------------------------------
INSERT INTO users (role, username, password_hash, real_name, branch_id, phone, parent_phone, school_origin, education_level_id, grade_id, status)
SELECT v.role, v.username,
       '$2a$10$CLjXDk4dqnL0gJHw5b6iW.FTDmBleeSo/Vy9soguhl6AF3LYB6Pc2',  -- password123
       v.real_name,
       (SELECT id FROM branches WHERE name = v.branch),
       v.phone, v.parent_phone, v.school_origin,
       (SELECT id FROM master_data_items WHERE type = 'education_level' AND code = v.level_code),
       (SELECT id FROM master_data_items WHERE type = 'grade' AND code = v.grade_code),
       v.status
FROM (VALUES
    -- role,          username,         nama asli,              cabang,           HP,             HP ortu,        asal sekolah,        jenjang,  kelas,       status
    ('super_admin', 'superadmin',     'Super Admin Rumbasha', NULL,             '081200000001', NULL,           NULL,                NULL,     NULL,        'active'),
    ('admin',       'admin.sbytimur', 'Admin Surabaya Timur', 'Surabaya Timur', '081200000002', NULL,           NULL,                NULL,     NULL,        'active'),
    ('tutor',       'tutor.rina',     'Rina Wulandari',       'Surabaya Timur', '081200000003', NULL,           NULL,                NULL,     NULL,        'active'),
    ('student',     'siswa.budi',     'Budi Santoso',         'Surabaya Timur', NULL,           '081234562210', 'SMPN 1 Surabaya',   'SMP',    'GRADE_7',   'active'),
    ('student',     'siswa.citra',    'Citra Lestari',        'Surabaya Barat', NULL,           '081298761234', 'SDN Kertajaya 1',   'SD',     'GRADE_5',   'active'),
    ('student',     'siswa.dimas',    'Dimas Pratama',        'Sidoarjo',       '081311112222', '081333334444', 'SMAN 1 Sidoarjo',   'SMA',    'GRADE_10',  'active'),
    ('student',     'siswa.eka',      'Eka Putri',            'Sidoarjo',       NULL,           '081355556666', 'TK Pelangi',        'PRA_SD', 'NOL_BESAR', 'inactive')
) AS v (role, username, real_name, branch, phone, parent_phone, school_origin, level_code, grade_code, status);

-- Data tambahan siswa (student_code dibuat otomatis: STU000001, ...)
INSERT INTO students (user_id, date_of_birth, target_school, coin_balance)
SELECT u.id, v.date_of_birth::date, v.target_school, v.coin_balance
FROM (VALUES
    ('siswa.budi',  '2014-03-12', 'SMAN 5 Surabaya',   100),
    ('siswa.citra', '2016-08-25', 'SMPN 6 Surabaya',   0),
    ('siswa.dimas', '2011-01-30', 'Universitas Airlangga', 0),
    ('siswa.eka',   '2020-05-17', NULL,                0)
) AS v (username, date_of_birth, target_school, coin_balance)
JOIN users u ON u.username = v.username;


-- ----------------------------------------------------------------------------
-- Paket akses (contoh dari desain BRD 3.8)
-- ----------------------------------------------------------------------------
INSERT INTO subscription_plans
    (name, description, highlights, education_level_id, monthly_coin, is_featured,
     price_monthly, price_semester, price_yearly, strike_price_monthly,
     has_module, has_daily_quiz, has_uji_kom, has_tryout, has_rumath, sort_order)
SELECT v.name, v.description, v.highlights,
       (SELECT id FROM master_data_items WHERE type = 'education_level' AND code = v.level_code),
       v.monthly_coin, v.is_featured,
       v.price_monthly, v.price_semester, v.price_yearly, v.strike_price_monthly,
       v.has_module, v.has_daily_quiz, v.has_uji_kom, v.has_tryout, v.has_rumath, v.sort_order
FROM (VALUES
    ('Reguler SD', 'Belajar rutin siswa SD', E'2 pertemuan per minggu\nModul dan Daily Quiz',
     'SD', 0, FALSE, 350000::BIGINT, NULL::BIGINT, NULL::BIGINT, NULL::BIGINT,
     TRUE, TRUE, FALSE, FALSE, FALSE, 1),
    ('Reguler SMP', 'Belajar rutin kelas 7-9', E'2 pertemuan per minggu\nModul, Daily Quiz, Uji Kom\n100 coin tiap bulan',
     'SMP', 100, FALSE, 450000, 2565000, 4320000, 600000,
     TRUE, TRUE, TRUE, FALSE, FALSE, 2),
    ('Intensif Masuk SMA', 'Persiapan TKA dan seleksi SMA', E'4 pertemuan per minggu\nSemua fitur + RUMATH',
     'SMP', 0, TRUE, 750000, NULL, 7200000, 950000,
     TRUE, TRUE, TRUE, TRUE, TRUE, 3)
) AS v (name, description, highlights, level_code, monthly_coin, is_featured,
        price_monthly, price_semester, price_yearly, strike_price_monthly,
        has_module, has_daily_quiz, has_uji_kom, has_tryout, has_rumath, sort_order);


-- ----------------------------------------------------------------------------
-- Paket milik siswa: 3 kondisi berbeda untuk dicoba di GET /auth/me
-- ----------------------------------------------------------------------------
-- budi  → OFFLINE, aktif sebulan penuh bulan ini (bayar manual di bimbel)
-- citra → ONLINE, aktif (mulai 10 hari lalu, berakhir 20 hari lagi)
-- dimas → ONLINE, sudah EXPIRED kemarin → tidak punya paket aktif
INSERT INTO student_subscriptions (student_id, plan_id, channel, period, price_paid, start_date, end_date, created_by)
SELECT u.id, p.id, v.channel, 'monthly', p.price_monthly, v.start_date, v.end_date,
       (SELECT id FROM users WHERE username = 'admin.sbytimur')
FROM (VALUES
    ('siswa.budi',  'Reguler SMP',        'offline',
        date_trunc('month', NOW() AT TIME ZONE 'Asia/Jakarta')::date,
        (date_trunc('month', NOW() AT TIME ZONE 'Asia/Jakarta') + INTERVAL '1 month - 1 day')::date),
    ('siswa.citra', 'Reguler SD',         'online',
        (NOW() AT TIME ZONE 'Asia/Jakarta')::date - 10,
        (NOW() AT TIME ZONE 'Asia/Jakarta')::date + 20),
    ('siswa.dimas', 'Intensif Masuk SMA', 'online',
        (NOW() AT TIME ZONE 'Asia/Jakarta')::date - 31,
        (NOW() AT TIME ZONE 'Asia/Jakarta')::date - 1)
) AS v (username, plan_name, channel, start_date, end_date)
JOIN users u              ON u.username = v.username
JOIN subscription_plans p ON p.name = v.plan_name;

-- Akses khusus: admin membuka RUMATH untuk citra walau paketnya Reguler SD.
INSERT INTO student_feature_overrides (student_id, feature, is_enabled, updated_by)
SELECT u.id, 'rumath', TRUE, (SELECT id FROM users WHERE username = 'admin.sbytimur')
FROM users u
WHERE u.username = 'siswa.citra';


-- ----------------------------------------------------------------------------
-- Coin
-- ----------------------------------------------------------------------------
INSERT INTO coin_packages (name, coin_amount, price, description, sort_order) VALUES
    ('100 coin',   100,  25000, 'Untuk 2 tryout', 1),
    ('300 coin',   300,  68000, 'Hemat 10%',      2),
    ('600 coin',   600, 128000, 'Hemat 15%',      3),
    ('1.200 coin', 1200, 240000, 'Hemat 20%',     4);

-- Riwayat coin budi: bonus 100 coin dari paket Reguler SMP (sesuai coin_balance = 100).
INSERT INTO coin_transactions (student_id, type, amount, balance_after, reference_type, reference_id, description)
SELECT ss.student_id, 'plan_grant', 100, 100, 'student_subscription', ss.id, 'Bonus coin bulanan Reguler SMP'
FROM student_subscriptions ss
JOIN users u ON u.id = ss.student_id
WHERE u.username = 'siswa.budi';

COMMIT;
