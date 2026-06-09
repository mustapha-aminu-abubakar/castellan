package server

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"flowgate/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/julienschmidt/httprouter"
)

func (h *Handlers) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OwnerID              string `json:"owner_id"`
		Name                 string `json:"name"`
		BaseURL              string `json:"base_url"`
		PayoutStellarAddress string `json:"payout_stellar_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.ErrorContext(r.Context(), "invalid request body", slog.Any("error", err))
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if input.OwnerID == "" || input.Name == "" || input.BaseURL == "" || input.PayoutStellarAddress == "" {
		writeError(w, http.StatusBadRequest, "owner_id, name, base_url, and payout_stellar_address are required")
		return
	}

	ownerID, err := uuid.Parse(input.OwnerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid owner_id: must be a valid UUID")
		return
	}

	if !validateStellarAddress(input.PayoutStellarAddress) {
		writeError(w, http.StatusBadRequest, "invalid payout_stellar_address: must be a valid Stellar ed25519 public key")
		return
	}

	provider, err := h.q.CreateProvider(r.Context(), repository.CreateProviderParams{
		OwnerID: ownerID,
		Name:    input.Name,
		BaseUrl: input.BaseURL,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create provider", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create provider")
		return
	}

	if err := h.q.UpdateUserPayoutAddress(r.Context(), repository.UpdateUserPayoutAddressParams{
		ID:                   ownerID,
		PayoutStellarAddress: pgtype.Text{String: input.PayoutStellarAddress, Valid: true},
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to update payout address", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to update payout address")
		return
	}

	slog.DebugContext(r.Context(), "provider created", slog.String("provider_id", provider.ID.String()))
	writeJSON(w, http.StatusCreated, provider)
}

func (h *Handlers) GetProvider(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	providerID, err := uuid.Parse(params.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	provider, err := h.q.GetProviderByID(r.Context(), providerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get provider", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (h *Handlers) ListProviders(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		writeError(w, http.StatusBadRequest, "owner_id query parameter is required")
		return
	}

	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid owner_id")
		return
	}

	providers, err := h.q.ListProvidersByOwner(r.Context(), ownerID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list providers", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}

	if providers == nil {
		providers = []repository.Provider{}
	}

	writeJSON(w, http.StatusOK, providers)
}

func (h *Handlers) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	providerID, err := uuid.Parse(params.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	var input struct {
		Name                 *string `json:"name,omitempty"`
		BaseURL              *string `json:"base_url,omitempty"`
		Status               *string `json:"status,omitempty"`
		PayoutStellarAddress *string `json:"payout_stellar_address,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	existing, err := h.q.GetProviderByID(r.Context(), providerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get provider", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	name := existing.Name
	baseURL := existing.BaseUrl
	status := existing.Status
	if input.Name != nil {
		name = *input.Name
	}
	if input.BaseURL != nil {
		baseURL = *input.BaseURL
	}
	if input.Status != nil {
		status = repository.ProviderStatus(*input.Status)
	}

	if input.PayoutStellarAddress != nil && !validateStellarAddress(*input.PayoutStellarAddress) {
		writeError(w, http.StatusBadRequest, "invalid payout_stellar_address: must be a valid Stellar ed25519 public key")
		return
	}

	if err := h.q.UpdateProvider(r.Context(), repository.UpdateProviderParams{
		ID:      providerID,
		Name:    name,
		BaseUrl: baseURL,
		Status:  status,
		OwnerID: existing.OwnerID,
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to update provider", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}

	if input.PayoutStellarAddress != nil {
		if err := h.q.UpdateUserPayoutAddress(r.Context(), repository.UpdateUserPayoutAddressParams{
			ID:                   existing.OwnerID,
			PayoutStellarAddress: pgtype.Text{String: *input.PayoutStellarAddress, Valid: true},
		}); err != nil {
			slog.ErrorContext(r.Context(), "failed to update payout address", slog.Any("error", err))
			writeError(w, http.StatusInternalServerError, "failed to update payout address")
			return
		}
	}

	updated, err := h.q.GetProviderByID(r.Context(), providerID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get updated provider", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get updated provider")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

const (
	stellarVersionByte = 0x30
	crcPolynomial      = 0x1021
)

func validateStellarAddress(addr string) bool {
	const expectedLen = 56
	if len(addr) != expectedLen || addr[0] != 'G' {
		return false
	}

	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(addr)
	if err != nil || len(decoded) != 35 {
		return false
	}

	if decoded[0] != stellarVersionByte {
		return false
	}

	checksum := uint16(decoded[33])<<8 | uint16(decoded[34])
	return crc16XModem(decoded[:33]) == checksum
}

func crc16XModem(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ crcPolynomial
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
