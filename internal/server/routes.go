package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"castellan/internal/repository/db"

	"github.com/julienschmidt/httprouter"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := httprouter.New()
	h := NewHandlers(repository.New(s.pool), s.pool)

	// Wrap all routes with CORS middleware
	corsWrapper := s.corsMiddleware(r)

	r.HandlerFunc(http.MethodGet, "/", s.HelloWorldHandler)

	r.HandlerFunc(http.MethodPost, "/api/v1/providers", h.CreateProvider)
	r.HandlerFunc(http.MethodGet, "/api/v1/providers", h.ListProviders)
	r.HandlerFunc(http.MethodGet, "/api/v1/providers/:id", h.GetProvider)
	r.HandlerFunc(http.MethodPatch, "/api/v1/providers/:id", h.UpdateProvider)

	return corsWrapper
}

// CORS middleware.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Use "*" for all origins, or replace with specific origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are needed

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		slog.ErrorContext(r.Context(), "error handling JSON marshal", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(jsonResp)
}
