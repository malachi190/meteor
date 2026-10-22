-- +goose Up
ALTER TABLE projects
    ADD CONSTRAINT projects_tenant_id_id_unique UNIQUE (tenant_id, id);

CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    prefix text NOT NULL UNIQUE,
    key_hash bytea NOT NULL UNIQUE,
    env text NOT NULL CHECK (env IN ('test', 'live')),
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    revoked_at timestamptz,
    FOREIGN KEY (tenant_id, project_id)
        REFERENCES projects (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX api_keys_by_tenant_project
    ON api_keys (tenant_id, project_id, created_at DESC);

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY api_keys_are_isolated_by_tenant ON api_keys
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO meteor_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE, DELETE ON api_keys FROM meteor_app;
DROP TABLE api_keys;
ALTER TABLE projects DROP CONSTRAINT projects_tenant_id_id_unique;
