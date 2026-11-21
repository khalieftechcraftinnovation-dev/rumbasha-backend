-- ============================================================================
-- Paket langganan dan akses fitur siswa. (BRD 3.8, FR-TOK-012 s/d 015, 8.7)
--
-- Relasi singkat:
--   subscription_plans ──< subscription_plan_prices      (harga per periode)
--   subscription_plans ──< subscription_plan_features >── features   (akses fitur paket)
--   students ──< student_subscriptions >── subscription_plans        (paket milik siswa)
--   student_subscriptions ──< subscription_histories                 (riwayat paket)
--   students ──< student_feature_overrides >── features              (akses khusus per siswa)
--
-- Uang disimpan dalam rupiah tanpa desimal (BIGINT), contoh 450000 = Rp450.000.
-- ============================================================================


-- Fitur yang aksesnya diatur oleh paket, beserta aturan kuota & biaya coin-nya.
-- (FR-TOK-014; BR-019: Daily Quiz gratis 2x sehari, setelah itu 10 coin)
CREATE TABLE features (
    code             VARCHAR(30)  PRIMARY KEY,  -- dipakai di kode program, jangan diubah
    name             VARCHAR(100) NOT NULL,
    daily_free_quota INT          CHECK (daily_free_quota >= 0),  -- NULL = tidak ada batas harian
    coin_cost        INT          CHECK (coin_cost >= 0),         -- NULL = tidak bisa dibayar dengan coin
    sort_order       INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Kuota & biaya selain Daily Quiz belum ditentukan (BD-006), jadi dibiarkan NULL.
-- Nilainya bisa diubah nanti tanpa mengubah kode (cukup UPDATE tabel ini).
INSERT INTO features (code, name, daily_free_quota, coin_cost, sort_order) VALUES
    ('module',     'Modul',      NULL, NULL, 1),
    ('daily_quiz', 'Daily Quiz', 2,    10,   2),
    ('uji_kom',    'Uji Kom',    NULL, NULL, 3),
    ('tryout',     'Tryout',     NULL, NULL, 4),
    ('rumath',     'RUMATH',     NULL, NULL, 5);

COMMENT ON TABLE  features                  IS 'Fitur yang aksesnya diatur paket, plus aturan kuota & coin';
COMMENT ON COLUMN features.daily_free_quota IS 'Jatah gratis per hari. NULL = tidak dibatasi';
COMMENT ON COLUMN features.coin_cost        IS 'Biaya coin setelah jatah gratis habis. NULL = tidak bisa pakai coin';


-- Paket langganan, contoh: Reguler SD, Reguler SMP, Intensif Masuk SMA. (FR-SUB-001, 014)
CREATE TABLE subscription_plans (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(100) NOT NULL,
    description        TEXT,
    highlights         TEXT,  -- poin fitur untuk ditampilkan; satu baris = satu poin
    education_level_id UUID         REFERENCES master_data_items (id),  -- sasaran jenjang; NULL = semua
    monthly_coin       INT          NOT NULL DEFAULT 0 CHECK (monthly_coin >= 0),
    is_featured        BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order         INT          NOT NULL DEFAULT 0,
    status             VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE UNIQUE INDEX subscription_plans_name_unique
    ON subscription_plans (LOWER(name))
    WHERE deleted_at IS NULL;

COMMENT ON TABLE  subscription_plans              IS 'Paket langganan siswa';
COMMENT ON COLUMN subscription_plans.highlights   IS 'Poin fitur untuk ditampilkan, satu baris = satu poin';
COMMENT ON COLUMN subscription_plans.monthly_coin IS 'Bonus coin tiap bulan, contoh Reguler SMP = 100';
COMMENT ON COLUMN subscription_plans.is_featured  IS 'Label "Unggulan" / "Paling banyak dipilih"';
COMMENT ON COLUMN subscription_plans.deleted_at   IS 'Soft delete: paket yang pernah dipakai siswa tidak boleh hilang dari riwayat';


-- Harga paket per periode: Bulanan, Semester, Tahunan. (FR-SUB-010)
CREATE TABLE subscription_plan_prices (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id      UUID        NOT NULL REFERENCES subscription_plans (id) ON DELETE CASCADE,
    period       VARCHAR(10) NOT NULL CHECK (period IN ('monthly', 'semester', 'yearly')),
    price        BIGINT      NOT NULL CHECK (price >= 0),  -- harga total untuk 1 periode
    strike_price BIGINT,                                   -- harga coret (opsional)
    status       VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Satu paket hanya punya satu harga untuk setiap periode.
    CONSTRAINT subscription_plan_prices_plan_period_unique UNIQUE (plan_id, period),
    -- Harga coret harus lebih mahal dari harga jual.
    CONSTRAINT subscription_plan_prices_strike_price_valid CHECK (strike_price IS NULL OR strike_price > price)
);

COMMENT ON TABLE  subscription_plan_prices        IS 'Harga paket per periode';
COMMENT ON COLUMN subscription_plan_prices.period IS 'monthly = 1 bulan, semester = 6 bulan, yearly = 12 bulan';
COMMENT ON COLUMN subscription_plan_prices.price  IS 'Harga total 1 periode dalam rupiah, contoh 450000';


-- Fitur yang terbuka untuk setiap paket (kolom "Akses" di menu Coin & Akses). (FR-TOK-014)
-- Contoh: Reguler SD → module, daily_quiz; Intensif Masuk SMA → semua + rumath.
CREATE TABLE subscription_plan_features (
    plan_id      UUID        NOT NULL REFERENCES subscription_plans (id) ON DELETE CASCADE,
    feature_code VARCHAR(30) NOT NULL REFERENCES features (code),
    PRIMARY KEY (plan_id, feature_code)
);

COMMENT ON TABLE subscription_plan_features IS 'Fitur yang terbuka untuk setiap paket';


-- Paket yang dimiliki siswa. (FR-SUB-003 s/d 007)
-- Perpanjangan paket mengubah end_date di baris yang sama (FR-SUB-011);
-- setiap kejadian (dibuat, diperpanjang, dibatalkan, ...) dicatat di subscription_histories.
--
-- Tanpa ON DELETE CASCADE: data paket & pembayaran tidak boleh ikut terhapus
-- kalau ada yang tidak sengaja menghapus siswa secara permanen.
CREATE TABLE student_subscriptions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID        NOT NULL REFERENCES students (user_id),
    plan_id    UUID        NOT NULL REFERENCES subscription_plans (id),
    status     VARCHAR(10) NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'active', 'expired', 'cancelled')),
    start_date DATE        NOT NULL,
    end_date   DATE        NOT NULL,  -- hari terakhir paket masih aktif ("Berlaku s/d")
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT student_subscriptions_dates_valid CHECK (end_date >= start_date)
);

