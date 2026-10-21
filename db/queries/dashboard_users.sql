-- name: CreateDashboardUser :one
INSERT INTO dashboard_users (tenant_id, email, normalized_email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, email, normalized_email, email_verified_at, role, created_at, last_login_at;

-- name: CreateVerifiedGitHubUser :one
INSERT INTO dashboard_users (tenant_id, email, normalized_email, password_hash, email_verified_at)
VALUES ($1, $2, $3, NULL, now())
RETURNING id, tenant_id, email, normalized_email, email_verified_at, role, created_at, last_login_at;

-- name: GetDashboardUserByTenantAndID :one
SELECT id, tenant_id, email, normalized_email, email_verified_at, role, created_at, last_login_at
FROM dashboard_users
WHERE tenant_id = $1
  AND id = $2;

-- name: MarkDashboardUserEmailVerified :one
UPDATE dashboard_users
SET email_verified_at = now()
WHERE tenant_id = $1
  AND id = $2
  AND email_verified_at IS NULL
RETURNING id, tenant_id, email, normalized_email, email_verified_at, role, created_at, last_login_at;

-- name: UpdateDashboardUserLastLogin :execrows
UPDATE dashboard_users
SET last_login_at = now()
WHERE tenant_id = $1
  AND id = $2;
