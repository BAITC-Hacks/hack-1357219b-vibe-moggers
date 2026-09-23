package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type databasePinger interface {
	PingContext(context.Context) error
}

type Server struct {
	db     databasePinger
	logger *slog.Logger
}

func New(db databasePinger, logger *slog.Logger, register ...func(*http.ServeMux)) http.Handler {
	server := &Server{db: db, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", server.health)
	mux.HandleFunc("GET /api/health", server.health)
	for _, routes := range register {
		routes(mux)
	}

	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK
	response := map[string]string{
		"status":   "ok",
		"database": "connected",
	}

	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(pingCtx); err != nil {
		statusCode = http.StatusServiceUnavailable
		response["status"] = "degraded"
		response["database"] = "unavailable"
		s.logger.Error("database health check failed", "error", err)
	}

	writeJSON(w, statusCode, response)
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
