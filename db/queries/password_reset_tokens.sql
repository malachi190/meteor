-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash, tenant_id, dashboard_user_id, expires_at)
VALUES ($1, $2, $3, $4);

-- name: ConsumePriorPasswordResetTokens :execrows
UPDATE password_reset_tokens
SET consumed_at = now()
WHERE tenant_id = $1
  AND dashboard_user_id = $2
  AND consumed_at IS NULL;

-- name: ConsumePasswordResetToken :one
UPDATE password_reset_tokens
SET consumed_at = now()
WHERE tenant_id = $1
  AND dashboard_user_id = $2
  AND token_hash = $3
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING dashboard_user_id;

-- name: UpdateDashboardPassword :execrows
UPDATE dashboard_users
SET password_hash = $3
WHERE tenant_id = $1
  AND id = $2;
