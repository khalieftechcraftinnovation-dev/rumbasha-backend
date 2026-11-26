-- ============================================================================
-- Manajemen paket akses oleh admin (BRD FR-SUB-001, 002, 014, 015, FR-TOK-015).
--
--   subscription_plans     → siapa yang membuat / mengubah / menghapus paket
--   student_subscriptions  → siapa yang membatalkan + kunci unik per request (cegah klik 2x)
-- ============================================================================

ALTER TABLE subscription_plans
    ADD COLUMN created_by UUID REFERENCES users (id),
    ADD COLUMN updated_by UUID REFERENCES users (id),
    ADD COLUMN deleted_by UUID REFERENCES users (id);

COMMENT ON COLUMN subscription_plans.created_by IS 'Admin yang membuat paket (NULL = data awal)';
COMMENT ON COLUMN subscription_plans.updated_by IS 'Admin yang terakhir mengubah paket';
COMMENT ON COLUMN subscription_plans.deleted_by IS 'Admin yang menghapus paket (soft delete)';

-- Daftar paket admin & siswa diurutkan sort_order lalu nama, hanya yang belum dihapus.
CREATE INDEX subscription_plans_list_idx ON subscription_plans (sort_order, name) WHERE deleted_at IS NULL;

ALTER TABLE student_subscriptions
    ADD COLUMN cancelled_by    UUID REFERENCES users (id),
    ADD COLUMN idempotency_key VARCHAR(100);

-- Request beri paket yang sama terkirim 2x (klik ganda) tidak membuat periode dobel.
CREATE UNIQUE INDEX student_subscriptions_idempotency_key_unique
    ON student_subscriptions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

COMMENT ON COLUMN student_subscriptions.cancelled_by    IS 'Admin yang membatalkan paket';
COMMENT ON COLUMN student_subscriptions.idempotency_key IS 'Kunci unik per request beri paket: assign:<request_id>:<id siswa>';
