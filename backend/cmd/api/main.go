package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vibe-moggers/backend/internal/config"
	"vibe-moggers/backend/internal/database"
	"vibe-moggers/backend/internal/offers"
	"vibe-moggers/backend/internal/server"
	"vibe-moggers/backend/internal/tasks"
	"vibe-moggers/backend/internal/teams"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationCtx, cancelMigrations := context.WithTimeout(context.Background(), 30*time.Second)
	err = database.Migrate(migrationCtx, db)
	cancelMigrations()
	if err != nil {
		logger.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}
	teamHandler := teams.NewHandler(db, logger)
	offerHandler := offers.NewHandler(offers.NewStore(db, tasks.SQLAccess{}), logger)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(db, logger, teamHandler.Register, offerHandler.Register),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API server started", "address", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdownSignal:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("API server stopped")
}
