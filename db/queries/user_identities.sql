-- name: CreateUserIdentity :one
INSERT INTO user_identities (tenant_id, dashboard_user_id, provider, provider_subject)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, dashboard_user_id, provider, provider_subject, created_at;

-- name: ListUserIdentities :many
SELECT id, tenant_id, dashboard_user_id, provider, provider_subject, created_at
FROM user_identities
WHERE tenant_id = $1
  AND dashboard_user_id = $2
ORDER BY created_at, id;
