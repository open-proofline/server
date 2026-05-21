package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	apiOptions := httpapi.Options{MaxUploadBytes: cfg.MaxUploadBytes, Logger: logger}
	privateServer := &http.Server{
		Addr:              cfg.PrivateBindAddr,
		Handler:           httpapi.NewPrivate(repo, blobStore, apiOptions),
		ReadHeaderTimeout: 10 * time.Second,
	}
	publicServer := &http.Server{
		Addr:              cfg.PublicBindAddr,
		Handler:           httpapi.NewPublic(repo, blobStore, apiOptions),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 2)
	startServer(errCh, logger, "private api", privateServer)
	startServer(errCh, logger, "public emergency viewer", publicServer)

	select {
	case <-ctx.Done():
		return shutdownServers([]*http.Server{privateServer, publicServer})
	case err := <-errCh:
		_ = shutdownServers([]*http.Server{privateServer, publicServer})
		return err
	}
}

func startServer(errCh chan<- error, logger *slog.Logger, name string, server *http.Server) {
	go func() {
		logger.Info("starting "+name+" server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("%s server: %w", name, err)
		}
	}()
}

func shutdownServers(servers []*http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var shutdownErr error
	for _, server := range servers {
		if err := server.Shutdown(shutdownCtx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}
