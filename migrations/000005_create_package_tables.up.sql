-- ============================================================================
-- Paket langganan dan akses fitur siswa. (BRD 3.8, FR-TOK-012 s/d 015, 8.7)
--
-- Relasi singkat:
--   students ──< student_subscriptions >── subscription_plans   (paket yang dibeli siswa)
--   students ──< student_feature_overrides                      (akses khusus per siswa)
--
-- Daftar fitur (dipakai di kolom has_* dan di student_feature_overrides):
--   module, daily_quiz, uji_kom, tryout, rumath
--
-- Uang disimpan dalam rupiah tanpa desimal (BIGINT), contoh 450000 = Rp450.000.
-- ============================================================================

-- Dibutuhkan untuk constraint "tidak boleh ada 2 paket aktif di tanggal yang sama"
-- (EXCLUDE ... WITH &&) di tabel student_subscriptions. Bawaan PostgreSQL.
CREATE EXTENSION IF NOT EXISTS btree_gist;


-- Paket langganan, contoh: Reguler SD, Reguler SMP, Intensif Masuk SMA. (FR-SUB-001, 010, 014)
CREATE TABLE subscription_plans (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 VARCHAR(100) NOT NULL,
    description          TEXT,
    highlights           TEXT,  -- poin fitur untuk ditampilkan; satu baris = satu poin
    education_level_id   UUID         REFERENCES master_data_items (id),  -- sasaran jenjang; NULL = semua
    monthly_coin         INT          NOT NULL DEFAULT 0 CHECK (monthly_coin >= 0),
    is_featured          BOOLEAN      NOT NULL DEFAULT FALSE,

    -- Harga per periode (rupiah). NULL = periode itu tidak dijual.
    price_monthly        BIGINT       CHECK (price_monthly >= 0),
    price_semester       BIGINT       CHECK (price_semester >= 0),
    price_yearly         BIGINT       CHECK (price_yearly >= 0),
    strike_price_monthly BIGINT,      -- harga coret untuk harga bulanan (opsional)

    -- Akses fitur yang didapat dari paket ini (kolom "Akses" di menu Coin & Akses).
    has_module           BOOLEAN      NOT NULL DEFAULT FALSE,
    has_daily_quiz       BOOLEAN      NOT NULL DEFAULT FALSE,
    has_uji_kom          BOOLEAN      NOT NULL DEFAULT FALSE,
    has_tryout           BOOLEAN      NOT NULL DEFAULT FALSE,
    has_rumath           BOOLEAN      NOT NULL DEFAULT FALSE,

    sort_order           INT          NOT NULL DEFAULT 0,
    status               VARCHAR(10)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ,

    -- Paket minimal dijual di salah satu periode.
    CONSTRAINT subscription_plans_has_price
        CHECK (price_monthly IS NOT NULL OR price_semester IS NOT NULL OR price_yearly IS NOT NULL),
    -- Harga coret hanya boleh ada kalau ada harga bulanan, dan harus lebih mahal.
    CONSTRAINT subscription_plans_strike_price_valid
        CHECK (strike_price_monthly IS NULL OR strike_price_monthly > price_monthly)
);

CREATE UNIQUE INDEX subscription_plans_name_unique
    ON subscription_plans (LOWER(name))
    WHERE deleted_at IS NULL;

COMMENT ON TABLE  subscription_plans                      IS 'Paket langganan siswa, termasuk harga dan akses fiturnya';
COMMENT ON COLUMN subscription_plans.highlights           IS 'Poin fitur untuk ditampilkan, satu baris = satu poin';
COMMENT ON COLUMN subscription_plans.monthly_coin         IS 'Bonus coin tiap bulan, contoh Reguler SMP = 100';
COMMENT ON COLUMN subscription_plans.is_featured          IS 'Label "Unggulan" / "Paling banyak dipilih"';
COMMENT ON COLUMN subscription_plans.price_monthly        IS 'Harga 1 bulan (rupiah). NULL = tidak dijual bulanan';
COMMENT ON COLUMN subscription_plans.price_semester       IS 'Harga 6 bulan (rupiah). NULL = tidak dijual per semester';
COMMENT ON COLUMN subscription_plans.price_yearly         IS 'Harga 12 bulan (rupiah). NULL = tidak dijual tahunan';
COMMENT ON COLUMN subscription_plans.strike_price_monthly IS 'Harga coret untuk harga bulanan';
COMMENT ON COLUMN subscription_plans.deleted_at           IS 'Soft delete: paket yang pernah dipakai siswa tidak boleh hilang dari riwayat';


