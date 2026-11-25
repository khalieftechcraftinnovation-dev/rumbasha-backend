-- ============================================================================
-- Tanggal bergabung & tanggal berakhir akun. (BRD FR-STU-024, FR-STU-025)
--
--   join_date  wajib. Default hari ini (WIB) supaya data lama / seed tetap terisi; API tambah,
--              ubah, dan import user selalu mengisinya.
--   end_date   opsional. Setelah tanggal ini lewat (WIB), akun TIDAK BISA LOGIN lagi
--              (dianggap berakhir). NULL = tanpa batas. Harus >= join_date.
-- ============================================================================

ALTER TABLE users
    ADD COLUMN join_date DATE NOT NULL DEFAULT ((NOW() AT TIME ZONE 'Asia/Jakarta')::date),
    ADD COLUMN end_date  DATE;

-- Akun lama: tanggal bergabung = tanggal akun dibuat (WIB).
UPDATE users SET join_date = (created_at AT TIME ZONE 'Asia/Jakarta')::date;

ALTER TABLE users ADD CONSTRAINT users_end_after_join CHECK (end_date IS NULL OR end_date >= join_date);

COMMENT ON COLUMN users.join_date IS 'Tanggal bergabung (WIB)';
COMMENT ON COLUMN users.end_date  IS 'Tanggal berakhir (WIB, inklusif). Lewat dari tanggal ini akun tidak bisa login. NULL = tanpa batas';
