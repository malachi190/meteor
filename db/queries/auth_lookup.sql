-- name: ResolveDashboardLogin :one
SELECT dashboard_user_id::uuid AS dashboard_user_id,
       tenant_id::uuid AS tenant_id,
       COALESCE(password_hash, '')::text AS password_hash,
       email_verified_at::timestamptz AS email_verified_at
FROM auth_private.resolve_dashboard_login($1)
    AS lookup(dashboard_user_id, tenant_id, password_hash, email_verified_at);

-- name: ResolveGitHubIdentity :one
SELECT dashboard_user_id::uuid AS dashboard_user_id,
       tenant_id::uuid AS tenant_id
FROM auth_private.resolve_github_identity($1) AS lookup(dashboard_user_id, tenant_id);

-- name: ResolveEmailVerificationToken :one
SELECT dashboard_user_id::uuid AS dashboard_user_id,
       tenant_id::uuid AS tenant_id
FROM auth_private.resolve_email_verification_token($1) AS lookup(dashboard_user_id, tenant_id);

-- name: ResolvePasswordResetToken :one
SELECT dashboard_user_id::uuid AS dashboard_user_id,
       tenant_id::uuid AS tenant_id
FROM auth_private.resolve_password_reset_token($1) AS lookup(dashboard_user_id, tenant_id);
