CREATE TABLE analytics_snapshots (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    post_id          UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    platform_post_id TEXT NOT NULL,
    metrics          JSONB NOT NULL DEFAULT '{}',
    collected_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX analytics_snapshots_workspace_idx ON analytics_snapshots(workspace_id);
CREATE INDEX analytics_snapshots_post_idx ON analytics_snapshots(post_id);
CREATE INDEX analytics_snapshots_collected_at_idx ON analytics_snapshots(collected_at);
