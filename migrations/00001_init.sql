-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('provider', 'consumer', 'both', 'admin');
CREATE TYPE api_key_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE endpoint_status AS ENUM ('active', 'inactive');
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE settlement_entry_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE deposit_status AS ENUM ('pending', 'confirmed', 'failed');

-- +goose Down
DROP TYPE IF EXISTS deposit_status;
DROP TYPE IF EXISTS settlement_entry_status;
DROP TYPE IF EXISTS batch_status;
DROP TYPE IF EXISTS usage_status;
DROP TYPE IF EXISTS ledger_status;
DROP TYPE IF EXISTS entry_type;
DROP TYPE IF EXISTS currency;
DROP TYPE IF EXISTS endpoint_status;
DROP TYPE IF EXISTS provider_status;
DROP TYPE IF EXISTS api_key_status;
DROP TYPE IF EXISTS user_role;
