-- name: CreateTenant :one
INSERT INTO tenants (id, name)
VALUES ($1, $2)
RETURNING id, name, status, last_login_at, created_at;
