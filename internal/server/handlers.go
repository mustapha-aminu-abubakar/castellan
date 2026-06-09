package server

import (
	"encoding/json"
	"net/http"

	"flowgate/internal/repository/db"
)

type Handlers struct {
	q repository.Querier
}

func NewHandlers(q repository.Querier) *Handlers {
	return &Handlers{q: q}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
