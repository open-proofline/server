package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/open-proofline/server/internal/config"
	"github.com/open-proofline/server/internal/db"
	"github.com/open-proofline/server/internal/httpapi"
	"github.com/open-proofline/server/internal/incidents"
	"github.com/open-proofline/server/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logStartupError(logger, err)
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

	blobStore, err := newBlobStore(cfg)
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

func newBlobStore(cfg config.Config) (storage.BlobStore, error) {
	switch cfg.Backends.Blob {
	case config.BlobBackendLocal:
		return storage.New(cfg.DataDir)
	case config.BlobBackendS3:
		return storage.NewS3(storage.S3Options{
			Endpoint:        cfg.S3Blob.Endpoint,
			Region:          cfg.S3Blob.Region,
			Bucket:          cfg.S3Blob.Bucket,
			Prefix:          cfg.S3Blob.Prefix,
			AccessKeyID:     cfg.S3Blob.AccessKeyID,
			SecretAccessKey: cfg.S3Blob.SecretAccessKey,
			SessionToken:    cfg.S3Blob.SessionToken,
			ForcePathStyle:  cfg.S3Blob.ForcePathStyle,
			TempDir:         filepath.Join(cfg.DataDir, "tmp"),
		})
	default:
		return nil, fmt.Errorf("unsupported blob backend %q", cfg.Backends.Blob)
	}
}
