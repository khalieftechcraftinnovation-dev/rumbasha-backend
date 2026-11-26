DROP INDEX IF EXISTS student_subscriptions_idempotency_key_unique;
ALTER TABLE student_subscriptions
    DROP COLUMN IF EXISTS idempotency_key,
    DROP COLUMN IF EXISTS cancelled_by;

DROP INDEX IF EXISTS subscription_plans_list_idx;
ALTER TABLE subscription_plans
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;
