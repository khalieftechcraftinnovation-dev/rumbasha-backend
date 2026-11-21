-- ============================================================================
-- Data tambahan khusus siswa. (BRD 3.2, 3.14, 8.2)
-- Username, nama, HP orang tua, asal sekolah, jenjang, kelas, cabang, dan status
-- siswa ada di tabel users.
-- ============================================================================

-- Nomor urut untuk kode siswa otomatis: STU000001, STU000002, ...
CREATE SEQUENCE student_code_seq;

-- Satu baris per siswa; user_id sekaligus menjadi primary key (relasi 1:1 dengan users).
CREATE TABLE students (
    user_id       UUID         PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    student_code  VARCHAR(20)  NOT NULL UNIQUE
                  DEFAULT ('STU' || LPAD(nextval('student_code_seq')::TEXT, 6, '0')),
    date_of_birth DATE,
    target_school VARCHAR(150),  -- contoh: "SMAN 5 Surabaya" (FR-STU-011)
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Sequence ikut terhapus kalau kolomnya dihapus.
ALTER SEQUENCE student_code_seq OWNED BY students.student_code;

COMMENT ON TABLE  students               IS 'Data tambahan siswa (1 baris per user ber-role student)';
COMMENT ON COLUMN students.student_code  IS 'Kode siswa otomatis (STU000001). Saat insert, jangan diisi';
COMMENT ON COLUMN students.target_school IS 'Target sekolah tujuan, contoh SMAN 5 Surabaya';
