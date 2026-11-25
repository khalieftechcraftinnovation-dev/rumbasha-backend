-- ============================================================================
-- Manajemen user oleh admin: list (filter role/jenjang/kelas/cabang/status + cari), detail,
-- ubah, hapus (soft delete). (BRD FR-STU-012 s/d FR-STU-015, FR-STU-019 s/d FR-STU-022)
--
--   - updated_by / deleted_by: siapa yang terakhir mengubah / menghapus akun (audit).
--   - Index list: terbaru dulu, hanya user yang belum dihapus.
--   - Index trigram (pg_trgm, sudah ada sejak migration 000013) untuk cari nama / username /
--     email dengan ILIKE '%kata%' tanpa membaca seluruh tabel.
-- ============================================================================

ALTER TABLE users
    ADD COLUMN updated_by UUID REFERENCES users (id),
    ADD COLUMN deleted_by UUID REFERENCES users (id);

COMMENT ON COLUMN users.updated_by IS 'Admin yang terakhir mengubah akun lewat menu User (NULL = belum pernah)';
COMMENT ON COLUMN users.deleted_by IS 'Admin yang menghapus akun (soft delete, diisi bersama deleted_at)';

CREATE INDEX users_list_idx ON users (created_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE INDEX users_real_name_trgm_idx ON users USING gin (real_name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX users_username_trgm_idx  ON users USING gin (username  gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX users_email_trgm_idx     ON users USING gin (email     gin_trgm_ops) WHERE deleted_at IS NULL;
