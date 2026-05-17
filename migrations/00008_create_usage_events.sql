-- +goose Up
CREATE TABLE usage_events (
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
    status        usage_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_consumer ON usage_events (consumer_id);
CREATE INDEX idx_usage_provider ON usage_events (provider_id);
CREATE INDEX idx_usage_endpoint ON usage_events (endpoint_id);
CREATE UNIQUE INDEX idx_usage_request_id ON usage_events (request_id);
CREATE INDEX idx_usage_created ON usage_events (created_at);
CREATE INDEX idx_usage_status ON usage_events (status);

-- +goose Down
DROP TABLE IF EXISTS usage_events;
