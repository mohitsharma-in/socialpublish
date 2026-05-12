CREATE TABLE encrypted_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ciphertext  BYTEA NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE accounts
    ADD CONSTRAINT accounts_token_id_fkey
    FOREIGN KEY (token_id) REFERENCES encrypted_tokens(id);
