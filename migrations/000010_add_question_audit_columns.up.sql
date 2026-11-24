-- Siapa yang terakhir mengubah dan siapa yang menghapus soal (untuk audit & debugging).
-- Soal dihapus dengan soft delete: deleted_at + deleted_by diisi, barisnya tetap ada.
ALTER TABLE questions
    ADD COLUMN updated_by UUID REFERENCES users (id),
    ADD COLUMN deleted_by UUID REFERENCES users (id);

COMMENT ON COLUMN questions.updated_by IS 'User yang terakhir mengubah soal (NULL = belum pernah diubah)';
COMMENT ON COLUMN questions.deleted_by IS 'User yang menghapus soal (soft delete, diisi bersama deleted_at)';
