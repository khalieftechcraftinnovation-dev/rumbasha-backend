ALTER TABLE users DROP CONSTRAINT IF EXISTS users_end_after_join;
ALTER TABLE users
    DROP COLUMN IF EXISTS join_date,
    DROP COLUMN IF EXISTS end_date;
