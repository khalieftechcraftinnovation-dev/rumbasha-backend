-- ============================================================================
-- Akun user dan sesi login. (BRD 3.1, 8.1)
-- ============================================================================

-- Semua akun: super admin, admin, tutor, dan siswa.
-- Login memakai username + password. Kolom lain adalah informasi detail.
-- Data tambahan khusus siswa ada di tabel students.
CREATE TABLE users (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    role               VARCHAR(20)  NOT NULL CHECK (role IN ('super_admin', 'admin', 'tutor', 'student')),
    branch_id          UUID         REFERENCES branches (id),

    -- Akun login
    username           VARCHAR(30)  NOT NULL,
    password_hash      VARCHAR(255) NOT NULL,  -- hasil bcrypt; password asli TIDAK pernah disimpan

    -- Informasi detail
    real_name          VARCHAR(150) NOT NULL,
    email              VARCHAR(255),
    phone              VARCHAR(20),
    parent_phone       VARCHAR(20),
    school_origin      VARCHAR(150),           -- asal sekolah
    education_level_id UUID         REFERENCES master_data_items (id),  -- jenjang
    grade_id           UUID         REFERENCES master_data_items (id),  -- kelas

    status             VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    last_login_at      TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,

    -- Username unik selamanya, termasuk milik user yang sudah dihapus, supaya
    -- riwayat lama (nilai, peringkat) tidak tertukar dengan user baru.
    -- Constraint UNIQUE otomatis membuat index, jadi pencarian per username cepat.
    CONSTRAINT users_username_unique UNIQUE (username),

    -- Username: 3-30 karakter, huruf kecil, angka, titik, atau garis bawah.
    -- Huruf kecil semua, supaya WHERE username = ? langsung cocok tanpa LOWER().
    CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9._]{3,30}$')
);

-- Untuk filter daftar user per role, cabang, jenjang, dan kelas (FR-STU-012, FR-STU-013).
CREATE INDEX users_role_idx               ON users (role);
CREATE INDEX users_branch_id_idx          ON users (branch_id);
CREATE INDEX users_education_level_id_idx ON users (education_level_id);
CREATE INDEX users_grade_id_idx           ON users (grade_id);

COMMENT ON TABLE  users                    IS 'Semua akun: super admin, admin, tutor, siswa';
COMMENT ON COLUMN users.role               IS 'super_admin, admin, tutor, atau student. Hak akses per role diatur di kode (middleware)';
COMMENT ON COLUMN users.branch_id          IS 'NULL = semua cabang (Super Admin). Pembatasan per cabang menunggu BD-025';
COMMENT ON COLUMN users.username           IS 'Untuk login. Unik, huruf kecil, 3-30 karakter (a-z 0-9 . _)';
COMMENT ON COLUMN users.password_hash      IS 'Hash bcrypt, bukan password asli. Reset password dilakukan oleh admin';
COMMENT ON COLUMN users.real_name          IS 'Nama asli / nama lengkap';
COMMENT ON COLUMN users.phone              IS 'Simpan dalam satu format yang sama, misalnya 081234567890';
COMMENT ON COLUMN users.parent_phone       IS 'HP orang tua/wali (siswa). Tampilkan tersamar: 0812-xxxx-2210';
COMMENT ON COLUMN users.school_origin      IS 'Asal sekolah (siswa)';
COMMENT ON COLUMN users.education_level_id IS 'Jenjang (siswa): master_data_items dengan type education_level';
COMMENT ON COLUMN users.grade_id           IS 'Kelas (siswa): master_data_items dengan type grade';
COMMENT ON COLUMN users.status             IS 'active atau inactive (dinonaktifkan admin)';
COMMENT ON COLUMN users.deleted_at         IS 'Terisi = sudah dihapus (soft delete). Arti hapus menunggu BD-026';


-- Refresh token = sesi login. (FR-AUTH-007)
-- Semua sesi berakhir setiap tanggal 1 jam 00:00 WIB, jadi semua user wajib
-- login ulang tiap awal bulan (berkaitan dengan paket offline yang dibayar bulanan).
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 dari token; token asli TIDAK disimpan
    user_agent TEXT,                         -- browser/perangkat yang dipakai login
    expires_at TIMESTAMPTZ NOT NULL,         -- tanggal 1 bulan berikutnya, 00:00 WIB
    revoked_at TIMESTAMPTZ,                  -- terisi saat logout atau token diganti (refresh)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

COMMENT ON TABLE  refresh_tokens            IS 'Sesi login (refresh token). Berakhir tiap tanggal 1, 00:00 WIB';
COMMENT ON COLUMN refresh_tokens.expires_at IS 'Tanggal 1 bulan berikutnya, 00:00 WIB';
COMMENT ON COLUMN refresh_tokens.revoked_at IS 'Terisi saat logout atau saat token ditukar (refresh)';