-- Paket yang dibeli siswa ("paket akses"; berbeda dengan paket isi coin).
-- Satu baris = satu periode yang dibeli, jadi tabel ini sekaligus menjadi
-- riwayat paket siswa (FR-SUB-008).
--
-- Ada 2 cara berlangganan (kolom channel):
--   offline → bayar manual di bimbel Rumbasha, dicatat admin, biasanya per bulan
--   online  → dibeli lewat aplikasi
-- Keduanya dicek dengan cara yang sama: setiap hari sistem melihat apakah
-- ada paket yang aktif di tanggal hari itu.
--
-- Status tidak disimpan, tapi dihitung dari tanggal:
--   aktif      → cancelled_at IS NULL AND CURRENT_DATE BETWEEN start_date AND end_date
--   expired    → cancelled_at IS NULL AND end_date < CURRENT_DATE
--   dibatalkan → cancelled_at IS NOT NULL
-- Jadi tidak butuh job untuk mengubah status menjadi expired.
--
-- Perpanjangan paket (FR-SUB-011) = baris baru yang start_date-nya tepat
-- setelah end_date paket sebelumnya.
--
-- Tanpa ON DELETE CASCADE: data pembelian tidak boleh ikut terhapus
-- kalau ada yang tidak sengaja menghapus siswa secara permanen.
CREATE TABLE student_subscriptions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id   UUID        NOT NULL REFERENCES students (user_id),
    plan_id      UUID        NOT NULL REFERENCES subscription_plans (id),
    channel      VARCHAR(10) NOT NULL CHECK (channel IN ('offline', 'online')),
    period       VARCHAR(10) NOT NULL CHECK (period IN ('monthly', 'semester', 'yearly')),
    price_paid   BIGINT      NOT NULL CHECK (price_paid >= 0),  -- salinan harga saat dibeli
    start_date   DATE        NOT NULL,
    end_date     DATE        NOT NULL,  -- hari terakhir paket aktif ("Berlaku s/d")
    cancelled_at TIMESTAMPTZ,
    notes        TEXT,
    created_by   UUID        REFERENCES users (id),  -- admin yang mencatat
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT student_subscriptions_dates_valid CHECK (end_date >= start_date),

    -- Satu siswa tidak boleh punya 2 paket (yang tidak dibatalkan) dengan tanggal bertumpuk.
    -- daterange(..., '[]') = rentang tanggal termasuk hari pertama dan terakhir; && = bertumpuk.
    CONSTRAINT student_subscriptions_no_overlap
        EXCLUDE USING gist (student_id WITH =, daterange(start_date, end_date, '[]') WITH &&)
        WHERE (cancelled_at IS NULL)
);

-- Untuk mencari paket aktif siswa dan filter "Berakhir" di menu Coin & Akses.
CREATE INDEX student_subscriptions_student_id_end_date_idx ON student_subscriptions (student_id, end_date);
CREATE INDEX student_subscriptions_plan_id_idx             ON student_subscriptions (plan_id);

COMMENT ON TABLE  student_subscriptions            IS 'Paket akses yang dibeli siswa; 1 baris = 1 periode (sekaligus riwayat)';
COMMENT ON COLUMN student_subscriptions.channel    IS 'offline = bayar manual di bimbel (dicatat admin), online = beli lewat aplikasi';
COMMENT ON COLUMN student_subscriptions.price_paid IS 'Harga saat dibeli (salinan), dalam rupiah';
COMMENT ON COLUMN student_subscriptions.end_date   IS 'Hari terakhir paket aktif (tanggal Asia/Jakarta)';
COMMENT ON COLUMN student_subscriptions.created_by IS 'Admin yang mencatat pembelian';


-- Akses khusus per siswa yang diubah admin, di luar aturan paketnya. (FR-TOK-015)
-- Akses akhir siswa = fitur dari paket aktif (has_*), lalu ditimpa oleh baris di tabel ini.
CREATE TABLE student_feature_overrides (
    student_id UUID        NOT NULL REFERENCES students (user_id) ON DELETE CASCADE,
    feature    VARCHAR(20) NOT NULL CHECK (feature IN ('module', 'daily_quiz', 'uji_kom', 'tryout', 'rumath')),
    is_enabled BOOLEAN     NOT NULL,  -- TRUE = dibuka, FALSE = ditutup
    updated_by UUID        REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, feature)
);

COMMENT ON TABLE  student_feature_overrides            IS 'Akses fitur khusus per siswa (menimpa aturan paket)';
COMMENT ON COLUMN student_feature_overrides.is_enabled IS 'TRUE = dibuka walau tidak ada di paket, FALSE = ditutup walau ada di paket';
