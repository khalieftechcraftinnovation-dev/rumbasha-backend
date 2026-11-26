CREATE TABLE announcements (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title        VARCHAR(200) NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
    body         TEXT         NOT NULL CHECK (char_length(btrim(body)) BETWEEN 1 AND 20000),
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at TIMESTAMPTZ,
    created_by   UUID         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    updated_by   UUID         REFERENCES users (id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    CHECK (
        (status = 'draft' AND published_at IS NULL) OR
        (status = 'published' AND published_at IS NOT NULL)
    )
);

CREATE TABLE announcement_grades (
    announcement_id UUID NOT NULL REFERENCES announcements (id) ON DELETE CASCADE,
    grade_id        UUID NOT NULL REFERENCES master_data_items (id) ON DELETE RESTRICT,
    PRIMARY KEY (announcement_id, grade_id)
);

CREATE TABLE announcement_reads (
    announcement_id UUID        NOT NULL REFERENCES announcements (id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (announcement_id, user_id)
);

CREATE INDEX announcements_admin_list_idx
    ON announcements (created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX announcements_student_feed_idx
    ON announcements (published_at DESC, id DESC)
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX announcement_grades_grade_idx ON announcement_grades (grade_id);
CREATE INDEX announcement_reads_user_idx ON announcement_reads (user_id, announcement_id);

COMMENT ON TABLE announcements IS 'Web-only announcements shown as in-app notifications to targeted students';
COMMENT ON TABLE announcement_grades IS 'Student grade targets for each announcement';
COMMENT ON TABLE announcement_reads IS 'Per-student read state for announcements';
