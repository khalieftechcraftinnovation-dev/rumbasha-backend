-- Siapa yang membuat akun lewat menu User (tambah user satu / banyak sekaligus). (BRD FR-STU-023)
-- NULL = akun dibuat sebelum kolom ini ada (data awal / seed).
ALTER TABLE users ADD COLUMN created_by UUID REFERENCES users (id);

COMMENT ON COLUMN users.created_by IS 'Admin yang membuat akun lewat menu User (NULL = data awal)';
