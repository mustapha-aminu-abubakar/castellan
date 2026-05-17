-- +goose Up
CREATE TABLE api_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    route           TEXT NOT NULL,
    method          TEXT NOT NULL DEFAULT 'GET',
    price_amount    NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    rate_limit      INT,
    status          endpoint_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);

CREATE INDEX idx_endpoints_provider ON api_endpoints (provider_id);
CREATE INDEX idx_endpoints_status ON api_endpoints (status);

-- +goose Down
DROP TABLE IF EXISTS api_endpoints;
