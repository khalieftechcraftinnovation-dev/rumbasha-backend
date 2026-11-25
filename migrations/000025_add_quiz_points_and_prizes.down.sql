DROP TABLE IF EXISTS quiz_prize_awards;

-- Transaksi hadiah tidak bisa disimpan dengan aturan lama. Riwayat coin tidak pernah dihapus,
-- jadi rollback ditolak kalau sudah ada hadiah yang dibagikan.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM coin_transactions WHERE type = 'reward') THEN
        RAISE EXCEPTION 'sudah ada coin_transactions type reward (hadiah quiz); rollback dibatalkan';
    END IF;
END $$;

ALTER TABLE coin_transactions DROP CONSTRAINT IF EXISTS coin_transactions_reward_has_reference;
ALTER TABLE coin_transactions DROP CONSTRAINT coin_transactions_type_check;
ALTER TABLE coin_transactions ADD CONSTRAINT coin_transactions_type_check
    CHECK (type IN ('plan_grant', 'purchase', 'usage', 'admin_adjustment', 'refund'));

DROP INDEX IF EXISTS quizzes_prize_pending_idx;

ALTER TABLE quizzes
    DROP COLUMN IF EXISTS points_easy,
    DROP COLUMN IF EXISTS points_medium,
    DROP COLUMN IF EXISTS points_hard,
    DROP COLUMN IF EXISTS prize_rank_1,
    DROP COLUMN IF EXISTS prize_rank_2,
    DROP COLUMN IF EXISTS prize_rank_3,
    DROP COLUMN IF EXISTS prizes_distributed_at;
