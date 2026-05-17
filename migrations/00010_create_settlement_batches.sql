-- +goose Up
CREATE TABLE settlement_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status        batch_status NOT NULL DEFAULT 'pending',
    total_amount  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    entry_count   INT NOT NULL DEFAULT 0,
    tx_hash       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX idx_settlement_batches_status ON settlement_batches (status);
CREATE INDEX idx_settlement_batches_created ON settlement_batches (created_at);

-- +goose Down
DROP TABLE IF EXISTS settlement_batches;
