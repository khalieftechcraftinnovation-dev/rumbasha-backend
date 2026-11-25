DROP INDEX IF EXISTS users_email_trgm_idx;
DROP INDEX IF EXISTS users_username_trgm_idx;
DROP INDEX IF EXISTS users_real_name_trgm_idx;
DROP INDEX IF EXISTS users_list_idx;
ALTER TABLE users
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS deleted_by;
