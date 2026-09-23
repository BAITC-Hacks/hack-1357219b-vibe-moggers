package app

import (
	"database/sql"
	"log/slog"
	"net/http"
	"vibe-moggers/backend/internal/ai"
	"vibe-moggers/backend/internal/config"
	"vibe-moggers/backend/internal/offers"
	"vibe-moggers/backend/internal/server"
	"vibe-moggers/backend/internal/tasks"
	"vibe-moggers/backend/internal/teams"
)

func New(db *sql.DB, cfg config.Config, logger *slog.Logger) http.Handler {
	return server.NewConfigured(db, logger, cfg.AIMode, cfg.DemoMode,
		teams.NewHandler(db, logger).Register,
		tasks.NewHandler(tasks.NewStore(db), logger).Register,
		offers.NewHandler(offers.NewStore(db, tasks.SQLAccess{}), logger).Register,
		ai.New(cfg.AIMode, cfg.OpenAIKey, cfg.OpenAIModel).Register(logger))
}
