-- name: CreateEmailVerificationToken :exec
INSERT INTO email_verification_tokens (token_hash, tenant_id, dashboard_user_id, expires_at)
VALUES ($1, $2, $3, $4);

-- name: ConsumePriorEmailVerificationTokens :execrows
UPDATE email_verification_tokens
SET consumed_at = now()
WHERE tenant_id = $1
  AND dashboard_user_id = $2
  AND consumed_at IS NULL;

-- name: ConsumeEmailVerificationToken :one
UPDATE email_verification_tokens
SET consumed_at = now()
WHERE tenant_id = $1
  AND token_hash = $2
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING dashboard_user_id;
