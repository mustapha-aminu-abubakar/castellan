-- name: CreateDeposit :one
INSERT INTO deposits (account_id, from_address, amount, currency, memo, tx_hash, status)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetDepositByTxHash :one
SELECT * FROM deposits WHERE tx_hash = $1;

-- name: ConfirmDeposit :exec
UPDATE deposits SET status = 'confirmed', confirmed_at = now() WHERE id = $1;

-- name: FailDeposit :exec
UPDATE deposits SET status = 'failed' WHERE id = $1;

-- name: ListDepositsByAccount :many
SELECT * FROM deposits
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
