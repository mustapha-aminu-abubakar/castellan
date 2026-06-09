package server

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flowgate/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/julienschmidt/httprouter"
)

type mockQuerier struct {
	repository.Querier
	createProviderFn          func(ctx context.Context, arg repository.CreateProviderParams) (repository.Provider, error)
	getProviderByIDFn         func(ctx context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error)
	listProvidersByOwnerFn    func(ctx context.Context, ownerID uuid.UUID) ([]repository.Provider, error)
	updateProviderFn          func(ctx context.Context, arg repository.UpdateProviderParams) error
	updateUserPayoutAddressFn func(ctx context.Context, arg repository.UpdateUserPayoutAddressParams) error
}

func (m *mockQuerier) CreateProvider(ctx context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
	return m.createProviderFn(ctx, arg)
}

func (m *mockQuerier) GetProviderByID(ctx context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
	return m.getProviderByIDFn(ctx, id)
}

func (m *mockQuerier) ListProvidersByOwner(ctx context.Context, ownerID uuid.UUID) ([]repository.Provider, error) {
	return m.listProvidersByOwnerFn(ctx, ownerID)
}

func (m *mockQuerier) UpdateProvider(ctx context.Context, arg repository.UpdateProviderParams) error {
	return m.updateProviderFn(ctx, arg)
}

func (m *mockQuerier) UpdateUserPayoutAddress(ctx context.Context, arg repository.UpdateUserPayoutAddressParams) error {
	return m.updateUserPayoutAddressFn(ctx, arg)
}

func testRouter(h *Handlers) http.Handler {
	r := httprouter.New()
	r.HandlerFunc(http.MethodPost, "/api/v1/providers", h.CreateProvider)
	r.HandlerFunc(http.MethodGet, "/api/v1/providers", h.ListProviders)
	r.HandlerFunc(http.MethodGet, "/api/v1/providers/:id", h.GetProvider)
	r.HandlerFunc(http.MethodPatch, "/api/v1/providers/:id", h.UpdateProvider)
	return r
}

func validTestAddress() string {
	payload := make([]byte, 33, 35)
	payload[0] = stellarVersionByte
	for i := 1; i < 33; i++ {
		payload[i] = byte(i)
	}
	c := crc16XModem(payload)
	payload = append(payload, byte(c>>8), byte(c))
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
}

const (
	baseURLKey       = "base_url"
	stellarAddrKey   = "payout_stellar_address"
	updatedAPIName   = "Updated API"
	testProviderName = "My API"
	testBaseURL      = "https://api.example.com"
)

var (
	testOwnerID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	testProvID  = uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	testTime    = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	testAddr    = validTestAddress()
)

