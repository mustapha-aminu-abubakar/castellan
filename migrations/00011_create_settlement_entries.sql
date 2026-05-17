-- +goose Up
CREATE TABLE settlement_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL REFERENCES settlement_batches(id) ON DELETE CASCADE,
    provider_id     UUID NOT NULL REFERENCES providers(id),
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    wallet_address  TEXT NOT NULL,
    status          settlement_entry_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlement_entries_batch ON settlement_entries (batch_id);
CREATE INDEX idx_settlement_entries_provider ON settlement_entries (provider_id);
CREATE INDEX idx_settlement_entries_status ON settlement_entries (status);

-- +goose Down
DROP TABLE IF EXISTS settlement_entries;
