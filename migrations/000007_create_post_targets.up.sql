CREATE TABLE post_targets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id           UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    account_id        UUID NOT NULL REFERENCES accounts(id),
    platform          TEXT NOT NULL,
    format            TEXT NOT NULL,
    config            JSONB NOT NULL DEFAULT '{}',
    status            TEXT NOT NULL DEFAULT 'pending',
    platform_post_id  TEXT,
    permalink         TEXT,
    failure_reason    TEXT,
    attempt_count     INT NOT NULL DEFAULT 0,
    last_attempted_at TIMESTAMPTZ,
    published_at      TIMESTAMPTZ
);

CREATE INDEX post_targets_post_id_idx ON post_targets(post_id);
