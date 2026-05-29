package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPostgresMaxOpenConns    = 10
	defaultPostgresMaxIdleConns    = 5
	defaultPostgresConnMaxLifetime = 30 * time.Minute
)

func postgresConfigFromEnv(metadataBackend string) (PostgresConfig, error) {
	cfg := PostgresConfig{
		DSN:             strings.TrimSpace(os.Getenv("SAFE_POSTGRES_DSN")),
		MaxOpenConns:    defaultPostgresMaxOpenConns,
		MaxIdleConns:    defaultPostgresMaxIdleConns,
		ConnMaxLifetime: defaultPostgresConnMaxLifetime,
	}
	if metadataBackend != MetadataBackendPostgres {
		return cfg, nil
	}

	if cfg.DSN == "" {
		return PostgresConfig{}, fmt.Errorf("parse SAFE_POSTGRES_DSN: required when SAFE_METADATA_BACKEND=postgresql")
	}

	maxOpenConns, err := nonNegativeIntFromEnv("SAFE_POSTGRES_MAX_OPEN_CONNS", defaultPostgresMaxOpenConns)
	if err != nil {
		return PostgresConfig{}, err
	}
	maxIdleConns, err := nonNegativeIntFromEnv("SAFE_POSTGRES_MAX_IDLE_CONNS", defaultPostgresMaxIdleConns)
	if err != nil {
		return PostgresConfig{}, err
	}
	connMaxLifetime, err := durationFromEnv("SAFE_POSTGRES_CONN_MAX_LIFETIME", defaultPostgresConnMaxLifetime)
	if err != nil {
		return PostgresConfig{}, err
	}

	cfg.MaxOpenConns = maxOpenConns
	cfg.MaxIdleConns = maxIdleConns
	cfg.ConnMaxLifetime = connMaxLifetime
	return cfg, nil
}

func nonNegativeIntFromEnv(name string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("parse %s: empty integer", name)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: invalid integer", name)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("parse %s: integer must be non-negative", name)
	}
	return parsed, nil
}
