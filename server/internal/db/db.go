package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Open creates a SQLite connection, enables required pragmas, and applies the
// embedded schema before returning the database handle.
func Open(ctx context.Context, dbPath string) (*sql.DB, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	conn.SetMaxOpenConns(1)

	// Foreign keys protect chunk/checkin rows from pointing at missing
	// incidents even if a future caller bypasses HTTP validation.
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := enableWALMode(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := Migrate(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

// WAL keeps the single local database responsive for concurrent reads while
// uploads are inserting metadata.
func enableWALMode(ctx context.Context, conn *sql.DB) error {
	var journalMode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		return fmt.Errorf("enable wal mode: %w", err)
	}
	if !isWALJournalMode(journalMode) {
		return fmt.Errorf("enable wal mode: sqlite returned journal mode %q", journalMode)
	}
	return nil
}

func isWALJournalMode(journalMode string) bool {
	return strings.EqualFold(strings.TrimSpace(journalMode), "wal")
}
