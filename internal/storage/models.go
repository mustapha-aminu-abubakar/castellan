package storage

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type APIKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	KeyHash   string     `json:"-"`
	Label     *string    `json:"label,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Provider struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"base_url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Endpoint struct {
	ID                string    `json:"id"`
	ProviderID        string    `json:"provider_id"`
	Route             string    `json:"route"`
	Method            string    `json:"method"`
	PriceAmount       float64   `json:"price_amount"`
	Currency          string    `json:"currency"`
	RateLimit         *int      `json:"rate_limit,omitempty"`
	Description       *string   `json:"description,omitempty"`
	OpenAPIOperation  *string   `json:"openapi_operation_id,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type BulkImportRequest struct {
	Endpoints   []BulkEndpoint `json:"endpoints"`
	AutoPublish bool           `json:"auto_publish"`
	DefaultPrice float64       `json:"default_price"`
}

type BulkEndpoint struct {
	Route       string  `json:"route"`
	Method      string  `json:"method"`
	Description *string `json:"description,omitempty"`
	OperationID *string `json:"operation_id,omitempty"`
}

type CreateProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type UpdateProviderRequest struct {
	Name    *string `json:"name,omitempty"`
	BaseURL *string `json:"base_url,omitempty"`
}

type UpdateEndpointRequest struct {
	PriceAmount *float64 `json:"price_amount,omitempty"`
	RateLimit   *int     `json:"rate_limit,omitempty"`
	Status      *string  `json:"status,omitempty"`
}

type StatusUpdateRequest struct {
	Status string `json:"status"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
