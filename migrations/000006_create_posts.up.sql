CREATE TABLE posts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'draft',
    media_ids    UUID[] NOT NULL DEFAULT '{}',
    scheduled_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX posts_workspace_status_idx ON posts(workspace_id, status);
CREATE INDEX posts_scheduled_at_idx ON posts(scheduled_at) WHERE status = 'scheduled';
