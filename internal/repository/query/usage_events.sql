-- name: CreateUsageEvent :one
INSERT INTO usage_events (
    consumer_id, provider_id, endpoint_id, request_cost, currency,
    status_code, latency_ms, response_size, request_id, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetUsageEventByRequestID :one
SELECT * FROM usage_events WHERE request_id = $1;

-- name: ListUsageEventsByConsumer :many
SELECT
    ue.*,
    p.name AS provider_name,
    ae.route AS endpoint_route
FROM usage_events ue
JOIN providers p ON p.id = ue.provider_id
JOIN api_endpoints ae ON ae.id = ue.endpoint_id
WHERE ue.consumer_id = $1
ORDER BY ue.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUsageEventsByProvider :many
SELECT
    ue.*,
    u.email AS consumer_email
FROM usage_events ue
JOIN users u ON u.id = ue.consumer_id
WHERE ue.provider_id = $1
ORDER BY ue.created_at DESC
LIMIT $2 OFFSET $3;

-- name: AggregateProviderEarnings :many
SELECT
    ue.provider_id,
    SUM(ue.request_cost)::numeric AS total_earned,
    COUNT(*)::bigint AS total_requests
FROM usage_events ue
WHERE ue.status = 'completed' AND ue.created_at < sqlc.arg(cutoff_time)
GROUP BY ue.provider_id
HAVING SUM(ue.request_cost) > 0;

-- name: MarkUsageEventsAsSettled :exec
UPDATE usage_events SET status = sqlc.arg(new_status)
WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- name: GetConsumerUsageSummary :many
SELECT
    p.name AS provider_name,
    COUNT(*)::bigint AS total_requests,
    SUM(ue.request_cost)::numeric AS total_spent
FROM usage_events ue
JOIN providers p ON p.id = ue.provider_id
WHERE ue.consumer_id = $1
GROUP BY p.name
ORDER BY total_spent DESC;