-- Satu siswa hanya boleh punya SATU paket yang sedang berjalan (pending atau active).
-- Paket lama yang sudah expired/cancelled tidak ikut dihitung, jadi riwayatnya tetap ada.
CREATE UNIQUE INDEX student_subscriptions_one_current_per_student
    ON student_subscriptions (student_id)
    WHERE status IN ('pending', 'active');

CREATE INDEX student_subscriptions_plan_id_idx         ON student_subscriptions (plan_id);
-- Untuk job pengecekan paket expired dan filter "Berakhir" di menu Coin & Akses.
CREATE INDEX student_subscriptions_status_end_date_idx ON student_subscriptions (status, end_date);

COMMENT ON TABLE  student_subscriptions          IS 'Paket yang dimiliki siswa';
COMMENT ON COLUMN student_subscriptions.status   IS 'pending = menunggu aktif/bayar, active, expired = lewat end_date, cancelled';
COMMENT ON COLUMN student_subscriptions.end_date IS 'Hari terakhir paket aktif (tanggal Asia/Jakarta)';


-- Riwayat setiap kejadian pada paket siswa. (FR-SUB-008)
-- Harga disimpan sebagai salinan (snapshot), karena harga paket bisa berubah di kemudian hari.
CREATE TABLE subscription_histories (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID        NOT NULL REFERENCES student_subscriptions (id),
    action          VARCHAR(20) NOT NULL
                    CHECK (action IN ('created', 'activated', 'renewed', 'plan_changed', 'cancelled', 'expired')),
    plan_id         UUID        NOT NULL REFERENCES subscription_plans (id),
    plan_price_id   UUID        REFERENCES subscription_plan_prices (id),
    price_paid      BIGINT      CHECK (price_paid >= 0),
    start_date      DATE,       -- masa aktif setelah kejadian ini
    end_date        DATE,
    performed_by    UUID        REFERENCES users (id),  -- admin yang melakukan; NULL = otomatis oleh sistem
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX subscription_histories_subscription_id_idx ON subscription_histories (subscription_id);

COMMENT ON TABLE  subscription_histories            IS 'Riwayat paket siswa: dibuat, diperpanjang, dibatalkan, expired';
COMMENT ON COLUMN subscription_histories.price_paid IS 'Harga saat kejadian (salinan), dalam rupiah';
COMMENT ON COLUMN subscription_histories.performed_by IS 'Admin yang melakukan; NULL = otomatis oleh sistem';


-- Akses khusus per siswa yang diubah admin, di luar aturan paketnya. (FR-TOK-015)
-- Akses akhir siswa = fitur dari paket aktif, lalu ditimpa oleh baris di tabel ini.
CREATE TABLE student_feature_overrides (
    student_id   UUID        NOT NULL REFERENCES students (user_id) ON DELETE CASCADE,
    feature_code VARCHAR(30) NOT NULL REFERENCES features (code),
    is_enabled   BOOLEAN     NOT NULL,  -- TRUE = dibuka, FALSE = ditutup
    updated_by   UUID        REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, feature_code)
);

COMMENT ON TABLE  student_feature_overrides            IS 'Akses fitur khusus per siswa (menimpa aturan paket)';
COMMENT ON COLUMN student_feature_overrides.is_enabled IS 'TRUE = dibuka walau tidak ada di paket, FALSE = ditutup walau ada di paket';
