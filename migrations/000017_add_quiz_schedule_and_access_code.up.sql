-- ============================================================================
-- Quiz: waktu mulai (WIB) & kode akses. (BRD FR-TRY-026, FR-TRY-030, FR-TRY-031)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Waktu mulai. Disimpan TIMESTAMPTZ (titik waktu absolut); API membaca & menampilkan dalam WIB.
--    Quiz yang sudah ada (data uji coba) diisi waktu dibuatnya.
-- ----------------------------------------------------------------------------
ALTER TABLE quizzes ADD COLUMN start_at TIMESTAMPTZ;
UPDATE quizzes SET start_at = created_at;
ALTER TABLE quizzes ALTER COLUMN start_at SET NOT NULL;

COMMENT ON COLUMN quizzes.start_at IS 'Waktu mulai quiz (diinput & ditampilkan dalam WIB). Selesai = start_at + duration_minutes';


-- ----------------------------------------------------------------------------
-- 2. Kode akses: 6 karakter A-Z / 0-9, wajib untuk Uji Kom & Tryout.
--    Dibuat acak saat quiz dibuat, lalu diganti setiap jam oleh scheduler
--    (hanya quiz yang belum selesai).
-- ----------------------------------------------------------------------------
ALTER TABLE quizzes
    ADD COLUMN access_code            VARCHAR(6),
    ADD COLUMN access_code_updated_at TIMESTAMPTZ;

-- Quiz yang sudah ada: kode sementara (heksadesimal); scheduler akan menggantinya.
UPDATE quizzes
SET access_code = upper(substr(md5(id::text || random()::text), 1, 6)),
    access_code_updated_at = NOW()
WHERE type IN ('uji_kom', 'tryout');

ALTER TABLE quizzes ADD CONSTRAINT quizzes_access_code_format
    CHECK (access_code IS NULL OR access_code ~ '^[A-Z0-9]{6}$');
ALTER TABLE quizzes ADD CONSTRAINT quizzes_access_code_required
    CHECK (type NOT IN ('uji_kom', 'tryout') OR access_code IS NOT NULL);

-- Siswa masuk quiz dengan kode → kode harus unik di antara quiz yang belum dihapus.
-- Index ini juga dipakai untuk mencari quiz berdasarkan kode.
CREATE UNIQUE INDEX quizzes_access_code_unique
    ON quizzes (access_code)
    WHERE deleted_at IS NULL AND access_code IS NOT NULL;

-- Scheduler: quiz berkode yang belum selesai. Karena durasi maks 600 menit, quiz yang belum
-- selesai pasti start_at > NOW() - 600 menit, jadi index start_at cukup untuk mempersempit.
CREATE INDEX quizzes_access_code_start_at_idx
    ON quizzes (start_at)
    WHERE deleted_at IS NULL AND access_code IS NOT NULL;

COMMENT ON COLUMN quizzes.access_code            IS 'Kode akses 6 karakter (A-Z, 0-9) untuk Uji Kom & Tryout, diganti setiap jam selama quiz belum selesai';
COMMENT ON COLUMN quizzes.access_code_updated_at IS 'Terakhir kali kode akses dibuat/diganti';
