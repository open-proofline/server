package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"safety-recorder/server/internal/config"
	"safety-recorder/server/internal/db"
	"safety-recorder/server/internal/httpapi"
	"safety-recorder/server/internal/incidents"
	"safety-recorder/server/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	blobStore, err := storage.New(cfg.DataDir)
	if err != nil {
		return err
	}

	repo := incidents.NewRepository(conn)
	apiOptions := httpapi.Options{
		MaxUploadBytes:          cfg.MaxUploadBytes,
		DefaultIncidentTokenTTL: &cfg.DefaultIncidentTokenTTL,
		Logger:                  logger,
	}
	privateHandler := httpapi.NewPrivate(repo, blobStore, apiOptions)
	publicHandler := httpapi.NewPublic(repo, blobStore, apiOptions)
	servers := newHTTPServers(cfg, privateHandler, publicHandler)

	errCh := make(chan error, len(servers))
	for _, server := range servers {
		startServer(errCh, logger, server)
	}

	select {
	case <-ctx.Done():
		return shutdownServers(servers)
	case err := <-errCh:
		_ = shutdownServers(servers)
		return err
	}
}
