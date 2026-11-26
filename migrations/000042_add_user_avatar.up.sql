-- ============================================================================
-- Foto profil akun (menu Profil siswa, BRD FR-STU-009 / FR-PRO-001).
--
-- File disimpan di STORAGE_DIR/images/avatar/<tahun>/<bulan>/<uuid>.<ext> (publik, sama seperti
-- gambar Image Library, supaya FE cukup memakai <img src="avatar_url">). Ganti foto = file lama dihapus.
-- ============================================================================

ALTER TABLE users
    ADD COLUMN avatar_path TEXT,  -- lokasi relatif di STORAGE_DIR (dipakai saat ganti / hapus foto)
    ADD COLUMN avatar_url  TEXT;  -- link lengkap untuk FE

COMMENT ON COLUMN users.avatar_path IS 'Foto profil: lokasi file relatif di STORAGE_DIR';
COMMENT ON COLUMN users.avatar_url  IS 'Foto profil: link lengkap (publik, tanpa login)';
