package postgresdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/open-proofline/server/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open creates a PostgreSQL connection and applies PostgreSQL metadata
// migrations before returning the database handle.
func Open(ctx context.Context, cfg config.PostgresConfig) (*sql.DB, error) {
	conn, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres metadata database: %w", err)
	}
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect postgres metadata database: %w", err)
	}
	if err := Migrate(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}
