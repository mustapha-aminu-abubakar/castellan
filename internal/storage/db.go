package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://flowgate:flowgate@localhost:5432/flowgate?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		migrationEnums,
		migrationUsers,
		migrationAPIKeys,
		migrationProviders,
		migrationEndpoints,
		migrationAccounts,
		migrationStellarWallets,
		migrationLedgerEntries,
		migrationUsageEvents,
		migrationDeposits,
		migrationSettlementBatches,
		migrationSettlementEntries,
	}

	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

const migrationEnums = `
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('provider', 'consumer', 'both', 'admin');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE api_key_status AS ENUM ('active', 'revoked', 'expired');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE endpoint_status AS ENUM ('draft', 'active', 'inactive');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE currency AS ENUM ('XLM', 'USDC');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
`

const migrationUsers = `
CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    role       user_role NOT NULL DEFAULT 'consumer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
`

const migrationAPIKeys = `
CREATE TABLE IF NOT EXISTS api_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash   TEXT NOT NULL,
    label      TEXT,
    status     api_key_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys (user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
`

const migrationProviders = `
CREATE TABLE IF NOT EXISTS providers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL,
    status     provider_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_providers_owner ON providers (owner_id);
`

const migrationEndpoints = `
CREATE TABLE IF NOT EXISTS api_endpoints (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id          UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    route                TEXT NOT NULL,
    method               TEXT NOT NULL DEFAULT 'GET',
    price_amount         NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency             currency NOT NULL DEFAULT 'XLM',
    rate_limit           INT,
    description          TEXT,
    openapi_operation_id TEXT,
    status               endpoint_status NOT NULL DEFAULT 'draft',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);
CREATE INDEX IF NOT EXISTS idx_endpoints_provider ON api_endpoints (provider_id);
`

const migrationAccounts = `
CREATE TABLE IF NOT EXISTS accounts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance    NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency   currency NOT NULL DEFAULT 'XLM',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_accounts_owner ON accounts (owner_id);
`

const migrationStellarWallets = `
CREATE TABLE IF NOT EXISTS stellar_wallets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL,
    public_key TEXT NOT NULL,
    memo       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

const migrationLedgerEntries = `
CREATE TABLE IF NOT EXISTS ledger_entries (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id     UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    entry_type     TEXT NOT NULL,
    amount         NUMERIC(20,10) NOT NULL,
    balance_after  NUMERIC(20,10) NOT NULL,
    currency       currency NOT NULL DEFAULT 'XLM',
    reference_id   UUID,
    reference_type TEXT,
    status         TEXT NOT NULL DEFAULT 'completed',
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ledger_account ON ledger_entries (account_id);
`

const migrationUsageEvents = `
CREATE TABLE IF NOT EXISTS usage_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id   UUID NOT NULL REFERENCES users(id),
    provider_id   UUID NOT NULL REFERENCES providers(id),
    endpoint_id   UUID NOT NULL REFERENCES api_endpoints(id),
    request_cost  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    status_code   INT,
    latency_ms    INT,
    response_size INT,
    request_id    TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_usage_consumer ON usage_events (consumer_id);
CREATE INDEX IF NOT EXISTS idx_usage_provider ON usage_events (provider_id);
CREATE INDEX IF NOT EXISTS idx_usage_endpoint ON usage_events (endpoint_id);
`

const migrationDeposits = `
CREATE TABLE IF NOT EXISTS deposits (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    from_address TEXT NOT NULL,
    amount       NUMERIC(20,10) NOT NULL,
    currency     currency NOT NULL DEFAULT 'XLM',
    memo         TEXT,
    tx_hash      TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ
);
`

const migrationSettlementBatches = `
CREATE TABLE IF NOT EXISTS settlement_batches (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status       TEXT NOT NULL DEFAULT 'pending',
    total_amount NUMERIC(20,10) NOT NULL,
    currency     currency NOT NULL DEFAULT 'XLM',
    entry_count  INT NOT NULL DEFAULT 0,
    tx_hash      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
`

const migrationSettlementEntries = `
CREATE TABLE IF NOT EXISTS settlement_entries (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id       UUID NOT NULL REFERENCES settlement_batches(id) ON DELETE CASCADE,
    provider_id    UUID NOT NULL REFERENCES providers(id),
    amount         NUMERIC(20,10) NOT NULL,
    currency       currency NOT NULL DEFAULT 'XLM',
    wallet_address TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
