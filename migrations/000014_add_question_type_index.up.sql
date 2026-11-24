-- ============================================================================
-- Index tipe soal untuk Bank Soal:
--   1. Jumlah soal per tipe (tab Semua / Pilihan ganda / PG kompleks / Benar-Salah / Isian):
--      SELECT type, COUNT(*) ... GROUP BY type → cukup baca index ini (index-only scan).
--   2. List saat tab tipe dipilih: WHERE type = ? ORDER BY created_at DESC, id DESC
--      → langsung urut dari index, berhenti setelah per_page baris.
-- ============================================================================
CREATE INDEX questions_type_created_at_idx
    ON questions (type, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
