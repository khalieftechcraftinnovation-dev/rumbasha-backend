ALTER TABLE questions DROP CONSTRAINT questions_image_id_fkey;
ALTER TABLE questions ADD CONSTRAINT questions_image_id_fkey
    FOREIGN KEY (image_id) REFERENCES media_assets (id);

DROP INDEX media_assets_hierarchy_idx;
DROP INDEX media_assets_category_created_idx;
ALTER TABLE media_assets DROP CONSTRAINT media_assets_hierarchy_check;

ALTER TABLE media_assets
    DROP COLUMN category,
    DROP COLUMN name,
    DROP COLUMN url,
    DROP COLUMN width,
    DROP COLUMN height,
    DROP COLUMN subject_id,
    DROP COLUMN subject_field_id,
    DROP COLUMN chapter_id,
    DROP COLUMN subchapter_id,
    DROP COLUMN topic_id;

COMMENT ON COLUMN media_assets.storage_path IS NULL;
