-- ============================================================================
-- Coin. (BRD 3.9, 8.8)
--
-- Relasi singkat:
--   students.coin_balance            (saldo saat ini, ada di tabel students)
--   students ──< coin_transactions    (riwayat coin masuk/keluar)
--   coin_packages                     (daftar paket "Isi coin")
--
-- Aturan penting (BR-010, BR-012):
--   - Saldo tidak boleh negatif → dijaga CHECK (coin_balance >= 0) di students.
--   - Setiap perubahan students.coin_balance WAJIB disertai 1 baris coin_transactions,
--     dalam satu database transaction yang sama.
--
-- Jatah gratis harian (contoh Daily Quiz 2x sehari, lalu 10 coin) berlaku global
-- untuk semua siswa, jadi diatur di konfigurasi aplikasi (.env), bukan di tabel.
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


-- Riwayat setiap coin masuk dan keluar. (FR-TOK-003, FR-TOK-007, FR-TOK-016)
-- Pembelian coin juga dicatat di sini (type = purchase).
-- Baris di tabel ini tidak pernah diubah atau dihapus; koreksi dicatat sebagai transaksi baru.
CREATE TABLE coin_transactions (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID         NOT NULL REFERENCES students (user_id),
    type            VARCHAR(20)  NOT NULL
                    CHECK (type IN ('plan_grant', 'purchase', 'usage', 'admin_adjustment', 'refund')),
    amount          INT          NOT NULL CHECK (amount <> 0),         -- positif = masuk, negatif = keluar
    balance_after   INT          NOT NULL CHECK (balance_after >= 0),  -- saldo setelah transaksi ini
    price_paid      BIGINT       CHECK (price_paid >= 0),              -- uang yang dibayar (khusus purchase)
    reference_type  VARCHAR(50),  -- asal transaksi, contoh: coin_package, student_subscription, daily_quiz_attempt
    reference_id    UUID,         -- ID data asalnya
    idempotency_key VARCHAR(100) UNIQUE,
    description     VARCHAR(255),
    created_by      UUID         REFERENCES users (id),  -- admin yang mencatat; NULL = sistem/siswa
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Setiap pemakaian coin wajib punya referensi (Important Technical Rules BRD 3.9).
    CONSTRAINT coin_transactions_usage_has_reference
        CHECK (type <> 'usage' OR (reference_type IS NOT NULL AND reference_id IS NOT NULL)),
    -- Pembelian coin wajib mencatat paket yang dibeli dan harganya.
    CONSTRAINT coin_transactions_purchase_has_package
        CHECK (type <> 'purchase' OR (reference_type = 'coin_package' AND reference_id IS NOT NULL AND price_paid IS NOT NULL)),
    -- Perubahan coin oleh admin wajib tercatat siapa adminnya.
    CONSTRAINT coin_transactions_admin_has_actor
        CHECK (type <> 'admin_adjustment' OR created_by IS NOT NULL)
);

-- Untuk menampilkan riwayat coin per siswa, terbaru di atas.
CREATE INDEX coin_transactions_student_id_created_at_idx ON coin_transactions (student_id, created_at DESC);
CREATE INDEX coin_transactions_reference_idx             ON coin_transactions (reference_type, reference_id);

COMMENT ON TABLE  coin_transactions                 IS 'Riwayat coin masuk/keluar, termasuk pembelian (tidak pernah diubah atau dihapus)';
COMMENT ON COLUMN coin_transactions.type            IS 'plan_grant = bonus paket, purchase = beli, usage = dipakai, admin_adjustment = diubah admin, refund';
COMMENT ON COLUMN coin_transactions.amount          IS 'Positif = coin masuk, negatif = coin keluar';
COMMENT ON COLUMN coin_transactions.price_paid      IS 'Uang yang dibayar (rupiah), wajib untuk type purchase';
COMMENT ON COLUMN coin_transactions.idempotency_key IS 'Kunci unik per request, mencegah coin terpotong dua kali';
