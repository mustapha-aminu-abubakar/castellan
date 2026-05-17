-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (
    account_id, entry_type, amount, balance_after,
    currency, reference_id, reference_type, status, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: ListLedgerEntriesByAccount :many
SELECT * FROM ledger_entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLastLedgerEntry :one
SELECT * FROM ledger_entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT 1;
