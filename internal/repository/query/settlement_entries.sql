-- name: CreateSettlementEntry :one
INSERT INTO settlement_entries (batch_id, provider_id, amount, currency, wallet_address, status)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: ListSettlementEntriesByBatch :many
SELECT
    se.*,
    p.name AS provider_name
FROM settlement_entries se
JOIN providers p ON p.id = se.provider_id
WHERE se.batch_id = $1
ORDER BY se.created_at ASC;

-- name: ListSettlementEntriesByProvider :many
SELECT * FROM settlement_entries
WHERE provider_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateSettlementEntryStatus :exec
UPDATE settlement_entries SET status = $2 WHERE id = $1;
