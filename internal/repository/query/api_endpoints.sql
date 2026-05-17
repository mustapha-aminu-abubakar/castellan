-- name: GetEndpointByID :one
SELECT * FROM api_endpoints WHERE id = $1;

-- name: GetEndpointByProviderRouteMethod :one
SELECT
    ae.*,
    p.owner_id AS provider_owner_id,
    p.base_url AS provider_base_url
FROM api_endpoints ae
JOIN providers p ON p.id = ae.provider_id
WHERE ae.provider_id = $1 AND ae.route = $2 AND ae.method = $3 AND ae.status = 'active';

-- name: ListEndpointsByProvider :many
SELECT * FROM api_endpoints WHERE provider_id = $1 ORDER BY route ASC;

-- name: CreateEndpoint :one
INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: UpdateEndpoint :exec
UPDATE api_endpoints
SET price_amount = $3, rate_limit = $4, status = $5, updated_at = now()
WHERE id = $1 AND provider_id = $2;

-- name: DeleteEndpoint :exec
DELETE FROM api_endpoints WHERE id = $1 AND provider_id = $2;
