-- name: GetAccountByOwnerID :one
SELECT * FROM accounts WHERE owner_id = $1;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: CreateAccount :one
INSERT INTO accounts (owner_id, currency) VALUES ($1, $2) RETURNING *;

-- name: DeductAccountBalance :one
UPDATE accounts
SET balance = balance - sqlc.arg(amount), updated_at = now()
WHERE id = sqlc.arg(id) AND balance >= sqlc.arg(amount)
RETURNING *;

-- name: CreditAccountBalance :one
UPDATE accounts
SET balance = balance + sqlc.arg(amount), updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;
