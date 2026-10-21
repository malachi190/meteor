-- +goose Up
CREATE TABLE dashboard_users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    email text NOT NULL,
    normalized_email text NOT NULL,
    password_hash text,
    email_verified_at timestamptz,
    role text NOT NULL DEFAULT 'owner',
    created_at timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz,
    CONSTRAINT dashboard_users_normalized_email_check
        CHECK (normalized_email = lower(btrim(normalized_email)) AND normalized_email <> ''),
    CONSTRAINT dashboard_users_role_check
        CHECK (role IN ('owner', 'admin', 'member')),
    CONSTRAINT dashboard_users_normalized_email_unique UNIQUE (normalized_email),
    CONSTRAINT dashboard_users_tenant_id_unique UNIQUE (tenant_id, id)
);

CREATE TABLE user_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    dashboard_user_id uuid NOT NULL,
    provider text NOT NULL,
    provider_subject text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_identities_provider_check CHECK (provider = 'github'),
    CONSTRAINT user_identities_subject_check CHECK (provider_subject <> ''),
    CONSTRAINT user_identities_provider_subject_unique UNIQUE (provider, provider_subject),
    CONSTRAINT user_identities_user_provider_unique UNIQUE (dashboard_user_id, provider),
    CONSTRAINT user_identities_user_tenant_fk
        FOREIGN KEY (tenant_id, dashboard_user_id)
        REFERENCES dashboard_users (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE dashboard_sessions (
    token_hash bytea PRIMARY KEY,
    tenant_id uuid NOT NULL,
    dashboard_user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CONSTRAINT dashboard_sessions_token_hash_length_check CHECK (octet_length(token_hash) = 32),
    CONSTRAINT dashboard_sessions_expiry_check CHECK (expires_at > created_at),
    CONSTRAINT dashboard_sessions_user_tenant_fk
        FOREIGN KEY (tenant_id, dashboard_user_id)
        REFERENCES dashboard_users (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE email_verification_tokens (
    token_hash bytea PRIMARY KEY,
    tenant_id uuid NOT NULL,
    dashboard_user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CONSTRAINT email_verification_token_hash_length_check CHECK (octet_length(token_hash) = 32),
    CONSTRAINT email_verification_token_expiry_check CHECK (expires_at > created_at),
    CONSTRAINT email_verification_user_tenant_fk
        FOREIGN KEY (tenant_id, dashboard_user_id)
        REFERENCES dashboard_users (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX user_identities_by_tenant_user ON user_identities (tenant_id, dashboard_user_id);
CREATE INDEX dashboard_sessions_by_tenant_user ON dashboard_sessions (tenant_id, dashboard_user_id);
CREATE INDEX dashboard_sessions_expiry ON dashboard_sessions (expires_at) WHERE revoked_at IS NULL;
CREATE INDEX email_verification_by_tenant_user ON email_verification_tokens (tenant_id, dashboard_user_id);
CREATE INDEX email_verification_expiry ON email_verification_tokens (expires_at) WHERE consumed_at IS NULL;

ALTER TABLE dashboard_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard_users FORCE ROW LEVEL SECURITY;
CREATE POLICY dashboard_users_are_isolated_by_tenant ON dashboard_users
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE user_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_identities FORCE ROW LEVEL SECURITY;
CREATE POLICY user_identities_are_isolated_by_tenant ON user_identities
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE dashboard_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY dashboard_sessions_are_isolated_by_tenant ON dashboard_sessions
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_verification_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY email_verification_tokens_are_isolated_by_tenant ON email_verification_tokens
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE
    ON dashboard_users, user_identities, dashboard_sessions, email_verification_tokens
    TO meteor_app;

-- +goose Down
DROP TABLE email_verification_tokens;
DROP TABLE dashboard_sessions;
DROP TABLE user_identities;
DROP TABLE dashboard_users;
