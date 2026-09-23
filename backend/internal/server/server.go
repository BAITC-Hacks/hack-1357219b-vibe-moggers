package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
	"vibe-moggers/backend/internal/httpapi"
)

type databasePinger interface {
	PingContext(context.Context) error
}

type Server struct {
	db     databasePinger
	logger *slog.Logger
	aiMode string
}

func New(db databasePinger, logger *slog.Logger, register ...func(*http.ServeMux)) http.Handler {
	return NewConfigured(db, logger, "fallback", true, register...)
}
func NewConfigured(db databasePinger, logger *slog.Logger, aiMode string, demoMode bool, register ...func(*http.ServeMux)) http.Handler {
	server := &Server{db: db, logger: logger, aiMode: aiMode}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", server.health)
	mux.HandleFunc("GET /api/health", server.health)
	for _, routes := range register {
		routes(mux)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !demoMode && r.URL.Path != "/health" && r.URL.Path != "/api/health" {
			httpapi.Fail(w, logger, httpapi.Problem(503, "demo_disabled", "Demo identity is disabled. Production authentication is not implemented."))
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK
	response := map[string]string{
		"status":   "ok",
		"database": "connected",
		"aiMode":   s.aiMode,
	}

	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(pingCtx); err != nil {
		statusCode = http.StatusServiceUnavailable
		response["status"] = "degraded"
		response["database"] = "unavailable"
		s.logger.Error("database health check failed", "error", err)
	}

	if statusCode != 200 {
		httpapi.Fail(w, s.logger, httpapi.Problem(statusCode, "database_unavailable", "Database is unavailable."))
		return
	}
	httpapi.Data(w, statusCode, response)
}
