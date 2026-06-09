package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
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
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", slog.Any("error", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
