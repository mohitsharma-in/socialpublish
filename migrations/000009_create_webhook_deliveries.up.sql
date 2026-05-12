CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id     UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    response_status INT,
    attempt_count   INT NOT NULL DEFAULT 0,
    delivered_at    TIMESTAMPTZ,
    next_retry_at   TIMESTAMPTZ
);
