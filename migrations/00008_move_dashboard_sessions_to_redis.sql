-- +goose Up
DROP FUNCTION auth_private.resolve_dashboard_session(bytea);
REVOKE SELECT ON dashboard_sessions FROM meteor_auth_lookup;
DROP TABLE dashboard_sessions;

-- +goose Down
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

CREATE INDEX dashboard_sessions_by_tenant_user ON dashboard_sessions (tenant_id, dashboard_user_id);
CREATE INDEX dashboard_sessions_expiry ON dashboard_sessions (expires_at) WHERE revoked_at IS NULL;

ALTER TABLE dashboard_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY dashboard_sessions_are_isolated_by_tenant ON dashboard_sessions
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON dashboard_sessions TO meteor_app;
GRANT SELECT ON dashboard_sessions TO meteor_auth_lookup;

CREATE FUNCTION auth_private.resolve_dashboard_session(p_token_hash bytea)
RETURNS TABLE (dashboard_user_id uuid, tenant_id uuid)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT ds.dashboard_user_id, ds.tenant_id
    FROM public.dashboard_sessions AS ds
    WHERE ds.token_hash = p_token_hash
      AND ds.revoked_at IS NULL
      AND ds.expires_at > now()
    LIMIT 1
$$;
ALTER FUNCTION auth_private.resolve_dashboard_session(bytea) OWNER TO meteor_auth_lookup;
REVOKE ALL ON FUNCTION auth_private.resolve_dashboard_session(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_private.resolve_dashboard_session(bytea) TO meteor_app;
