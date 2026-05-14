// Package server provides HTTP handlers for the MCP proxy server.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Handler holds HTTP handler dependencies.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new Handler instance.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// HandleTools handles tool listing and execution.
func (h *Handler) HandleTools(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handleTools called", slog.String("method", r.Method))

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Method string `json:"method"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
