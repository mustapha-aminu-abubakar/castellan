-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByDepositMemo :one
SELECT * FROM users WHERE deposit_memo = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, role) VALUES ($1, $2) RETURNING *;

-- name: UpdateUserPayoutAddress :exec
UPDATE users SET payout_stellar_address = $2, updated_at = now() WHERE id = $1;

-- name: SetUserDepositMemo :exec
UPDATE users SET deposit_memo = $2, updated_at = now() WHERE id = $1;
