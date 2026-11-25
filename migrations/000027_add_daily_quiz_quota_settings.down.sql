-- Setelah rollback, jatah & biaya kembali diatur lewat .env (DAILY_QUIZ_FREE_QUOTA, DAILY_QUIZ_COIN_COST)
-- pada kode versi sebelumnya.
ALTER TABLE daily_quiz_settings
    DROP COLUMN IF EXISTS free_per_day,
    DROP COLUMN IF EXISTS coin_cost;
