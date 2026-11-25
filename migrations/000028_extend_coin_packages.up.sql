-- ============================================================================
-- Paket isi coin dikelola admin (CRUD) + top-up coin siswa oleh admin per paket.
-- (BRD FR-TOK-007, FR-TOK-010, FR-TOK-015, FR-TOK-018 s/d FR-TOK-021)
--
--   - coin_packages: kolom audit (siapa membuat / mengubah / menghapus) & nama paket unik
--     (tidak peka huruf besar/kecil) di antara paket yang belum dihapus.
--   - Top-up oleh admin dicatat di coin_transactions sebagai type 'purchase' (bayar tunai di cabang):
--     reference coin_package + price_paid + created_by = admin pencatat.
-- ============================================================================

ALTER TABLE coin_packages
    ADD COLUMN created_by UUID REFERENCES users (id),  -- NULL = data awal (seed)
    ADD COLUMN updated_by UUID REFERENCES users (id),
    ADD COLUMN deleted_by UUID REFERENCES users (id);

CREATE UNIQUE INDEX coin_packages_name_unique ON coin_packages (LOWER(name)) WHERE deleted_at IS NULL;

-- List paket (urut tampil) hanya paket yang belum dihapus.
CREATE INDEX coin_packages_sort_idx ON coin_packages (sort_order, coin_amount) WHERE deleted_at IS NULL;

COMMENT ON COLUMN coin_packages.created_by IS 'Admin yang membuat paket (NULL = data awal)';
