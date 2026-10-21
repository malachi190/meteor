-- +goose Up
CREATE FUNCTION auth_private.resolve_password_reset_token(p_token_hash bytea)
RETURNS TABLE (dashboard_user_id uuid, tenant_id uuid)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $$
    SELECT pr.dashboard_user_id, pr.tenant_id
    FROM public.password_reset_tokens AS pr
    JOIN public.dashboard_users AS du
      ON du.tenant_id = pr.tenant_id
     AND du.id = pr.dashboard_user_id
    WHERE pr.token_hash = p_token_hash
      AND pr.consumed_at IS NULL
      AND pr.expires_at > now()
      AND du.password_hash IS NOT NULL
      AND du.email_verified_at IS NOT NULL
    LIMIT 1
$$;
ALTER FUNCTION auth_private.resolve_password_reset_token(bytea) OWNER TO meteor_auth_lookup;
REVOKE ALL ON FUNCTION auth_private.resolve_password_reset_token(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_private.resolve_password_reset_token(bytea) TO meteor_app;

-- +goose Down
DROP FUNCTION auth_private.resolve_password_reset_token(bytea);
