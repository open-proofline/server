package main

import (
	"context"
	"errors"

	"github.com/open-proofline/server/internal/config"
	"github.com/open-proofline/server/internal/httpapi"
)

var errAuthBootstrapRequired = errors.New("auth bootstrap required")

func checkAuthBootstrap(ctx context.Context, repo httpapi.MetadataRepository, cfg config.Config) error {
	hasAdmin, err := repo.HasAdminAccount(ctx)
	if err != nil {
		return err
	}
	if hasAdmin || cfg.AuthBootstrapSecret != "" {
		return nil
	}
	return errAuthBootstrapRequired
}
