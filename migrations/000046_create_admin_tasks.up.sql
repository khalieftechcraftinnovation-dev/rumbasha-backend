CREATE TABLE admin_tasks (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_text          TEXT         NOT NULL CHECK (char_length(btrim(task_text)) BETWEEN 1 AND 5000),
    status             VARCHAR(16)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    created_by         UUID         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    status_updated_by  UUID         REFERENCES users (id) ON DELETE RESTRICT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    status_updated_at TIMESTAMPTZ
);

CREATE INDEX admin_tasks_created_at_idx ON admin_tasks (created_at DESC, id DESC);

COMMENT ON TABLE admin_tasks IS 'Shared text tasks created by Super Admin for all Admin users';
COMMENT ON COLUMN admin_tasks.status IS 'pending or completed; status is shared by all Admin users';
COMMENT ON COLUMN admin_tasks.status_updated_by IS 'Last Admin who changed the shared task status';
