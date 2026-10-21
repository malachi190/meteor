-- +goose Up
CREATE TABLE password_reset_tokens (
    token_hash bytea PRIMARY KEY,
    tenant_id uuid NOT NULL,
    dashboard_user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CONSTRAINT password_reset_token_hash_length_check CHECK (octet_length(token_hash) = 32),
    CONSTRAINT password_reset_token_expiry_check CHECK (expires_at > created_at),
    CONSTRAINT password_reset_user_tenant_fk
        FOREIGN KEY (tenant_id, dashboard_user_id)
        REFERENCES dashboard_users (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX password_reset_by_tenant_user ON password_reset_tokens (tenant_id, dashboard_user_id);
CREATE INDEX password_reset_expiry ON password_reset_tokens (expires_at) WHERE consumed_at IS NULL;

ALTER TABLE password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_reset_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY password_reset_tokens_are_isolated_by_tenant ON password_reset_tokens
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON password_reset_tokens TO meteor_app;
GRANT SELECT ON password_reset_tokens TO meteor_auth_lookup;

-- +goose Down
DROP TABLE password_reset_tokens;
