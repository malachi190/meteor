-- name: UpdateTenantLastLogin :execrows
UPDATE tenants
SET last_login_at = now()
WHERE id = $1;
