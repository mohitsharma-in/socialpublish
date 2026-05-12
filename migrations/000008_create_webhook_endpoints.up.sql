CREATE TABLE webhook_endpoints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    url          TEXT NOT NULL,
    secret_hash  TEXT NOT NULL,
    secret_enc   BYTEA NOT NULL,
    events       TEXT[] NOT NULL DEFAULT '{}',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
