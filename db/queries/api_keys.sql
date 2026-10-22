-- name: CreateAPIKey :one
INSERT INTO api_keys (tenant_id, project_id, prefix, key_hash, env)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, project_id, prefix, env, created_at, last_used_at, revoked_at;

-- name: ListAPIKeysByTenantProject :many
SELECT id, tenant_id, project_id, prefix, env, created_at, last_used_at, revoked_at
FROM api_keys
WHERE tenant_id = $1
  AND project_id = $2
ORDER BY created_at DESC, id;

-- name: RevokeAPIKey :execrows
UPDATE api_keys
SET revoked_at = now()
WHERE tenant_id = $1
  AND project_id = $2
  AND id = $3
  AND revoked_at IS NULL;
