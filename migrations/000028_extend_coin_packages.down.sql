DROP INDEX IF EXISTS coin_packages_sort_idx;
DROP INDEX IF EXISTS coin_packages_name_unique;
ALTER TABLE coin_packages
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS deleted_by;
