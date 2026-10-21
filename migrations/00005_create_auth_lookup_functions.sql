-- +goose Up
-- This role can bypass RLS only inside narrowly scoped, non-login lookup functions.
-- It has no login, no write access, and no membership granted to the API role.
CREATE ROLE meteor_auth_lookup
    NOLOGIN
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOINHERIT
    BYPASSRLS;

GRANT USAGE ON SCHEMA public TO meteor_auth_lookup;
GRANT SELECT ON dashboard_users, user_identities, dashboard_sessions, email_verification_tokens
    TO meteor_auth_lookup;

CREATE SCHEMA auth_private;
REVOKE ALL ON SCHEMA auth_private FROM PUBLIC;
GRANT USAGE ON SCHEMA auth_private TO meteor_app;
GRANT CREATE ON SCHEMA auth_private TO meteor_auth_lookup;

CREATE FUNCTION auth_private.resolve_dashboard_login(p_normalized_email text)
RETURNS TABLE (
    dashboard_user_id uuid,
    tenant_id uuid,
    password_hash text,
    email_verified_at timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT du.id, du.tenant_id, du.password_hash, du.email_verified_at
    FROM public.dashboard_users AS du
    WHERE du.normalized_email = p_normalized_email
    LIMIT 1
$$;
ALTER FUNCTION auth_private.resolve_dashboard_login(text) OWNER TO meteor_auth_lookup;
REVOKE ALL ON FUNCTION auth_private.resolve_dashboard_login(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_private.resolve_dashboard_login(text) TO meteor_app;

CREATE FUNCTION auth_private.resolve_github_identity(p_provider_subject text)
RETURNS TABLE (dashboard_user_id uuid, tenant_id uuid)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT ui.dashboard_user_id, ui.tenant_id
    FROM public.user_identities AS ui
    WHERE ui.provider = 'github'
      AND ui.provider_subject = p_provider_subject
    LIMIT 1
$$;
ALTER FUNCTION auth_private.resolve_github_identity(text) OWNER TO meteor_auth_lookup;
REVOKE ALL ON FUNCTION auth_private.resolve_github_identity(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_private.resolve_github_identity(text) TO meteor_app;

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

CREATE FUNCTION auth_private.resolve_email_verification_token(p_token_hash bytea)
RETURNS TABLE (dashboard_user_id uuid, tenant_id uuid)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT ev.dashboard_user_id, ev.tenant_id
    FROM public.email_verification_tokens AS ev
    JOIN public.dashboard_users AS du
      ON du.tenant_id = ev.tenant_id
     AND du.id = ev.dashboard_user_id
    WHERE ev.token_hash = p_token_hash
      AND ev.consumed_at IS NULL
      AND ev.expires_at > now()
      AND du.email_verified_at IS NULL
    LIMIT 1
$$;
ALTER FUNCTION auth_private.resolve_email_verification_token(bytea) OWNER TO meteor_auth_lookup;
REVOKE ALL ON FUNCTION auth_private.resolve_email_verification_token(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_private.resolve_email_verification_token(bytea) TO meteor_app;

-- +goose Down
DROP FUNCTION auth_private.resolve_email_verification_token(bytea);
DROP FUNCTION auth_private.resolve_dashboard_session(bytea);
DROP FUNCTION auth_private.resolve_github_identity(text);
DROP FUNCTION auth_private.resolve_dashboard_login(text);
DROP SCHEMA auth_private;

REVOKE SELECT ON dashboard_users, user_identities, dashboard_sessions, email_verification_tokens
    FROM meteor_auth_lookup;
REVOKE USAGE ON SCHEMA public FROM meteor_auth_lookup;
DROP ROLE meteor_auth_lookup;
