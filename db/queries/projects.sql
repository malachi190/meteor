-- name: CreateProject :one
INSERT INTO projects (tenant_id, name)
VALUES ($1, $2)
RETURNING id, tenant_id, name, created_at;

-- name: ListProjectsByTenant :many
SELECT id, tenant_id, name, created_at
FROM projects
WHERE tenant_id = $1
ORDER BY created_at DESC, id;

-- name: GetProjectByTenantAndID :one
SELECT id, tenant_id, name, created_at
FROM projects
WHERE tenant_id = $1
  AND id = $2;
