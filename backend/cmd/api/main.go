package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vibe-moggers/backend/internal/app"
	"vibe-moggers/backend/internal/config"
	"vibe-moggers/backend/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	migrationCtx, cancelMigrations := context.WithTimeout(context.Background(), 30*time.Second)
	err = database.Migrate(migrationCtx, db)
	cancelMigrations()
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              "127.0.0.1:" + cfg.Port,
		Handler:           app.New(db, cfg, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return err
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API server started", "address", httpServer.Addr)
		serverErrors <- httpServer.Serve(listener)
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case sig := <-shutdownSignal:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		httpServer.Close()
		return err
	}

	logger.Info("API server stopped")
	return nil
}
