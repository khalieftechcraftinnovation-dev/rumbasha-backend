-- ============================================================================
-- Akun user, role, permission, dan token login. (BRD 3.1, 8.1)
-- ============================================================================

-- Role user. Satu user punya tepat SATU role: di desain menu User, jumlah per tab
-- (Siswa 47 + Tutor 4 + Admin 3 + Super Admin 1) sama dengan tab Semua (55).
CREATE TABLE roles (
    code       VARCHAR(20) PRIMARY KEY,  -- dipakai di kode program, jangan diubah
    name       VARCHAR(50) NOT NULL,     -- nama yang ditampilkan ke user
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Role adalah data tetap yang dibutuhkan aplikasi, jadi diisi langsung di migration.
INSERT INTO roles (code, name) VALUES
    ('super_admin', 'Super Admin'),
    ('admin',       'Admin'),
    ('tutor',       'Tutor'),
    ('student',     'Siswa');

COMMENT ON TABLE roles IS 'Role user: super_admin, admin, tutor, student';


-- Permission = izin melakukan aksi tertentu, misalnya "users.create". (FR-AUTH-004)
-- Daftarnya diisi nanti saat fitur auth dibuat.
CREATE TABLE permissions (
    code        VARCHAR(100) PRIMARY KEY,  -- contoh: users.create, users.import
    description VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Permission apa saja yang dimiliki setiap role.
CREATE TABLE role_permissions (
    role_code       VARCHAR(20)  NOT NULL REFERENCES roles (code) ON DELETE CASCADE,
    permission_code VARCHAR(100) NOT NULL REFERENCES permissions (code) ON DELETE CASCADE,
    PRIMARY KEY (role_code, permission_code)
);

COMMENT ON TABLE permissions      IS 'Daftar izin aksi, contoh: users.create';
COMMENT ON TABLE role_permissions IS 'Izin yang dimiliki setiap role';


-- Semua akun: super admin, admin, tutor, dan siswa.
-- Login memakai username + password. Kolom lain adalah informasi detail.
-- Data khusus siswa lainnya ada di tabel students, data khusus tutor di tabel tutors.
CREATE TABLE users (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    role_code          VARCHAR(20)  NOT NULL REFERENCES roles (code),
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
CREATE INDEX users_role_code_idx          ON users (role_code);
CREATE INDEX users_branch_id_idx          ON users (branch_id);
CREATE INDEX users_education_level_id_idx ON users (education_level_id);
CREATE INDEX users_grade_id_idx           ON users (grade_id);

COMMENT ON TABLE  users                    IS 'Semua akun: super admin, admin, tutor, siswa';
COMMENT ON COLUMN users.branch_id          IS 'NULL = semua cabang (Super Admin). Pembatasan per cabang menunggu BD-025';
COMMENT ON COLUMN users.username           IS 'Untuk login. Unik, huruf kecil, 3-30 karakter (a-z 0-9 . _)';
COMMENT ON COLUMN users.password_hash      IS 'Hash bcrypt, bukan password asli';
COMMENT ON COLUMN users.real_name          IS 'Nama asli / nama lengkap';
COMMENT ON COLUMN users.phone              IS 'Simpan dalam satu format yang sama, misalnya 081234567890';
COMMENT ON COLUMN users.parent_phone       IS 'HP orang tua/wali (siswa). Tampilkan tersamar: 0812-xxxx-2210';
COMMENT ON COLUMN users.school_origin      IS 'Asal sekolah (siswa)';
COMMENT ON COLUMN users.education_level_id IS 'Jenjang (siswa): master_data_items dengan type education_level';
COMMENT ON COLUMN users.grade_id           IS 'Kelas (siswa): master_data_items dengan type grade';
COMMENT ON COLUMN users.status             IS 'active atau inactive (dinonaktifkan admin)';
COMMENT ON COLUMN users.deleted_at         IS 'Terisi = sudah dihapus (soft delete). Arti hapus menunggu BD-026';


-- Refresh token untuk memperpanjang sesi login tanpa login ulang. (FR-AUTH-007, FR-AUTH-010)
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 dari token; token asli TIDAK disimpan
    remember_me BOOLEAN     NOT NULL DEFAULT FALSE,
    user_agent  TEXT,                         -- browser/perangkat yang dipakai login
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,                  -- terisi saat logout atau token diganti
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

COMMENT ON TABLE  refresh_tokens             IS 'Sesi login (refresh token)';
COMMENT ON COLUMN refresh_tokens.remember_me IS 'Opsi "Ingat saya": masa berlaku lebih panjang';


-- Token sekali pakai untuk reset kata sandi. (FR-AUTH-006)
-- Cara token dikirim ke user (email/WhatsApp/lewat admin) belum ditentukan.
CREATE TABLE password_reset_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,  -- terisi setelah dipakai, supaya tidak bisa dipakai lagi
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

COMMENT ON TABLE password_reset_tokens IS 'Token sekali pakai untuk reset kata sandi';
