-- name: CreateSettlementBatch :one
INSERT INTO settlement_batches (status, total_amount, currency, entry_count)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetSettlementBatchByID :one
SELECT * FROM settlement_batches WHERE id = $1;

-- name: UpdateSettlementBatch :exec
UPDATE settlement_batches
SET status = $2, tx_hash = $3, completed_at = now()
WHERE id = $1;

-- name: ListSettlementBatches :many
SELECT * FROM settlement_batches
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListSettlementBatchesByProvider :many
SELECT DISTINCT sb.*
FROM settlement_batches sb
JOIN settlement_entries se ON se.batch_id = sb.id
WHERE se.provider_id = $1
ORDER BY sb.created_at DESC
LIMIT $2 OFFSET $3;
