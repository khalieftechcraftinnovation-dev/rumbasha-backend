-- ============================================================================
-- Coin dan kuota gratis harian. (BRD 3.9, 8.8)
--
-- Relasi singkat:
--   students ──1 student_coin_balances                      (saldo saat ini)
--   students ──< coin_transactions                          (riwayat masuk/keluar coin)
--   coin_packages ──< coin_purchases >── students           (pembelian "Isi coin")
--   coin_purchases ──1 coin_transactions                    (coin masuk dari pembelian)
--   students ──< daily_quota_usages >── features            (pemakaian jatah gratis per hari)
--
-- Aturan penting (BR-010, BR-012):
--   - Saldo coin tidak boleh negatif → dijaga CHECK (balance >= 0).
--   - Setiap perubahan saldo WAJIB dicatat di coin_transactions, dalam satu
--     database transaction yang sama dengan UPDATE saldo.
-- ============================================================================


-- Paket isi coin, contoh: 100 coin Rp25.000, 300 coin Rp68.000. (FR-TOK-010)
CREATE TABLE coin_packages (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    coin_amount INT          NOT NULL CHECK (coin_amount > 0),
    price       BIGINT       NOT NULL CHECK (price >= 0),  -- rupiah
    description VARCHAR(150),                              -- contoh: "Hemat 10%", "Untuk 2 tryout"
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

COMMENT ON TABLE  coin_packages             IS 'Paket isi coin yang bisa dibeli siswa';
COMMENT ON COLUMN coin_packages.coin_amount IS 'Jumlah coin yang didapat';
COMMENT ON COLUMN coin_packages.price       IS 'Harga dalam rupiah, contoh 25000';


-- Saldo coin siswa saat ini. (FR-TOK-001, FR-TOK-005)
-- Dipisah dari tabel students supaya baris ini bisa dikunci (SELECT ... FOR UPDATE)
-- saat mengurangi coin, tanpa mengunci data profil siswa.
CREATE TABLE student_coin_balances (
    student_id UUID        PRIMARY KEY REFERENCES students (user_id),
    balance    INT         NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  student_coin_balances         IS 'Saldo coin siswa saat ini';
COMMENT ON COLUMN student_coin_balances.balance IS 'Tidak boleh negatif (BR-010)';


-- Riwayat setiap coin masuk dan keluar. (FR-TOK-003, FR-TOK-007, FR-TOK-016)
-- Baris di tabel ini tidak pernah diubah atau dihapus; koreksi dicatat sebagai transaksi baru.
CREATE TABLE coin_transactions (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID         NOT NULL REFERENCES students (user_id),
    type            VARCHAR(20)  NOT NULL
                    CHECK (type IN ('plan_grant', 'purchase', 'usage', 'admin_adjustment', 'refund')),
    amount          INT          NOT NULL CHECK (amount <> 0),       -- positif = masuk, negatif = keluar
    balance_after   INT          NOT NULL CHECK (balance_after >= 0),  -- saldo setelah transaksi ini
    reference_type  VARCHAR(50),  -- asal transaksi, contoh: daily_quiz_attempt, coin_purchase
    reference_id    UUID,         -- ID data asalnya
    idempotency_key VARCHAR(100) UNIQUE,
    description     VARCHAR(255),
    created_by      UUID         REFERENCES users (id),  -- admin yang mengubah; NULL = sistem/siswa
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Setiap pemakaian coin wajib punya referensi (Important Technical Rules BRD 3.9).
    CONSTRAINT coin_transactions_usage_has_reference
        CHECK (type <> 'usage' OR (reference_type IS NOT NULL AND reference_id IS NOT NULL)),
    -- Perubahan coin oleh admin wajib tercatat siapa adminnya.
    CONSTRAINT coin_transactions_admin_has_actor
        CHECK (type <> 'admin_adjustment' OR created_by IS NOT NULL)
);

-- Untuk menampilkan riwayat coin per siswa, terbaru di atas.
CREATE INDEX coin_transactions_student_id_created_at_idx ON coin_transactions (student_id, created_at DESC);
CREATE INDEX coin_transactions_reference_idx             ON coin_transactions (reference_type, reference_id);

COMMENT ON TABLE  coin_transactions                 IS 'Riwayat coin masuk/keluar (tidak pernah diubah atau dihapus)';
COMMENT ON COLUMN coin_transactions.type            IS 'plan_grant = bonus paket, purchase = beli, usage = dipakai, admin_adjustment = diubah admin, refund';
COMMENT ON COLUMN coin_transactions.amount          IS 'Positif = coin masuk, negatif = coin keluar';
COMMENT ON COLUMN coin_transactions.idempotency_key IS 'Kunci unik per request, mencegah coin terpotong dua kali';


-- Pembelian paket isi coin oleh siswa. (FR-TOK-010)
-- Cara pembayaran (lewat admin atau payment gateway) belum ditentukan (BD-011).
CREATE TABLE coin_purchases (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id          UUID        NOT NULL REFERENCES students (user_id),
    coin_package_id     UUID        NOT NULL REFERENCES coin_packages (id),
    coin_amount         INT         NOT NULL CHECK (coin_amount > 0),  -- salinan saat membeli
    price               BIGINT      NOT NULL CHECK (price >= 0),       -- salinan saat membeli
    status              VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'cancelled')),
    paid_at             TIMESTAMPTZ,
    coin_transaction_id UUID        UNIQUE REFERENCES coin_transactions (id),  -- terisi setelah coin masuk
    created_by          UUID        REFERENCES users (id),  -- admin yang mencatat; NULL = dibeli siswa sendiri
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT coin_purchases_paid_has_time CHECK (status <> 'paid' OR paid_at IS NOT NULL)
);

CREATE INDEX coin_purchases_student_id_idx ON coin_purchases (student_id);

COMMENT ON TABLE  coin_purchases                     IS 'Pembelian paket isi coin';
COMMENT ON COLUMN coin_purchases.coin_transaction_id IS 'Transaksi coin masuk; terisi setelah pembelian dibayar';


-- Pemakaian jatah gratis harian per siswa per fitur. (FR-TOK-004, FR-TOK-006, FR-TOK-008)
-- "Reset harian" terjadi otomatis: hari baru = usage_date baru = baris baru dengan
-- used_count 0. Jadi tidak butuh job reset, dan tidak mungkin ter-reset dua kali.
CREATE TABLE daily_quota_usages (
    student_id   UUID        NOT NULL REFERENCES students (user_id) ON DELETE CASCADE,
    feature_code VARCHAR(30) NOT NULL REFERENCES features (code),
    usage_date   DATE        NOT NULL,  -- tanggal menurut Asia/Jakarta
    used_count   INT         NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, feature_code, usage_date)
);

COMMENT ON TABLE  daily_quota_usages            IS 'Pemakaian jatah gratis per hari (reset otomatis karena per tanggal)';
COMMENT ON COLUMN daily_quota_usages.usage_date IS 'Tanggal menurut Asia/Jakarta';
COMMENT ON COLUMN daily_quota_usages.used_count IS 'Berapa kali jatah gratis sudah dipakai di tanggal ini';
