DROP INDEX IF EXISTS quizzes_access_code_end_at_idx;
CREATE INDEX quizzes_access_code_start_at_idx
    ON quizzes (start_at)
    WHERE deleted_at IS NULL AND access_code IS NOT NULL;

ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_no_overlap;
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_end_after_start;
ALTER TABLE quizzes DROP COLUMN IF EXISTS end_at;

COMMENT ON COLUMN quizzes.start_at IS 'Waktu mulai quiz (diinput & ditampilkan dalam WIB). Selesai = start_at + duration_minutes';
COMMENT ON COLUMN quizzes.duration_minutes IS NULL;

-- btree_gist tidak dihapus: extension bersama, bisa dipakai fitur lain.
