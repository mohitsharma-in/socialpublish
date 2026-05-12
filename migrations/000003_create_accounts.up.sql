CREATE TABLE accounts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform         TEXT NOT NULL,
    platform_user_id TEXT NOT NULL,
    display_name     TEXT NOT NULL,
    avatar_url       TEXT,
    token_id         UUID NOT NULL,
    token_expires_at TIMESTAMPTZ,
    token_healthy    BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, platform, platform_user_id)
);

CREATE INDEX accounts_workspace_id_idx ON accounts(workspace_id);
