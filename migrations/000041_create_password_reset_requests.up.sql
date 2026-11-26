-- ============================================================================
-- Lupa kata sandi lewat admin & "Ingat saya". (BRD FR-AUTH-005, 006, 010, 011, 013)
--
--   password_reset_requests  permintaan "Lupa kata sandi" dari halaman login. Tidak ada email /
--                            OTP: admin melihat daftar permintaan, menghubungi user, lalu mereset
--                            kata sandinya. 1 user maksimal 1 permintaan yang masih menunggu
--                            (kirim lagi = permintaan yang sama diperbarui, jumlahnya bertambah).
--   refresh_tokens           + remember_me: sesi "Ingat saya" (30 hari, boleh melewati tanggal 1);
--                            tanpa centang tetap berakhir tanggal 1 bulan berikutnya 00:00 WIB.
-- ============================================================================

CREATE TABLE password_reset_requests (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    identifier      VARCHAR(255) NOT NULL,           -- yang diketik user: username / email / no HP
    contact         VARCHAR(100),                    -- kontak yang bisa dihubungi admin (opsional)
    note            TEXT,                            -- pesan dari user (opsional)
    request_count   INT          NOT NULL DEFAULT 1 CHECK (request_count > 0),
    status          VARCHAR(10)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved', 'rejected')),
    ip_address      VARCHAR(64),
    user_agent      TEXT,
    requested_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),  -- permintaan terakhir
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID         REFERENCES users (id),
    resolution_note TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT password_reset_requests_resolved_check
        CHECK ((status = 'pending') = (resolved_at IS NULL))
);

-- 1 user hanya punya 1 permintaan yang masih menunggu.
CREATE UNIQUE INDEX password_reset_requests_pending_unique
    ON password_reset_requests (user_id)
    WHERE status = 'pending';

-- Daftar admin: per status, terbaru dulu.
CREATE INDEX password_reset_requests_list_idx
    ON password_reset_requests (status, requested_at DESC, id DESC);

COMMENT ON TABLE  password_reset_requests               IS 'Permintaan lupa kata sandi; admin mereset sandi user secara manual';
COMMENT ON COLUMN password_reset_requests.identifier    IS 'Username / email / no HP yang diketik di form Lupa kata sandi';
COMMENT ON COLUMN password_reset_requests.request_count IS 'Berapa kali user mengirim permintaan selama masih menunggu';
COMMENT ON COLUMN password_reset_requests.resolved_by   IS 'Admin yang mereset kata sandi / menolak permintaan';

ALTER TABLE refresh_tokens ADD COLUMN remember_me BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN refresh_tokens.remember_me IS 'TRUE = login dengan "Ingat saya" (30 hari); FALSE = berakhir tanggal 1 bulan berikutnya';
