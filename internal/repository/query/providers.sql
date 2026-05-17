-- name: GetProviderByID :one
SELECT
    p.*,
    u.payout_stellar_address
FROM providers p
JOIN users u ON u.id = p.owner_id
WHERE p.id = $1;

-- name: ListProvidersByOwner :many
SELECT * FROM providers WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: CreateProvider :one
INSERT INTO providers (owner_id, name, base_url) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateProvider :exec
UPDATE providers
SET name = $2, base_url = $3, status = $4, updated_at = now()
WHERE id = $1 AND owner_id = $5;

-- name: DeleteProvider :exec
DELETE FROM providers WHERE id = $1 AND owner_id = $2;
