-- ============================================================================
-- Modul pembelajaran untuk siswa: list sesuai jenjang + paket akses, dan riwayat buka modul.
-- (BRD FR-LMS-031 s/d FR-LMS-034)
--
--   learning_module_opens  1 baris per siswa per modul: pertama & terakhir dibuka, berapa kali.
--                          Dipakai "Terakhir kamu buka" dan progres modul (FR-LMS-013, FR-LMS-016).
-- ============================================================================

CREATE TABLE learning_module_opens (
    student_id      UUID        NOT NULL REFERENCES users (id),
    module_id       UUID        NOT NULL REFERENCES learning_modules (id) ON DELETE CASCADE,
    first_opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_opened_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    open_count      INT         NOT NULL DEFAULT 1 CHECK (open_count > 0),

    PRIMARY KEY (student_id, module_id)
);

-- "Terakhir kamu buka": modul siswa urut terakhir dibuka.
CREATE INDEX learning_module_opens_recent_idx ON learning_module_opens (student_id, last_opened_at DESC);

-- List modul siswa: modul publish di 1 jenjang.
CREATE INDEX learning_modules_student_idx ON learning_modules (education_level_id, grade_id, subject_id)
    WHERE deleted_at IS NULL AND status = 'published';

COMMENT ON TABLE learning_module_opens IS 'Riwayat siswa membuka modul pembelajaran (pertama, terakhir, jumlah buka)';
