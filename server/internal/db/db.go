package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"safety-recorder/server/migrations"
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
	// WAL keeps the single local database responsive for concurrent reads while
	// uploads are inserting metadata.
	if _, err := conn.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable wal mode: %w", err)
	}
	if err := Migrate(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

// Migrate applies every embedded SQL migration in lexical order.
func Migrate(ctx context.Context, conn *sql.DB) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		migration, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := conn.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	if err := ensureChunkStreamColumn(ctx, conn); err != nil {
		return err
	}

	return nil
}

func ensureChunkStreamColumn(ctx context.Context, conn *sql.DB) error {
	hasStreamID, err := tableHasColumn(ctx, conn, "chunks", "stream_id")
	if err != nil {
		return err
	}
	if !hasStreamID {
		if _, err := conn.ExecContext(ctx, "ALTER TABLE chunks ADD COLUMN stream_id TEXT"); err != nil {
			return fmt.Errorf("add chunks.stream_id: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_chunks_stream_id ON chunks(stream_id)"); err != nil {
		return fmt.Errorf("create chunks stream index: %w", err)
	}
	return nil
}

func tableHasColumn(ctx context.Context, conn *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := conn.QueryContext(ctx, "PRAGMA table_info("+tableName+")")
	if err != nil {
		return false, fmt.Errorf("inspect %s columns: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("scan %s column: %w", tableName, err)
		}
		if name == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate %s columns: %w", tableName, err)
	}
	return false, nil
}
