-- ============================================================================
-- Profil tutor dan siswa bimbingannya. (BRD 3.3, 8.2)
-- Username, nama, cabang, dan status tutor ada di tabel users.
-- ============================================================================

-- Satu baris per tutor. Saat ini belum ada data khusus tutor di BRD; kolom seperti
-- mapel keahlian bisa ditambahkan nanti lewat migration baru.
CREATE TABLE tutors (
    user_id    UUID        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tutors IS 'Profil tutor (1 baris per user ber-role tutor)';


-- Siswa yang menjadi tanggung jawab seorang tutor. (FR-TUT-005, FR-TUT-006)
-- Dibuat many-to-many supaya tetap aman kalau 1 siswa boleh punya >1 tutor (BD-013).
CREATE TABLE tutor_students (
    tutor_id    UUID        NOT NULL REFERENCES tutors (user_id) ON DELETE CASCADE,
    student_id  UUID        NOT NULL REFERENCES students (user_id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tutor_id, student_id)
);

-- Untuk mencari tutor dari seorang siswa (primary key sudah mencakup pencarian per tutor).
CREATE INDEX tutor_students_student_id_idx ON tutor_students (student_id);

COMMENT ON TABLE tutor_students IS 'Siswa bimbingan setiap tutor';
