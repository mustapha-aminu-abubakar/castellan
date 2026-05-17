-- name: GetAPIKeyByHash :one
SELECT
    ak.*,
    u.id AS user_id,
    u.email AS user_email,
    u.role AS user_role
FROM api_keys ak
JOIN users u ON u.id = ak.user_id
WHERE ak.key_hash = $1 AND ak.status = 'active';

-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, label, expires_at)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC;

-- name: RevokeAPIKey :exec
UPDATE api_keys SET status = 'revoked' WHERE id = $1 AND user_id = $2;
