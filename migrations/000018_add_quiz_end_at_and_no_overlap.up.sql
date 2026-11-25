-- ============================================================================
-- Quiz: waktu selesai event (bisa lebih dari 1 hari) & larangan jadwal beririsan.
-- (BRD FR-TRY-024, FR-TRY-031, FR-TRY-032)
--
--   start_at .. end_at  = jendela event (contoh 1 Des 08:00 s/d 3 Des 17:00 WIB)
--   duration_minutes    = lama mengerjakan 1 kali (contoh 60 menit)
-- ============================================================================

-- btree_gist dibutuhkan exclusion constraint di bawah (gabungan "=" untuk UUID dan "&&" untuk range).
-- Extension "trusted" (PostgreSQL 13+): cukup pemilik database.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ----------------------------------------------------------------------------
-- 1. Waktu selesai event. Quiz yang sudah ada: selesai = mulai + durasi (aturan lama).
-- ----------------------------------------------------------------------------
ALTER TABLE quizzes ADD COLUMN end_at TIMESTAMPTZ;
UPDATE quizzes SET end_at = start_at + make_interval(mins => duration_minutes);
ALTER TABLE quizzes ALTER COLUMN end_at SET NOT NULL;

-- Event minimal sepanjang durasi pengerjaan (siswa yang mulai tepat di awal bisa selesai).
ALTER TABLE quizzes ADD CONSTRAINT quizzes_end_after_start
    CHECK (end_at - start_at >= duration_minutes * INTERVAL '1 minute');

COMMENT ON COLUMN quizzes.start_at         IS 'Mulai event quiz (diinput & ditampilkan dalam WIB)';
COMMENT ON COLUMN quizzes.end_at           IS 'Selesai event quiz (WIB), boleh beda hari dengan start_at';
COMMENT ON COLUMN quizzes.duration_minutes IS 'Lama mengerjakan 1 kali (menit), di dalam jendela start_at..end_at';


-- ----------------------------------------------------------------------------
-- 2. Tidak boleh ada 2 quiz (yang belum dihapus) dengan mapel & kelas sama yang jadwalnya beririsan.
--    Range '[)': quiz yang selesai 10:00 boleh disusul quiz yang mulai 10:00.
--    Constraint ini sekaligus index GiST untuk mencari quiz yang bentrok (service memakainya
--    untuk pesan error yang jelas; constraint menjaga dari 2 request bersamaan).
-- ----------------------------------------------------------------------------
DO $$
DECLARE
    conflict TEXT;
BEGIN
    SELECT string_agg(format('"%s" & "%s"', a.name, b.name), ', ')
    INTO conflict
    FROM quizzes a
    JOIN quizzes b ON a.id < b.id
                  AND a.subject_id = b.subject_id
                  AND a.grade_id = b.grade_id
                  AND tstzrange(a.start_at, a.end_at, '[)') && tstzrange(b.start_at, b.end_at, '[)')
    WHERE a.deleted_at IS NULL AND b.deleted_at IS NULL;

    IF conflict IS NOT NULL THEN
        RAISE EXCEPTION 'Ada quiz dengan mapel & kelas sama yang jadwalnya beririsan: %. Ubah jadwal atau hapus salah satunya dulu.', conflict;
    END IF;
END $$;

ALTER TABLE quizzes ADD CONSTRAINT quizzes_no_overlap
    EXCLUDE USING gist (
        subject_id WITH =,
        grade_id   WITH =,
        tstzrange(start_at, end_at, '[)') WITH &&
    ) WHERE (deleted_at IS NULL);


-- ----------------------------------------------------------------------------
-- 3. Index scheduler & filter jadwal: "belum selesai" sekarang cukup end_at > NOW().
-- ----------------------------------------------------------------------------
DROP INDEX IF EXISTS quizzes_access_code_start_at_idx;
CREATE INDEX quizzes_access_code_end_at_idx
    ON quizzes (end_at)
    WHERE deleted_at IS NULL AND access_code IS NOT NULL;
