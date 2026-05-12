CREATE TABLE media (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'uploading',
    media_type    TEXT NOT NULL,
    original_key  TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL DEFAULT 0,
    duration_ms   INT,
    formats       JSONB NOT NULL DEFAULT '{}',
    thumbnail_key TEXT,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX media_workspace_id_status_idx ON media(workspace_id, status);