func TestCreateProvider_Success(t *testing.T) {
	mock := &mockQuerier{
		createProviderFn: func(_ context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
			return repository.Provider{
				ID:        testProvID,
				OwnerID:   arg.OwnerID,
				Name:      arg.Name,
				BaseUrl:   arg.BaseUrl,
				Status:    repository.ProviderStatusActive,
				CreatedAt: testTime,
				UpdatedAt: testTime,
			}, nil
		},
		updateUserPayoutAddressFn: func(_ context.Context, _ repository.UpdateUserPayoutAddressParams) error {
			return nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"owner_id":     testOwnerID.String(),
		"name":         testProviderName,
		baseURLKey:     testBaseURL,
		stellarAddrKey: testAddr,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var provider repository.Provider
	if err := json.NewDecoder(resp.Body).Decode(&provider); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if provider.Name != testProviderName {
		t.Errorf("expected name 'My API', got %q", provider.Name)
	}
	if provider.Status != repository.ProviderStatusActive {
		t.Errorf("expected status 'active', got %q", provider.Status)
	}
}

func TestCreateProvider_MissingFields(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing owner_id", map[string]string{"name": "x", baseURLKey: "x", stellarAddrKey: "G"}},
		{"missing name", map[string]string{"owner_id": testOwnerID.String(), baseURLKey: "x", stellarAddrKey: "G"}},
		{"missing base_url", map[string]string{"owner_id": testOwnerID.String(), "name": "x", stellarAddrKey: "G"}},
		{"missing payout", map[string]string{"owner_id": testOwnerID.String(), "name": "x", baseURLKey: "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("failed to marshal body: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestCreateProvider_InvalidStellarAddress(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"owner_id":     testOwnerID.String(),
		"name":         testProviderName,
		baseURLKey:     testBaseURL,
		stellarAddrKey: "not-a-stellar-address",
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid stellar address, got %d", resp.StatusCode)
	}
}

func TestGetProvider_Success(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+testProvID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var row repository.GetProviderByIDRow
	if err := json.NewDecoder(resp.Body).Decode(&row); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if row.Name != testProviderName {
		t.Errorf("expected name 'My API', got %q", row.Name)
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, _ uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{}, pgx.ErrNoRows
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/660e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetProvider_InvalidID(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/not-a-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListProviders_Success(t *testing.T) {
	expected := []repository.Provider{
		{
			ID:        testProvID,
			OwnerID:   testOwnerID,
			Name:      testProviderName,
			BaseUrl:   testBaseURL,
			Status:    repository.ProviderStatusActive,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		},
	}
	mock := &mockQuerier{
		listProvidersByOwnerFn: func(_ context.Context, _ uuid.UUID) ([]repository.Provider, error) {
			return expected, nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers?owner_id="+testOwnerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var providers []repository.Provider
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].Name != testProviderName {
		t.Errorf("expected name 'My API', got %q", providers[0].Name)
	}
}

func TestListProviders_Empty(t *testing.T) {
	mock := &mockQuerier{
		listProvidersByOwnerFn: func(_ context.Context, _ uuid.UUID) ([]repository.Provider, error) {
			return nil, nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers?owner_id="+testOwnerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "[]\n" && string(body) != "[]" {
		t.Errorf("expected empty array, got %s", string(body))
	}
}

func TestListProviders_MissingOwnerID(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_Success(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
		updateProviderFn: func(_ context.Context, arg repository.UpdateProviderParams) error {
			if arg.Name != updatedAPIName {
				t.Errorf("expected name 'Updated API', got %q", arg.Name)
			}
			if arg.BaseUrl != testBaseURL {
				t.Errorf("expected base_url preserved, got %q", arg.BaseUrl)
			}
			return nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"name": updatedAPIName}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_WithStellarAddress(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
		updateProviderFn: func(_ context.Context, _ repository.UpdateProviderParams) error {
			return nil
		},
		updateUserPayoutAddressFn: func(_ context.Context, _ repository.UpdateUserPayoutAddressParams) error {
			return nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"name":         updatedAPIName,
		stellarAddrKey: testAddr,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_InvalidStellarAddress(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{stellarAddrKey: "bad-address"}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid stellar address, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_NotFound(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, _ uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{}, pgx.ErrNoRows
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"name": updatedAPIName}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestValidProviderStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{"active", "active", true},
		{"inactive", "inactive", true},
		{"suspended", "suspended", true},
		{"bogus", "bogus", false},
		{"empty", "", false},
		{"mixed case", "Active", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validProviderStatus(tt.status); got != tt.valid {
				t.Errorf("validProviderStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestWriteJSON_EncodeError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, make(chan int))

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestCreateProvider_InvalidJSON(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader([]byte(`{invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateProvider_InvalidOwnerID(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"owner_id":     "not-a-uuid",
		"name":         testProviderName,
		baseURLKey:     testBaseURL,
		stellarAddrKey: testAddr,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateProvider_DBError(t *testing.T) {
	mock := &mockQuerier{
		createProviderFn: func(_ context.Context, _ repository.CreateProviderParams) (repository.Provider, error) {
			return repository.Provider{}, pgx.ErrNoRows
		},
		updateUserPayoutAddressFn: func(_ context.Context, _ repository.UpdateUserPayoutAddressParams) error {
			return nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"owner_id":     testOwnerID.String(),
		"name":         testProviderName,
		baseURLKey:     testBaseURL,
		stellarAddrKey: testAddr,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestCreateProvider_PayoutError(t *testing.T) {
	mock := &mockQuerier{
		createProviderFn: func(_ context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
			return repository.Provider{
				ID:        testProvID,
				OwnerID:   arg.OwnerID,
				Name:      arg.Name,
				BaseUrl:   arg.BaseUrl,
				Status:    repository.ProviderStatusActive,
				CreatedAt: testTime,
				UpdatedAt: testTime,
			}, nil
		},
		updateUserPayoutAddressFn: func(_ context.Context, _ repository.UpdateUserPayoutAddressParams) error {
			return pgx.ErrNoRows
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{
		"owner_id":     testOwnerID.String(),
		"name":         testProviderName,
		baseURLKey:     testBaseURL,
		stellarAddrKey: testAddr,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestGetProvider_DBError(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, _ uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{}, pgx.ErrTxClosed
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+testProvID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestListProviders_InvalidOwnerID(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers?owner_id=not-a-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListProviders_DBError(t *testing.T) {
	mock := &mockQuerier{
		listProvidersByOwnerFn: func(_ context.Context, _ uuid.UUID) ([]repository.Provider, error) {
			return nil, pgx.ErrTxClosed
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers?owner_id="+testOwnerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_InvalidID(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"name": updatedAPIName}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/not-a-uuid", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_InvalidJSON(t *testing.T) {
	mock := &mockQuerier{}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader([]byte(`{invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_GetDBError(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, _ uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{}, pgx.ErrTxClosed
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"name": updatedAPIName}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_InvalidStatus(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"status": "bogus"}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_UpdateError(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
		updateProviderFn: func(_ context.Context, _ repository.UpdateProviderParams) error {
			return pgx.ErrNoRows
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{"name": updatedAPIName}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestUpdateProvider_PayoutUpdateError(t *testing.T) {
	mock := &mockQuerier{
		getProviderByIDFn: func(_ context.Context, id uuid.UUID) (repository.GetProviderByIDRow, error) {
			return repository.GetProviderByIDRow{
				ID:      id,
				OwnerID: testOwnerID,
				Name:    testProviderName,
				BaseUrl: testBaseURL,
				Status:  repository.ProviderStatusActive,
			}, nil
		},
		updateProviderFn: func(_ context.Context, _ repository.UpdateProviderParams) error {
			return nil
		},
		updateUserPayoutAddressFn: func(_ context.Context, _ repository.UpdateUserPayoutAddressParams) error {
			return pgx.ErrNoRows
		},
	}
	h := NewHandlers(mock, nil)
	router := testRouter(h)

	body := map[string]string{stellarAddrKey: testAddr}
	reqBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+testProvID.String(), bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestValidateStellarAddress(t *testing.T) {
	tests := []struct {
		name  string
		addr  string
		valid bool
	}{
		{
			name:  "valid generated address",
			addr:  testAddr,
			valid: true,
		},
		{
			name:  "wrong prefix",
			addr:  "A" + testAddr[1:],
			valid: false,
		},
		{
			name:  "too short",
			addr:  "GBZXN7",
			valid: false,
		},
		{
			name:  "empty string",
			addr:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateStellarAddress(tt.addr); got != tt.valid {
				t.Errorf("validateStellarAddress(%q) = %v, want %v", tt.addr, got, tt.valid)
			}
		})
	}
}
