package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Auth ---

func (r *Repository) GetUserByAPIKey(ctx context.Context, keyHash string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.role, u.created_at, u.updated_at
		FROM users u
		JOIN api_keys ak ON ak.user_id = u.id
		WHERE ak.key_hash = $1 AND ak.status = 'active'
		AND (ak.expires_at IS NULL OR ak.expires_at > now())
	`
	u := &User{}
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired API key")
		}
		return nil, fmt.Errorf("query user by api key: %w", err)
	}
	return u, nil
}

// --- Providers ---

func (r *Repository) CreateProvider(ctx context.Context, p *Provider) error {
	query := `
		INSERT INTO providers (owner_id, name, base_url)
		VALUES ($1, $2, $3)
		RETURNING id, status, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query, p.OwnerID, p.Name, p.BaseURL).
		Scan(&p.ID, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	return nil
}

func (r *Repository) GetProvider(ctx context.Context, id string) (*Provider, error) {
	query := `
		SELECT id, owner_id, name, base_url, status, created_at, updated_at
		FROM providers WHERE id = $1
	`
	p := &Provider{}
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&p.ID, &p.OwnerID, &p.Name, &p.BaseURL, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("provider not found")
		}
		return nil, fmt.Errorf("get provider: %w", err)
	}
	return p, nil
}

func (r *Repository) ListProviders(ctx context.Context, ownerID string) ([]*Provider, error) {
	query := `
		SELECT id, owner_id, name, base_url, status, created_at, updated_at
		FROM providers WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.BaseURL, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (r *Repository) UpdateProvider(ctx context.Context, p *Provider) error {
	query := `
		UPDATE providers SET name = $1, base_url = $2, updated_at = now()
		WHERE id = $3
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, p.Name, p.BaseURL, p.ID).Scan(&p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}
	return nil
}

func (r *Repository) DeleteProvider(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM providers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	return nil
}

// --- Endpoints ---

func (r *Repository) CreateEndpoint(ctx context.Context, e *Endpoint) error {
	query := `
		INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit, description, openapi_operation_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		e.ProviderID, e.Route, e.Method, e.PriceAmount, e.Currency,
		e.RateLimit, e.Description, e.OpenAPIOperation, e.Status,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create endpoint: %w", err)
	}
	return nil
}

func (r *Repository) BulkCreateEndpoints(ctx context.Context, endpoints []*Endpoint) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bulk tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, description, openapi_operation_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (provider_id, route, method) DO UPDATE SET
			description = EXCLUDED.description,
			openapi_operation_id = EXCLUDED.openapi_operation_id,
			updated_at = now()
		RETURNING id, created_at, updated_at
	`
	for _, e := range endpoints {
		err := tx.QueryRow(ctx, query,
			e.ProviderID, e.Route, e.Method, e.PriceAmount, e.Currency,
			e.Description, e.OpenAPIOperation, e.Status,
		).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return fmt.Errorf("bulk insert endpoint %s %s: %w", e.Method, e.Route, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetEndpoint(ctx context.Context, id string) (*Endpoint, error) {
	query := `
		SELECT id, provider_id, route, method, price_amount, currency,
		       rate_limit, description, openapi_operation_id, status, created_at, updated_at
		FROM api_endpoints WHERE id = $1
	`
	e := &Endpoint{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.ProviderID, &e.Route, &e.Method, &e.PriceAmount, &e.Currency,
		&e.RateLimit, &e.Description, &e.OpenAPIOperation, &e.Status, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("endpoint not found")
		}
		return nil, fmt.Errorf("get endpoint: %w", err)
	}
	return e, nil
}

func (r *Repository) ListEndpoints(ctx context.Context, providerID string) ([]*Endpoint, error) {
	query := `
		SELECT id, provider_id, route, method, price_amount, currency,
		       rate_limit, description, openapi_operation_id, status, created_at, updated_at
		FROM api_endpoints WHERE provider_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*Endpoint
	for rows.Next() {
		e := &Endpoint{}
		if err := rows.Scan(
			&e.ID, &e.ProviderID, &e.Route, &e.Method, &e.PriceAmount, &e.Currency,
			&e.RateLimit, &e.Description, &e.OpenAPIOperation, &e.Status, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan endpoint: %w", err)
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, nil
}

func (r *Repository) UpdateEndpoint(ctx context.Context, e *Endpoint) error {
	query := `
		UPDATE api_endpoints
		SET price_amount = $1, rate_limit = $2, updated_at = now()
		WHERE id = $3
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, e.PriceAmount, e.RateLimit, e.ID).Scan(&e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update endpoint: %w", err)
	}
	return nil
}

func (r *Repository) UpdateEndpointStatus(ctx context.Context, id, status string) error {
	query := `UPDATE api_endpoints SET status = $1, updated_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update endpoint status: %w", err)
	}
	return nil
}

func (r *Repository) DeleteEndpoint(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_endpoints WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}
