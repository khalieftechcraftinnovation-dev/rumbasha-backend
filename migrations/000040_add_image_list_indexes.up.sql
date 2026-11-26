-- ============================================================================
-- Galeri Image Library (BRD FR-IMG-001): list semua kategori urut terbaru, dan cari nama gambar /
-- nama file (sebagian kata, tanpa peduli huruf besar/kecil). List per kategori memakai index
-- media_assets_category_created_idx (migration 000012).
-- ============================================================================

CREATE INDEX media_assets_list_idx
    ON media_assets (created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX media_assets_name_trgm_idx
    ON media_assets USING gin (name gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX media_assets_file_name_trgm_idx
    ON media_assets USING gin (file_name gin_trgm_ops)
    WHERE deleted_at IS NULL;
