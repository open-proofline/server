package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

const createSchemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TEXT NOT NULL
);`

type migrationStep struct {
	id       string
	checksum string
	apply    func(context.Context, *sql.Tx) error
}

type schemaStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Migrate applies every embedded SQL migration in lexical order and records
// completed steps in schema_migrations.
func Migrate(ctx context.Context, conn *sql.DB) error {
	if _, err := conn.ExecContext(ctx, createSchemaMigrationsSQL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	steps, err := migrationSteps()
	if err != nil {
		return err
	}
	for _, step := range steps {
		if err := applyMigrationStep(ctx, conn, step); err != nil {
			return err
		}
	}
	return nil
}

func migrationSteps() ([]migrationStep, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	steps := []migrationStep{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		id := entry.Name()
		migration, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		sqlText := string(migration)
		steps = append(steps, migrationStep{
			id:       id,
			checksum: checksumBytes(migration),
			apply: func(ctx context.Context, tx *sql.Tx) error {
				if _, err := tx.ExecContext(ctx, sqlText); err != nil {
					return fmt.Errorf("apply sql: %w", err)
				}
				return nil
			},
		})
	}

	steps = append(steps,
		compatMigrationStep("004_chunks_stream_id_compat", "add chunks.stream_id column and index when missing", ensureChunkStreamColumn),
		compatMigrationStep("005_drop_emergency_token_last_used_compat", "drop obsolete emergency_tokens.last_used_at column when present", dropEmergencyTokenLastUsedColumn),
		compatMigrationStep("006_chunks_stream_identity_compat", "rebuild chunks uniqueness around stream scoped identity", ensureChunkStreamIdentity),
	)
	return steps, nil
}

func compatMigrationStep(id, description string, apply func(context.Context, schemaStore) error) migrationStep {
	return migrationStep{
		id:       id,
		checksum: checksumString("compat:" + id + ":" + description),
		apply: func(ctx context.Context, tx *sql.Tx) error {
			return apply(ctx, tx)
		},
	}
}

func applyMigrationStep(ctx context.Context, conn *sql.DB, step migrationStep) error {
	recordedChecksum, recorded, err := migrationRecord(ctx, conn, step.id)
	if err != nil {
		return err
	}
	if recorded {
		if recordedChecksum != step.checksum {
			return fmt.Errorf("migration %s checksum mismatch: recorded %s, current %s", step.id, recordedChecksum, step.checksum)
		}
		return nil
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", step.id, err)
	}
	if err := step.apply(ctx, tx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", step.id, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (id, checksum, applied_at)
		VALUES (?, ?, ?)`,
		step.id,
		step.checksum,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", step.id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", step.id, err)
	}
	return nil
}

func migrationRecord(ctx context.Context, conn *sql.DB, id string) (string, bool, error) {
	var checksum string
	err := conn.QueryRowContext(ctx, `
		SELECT checksum
		FROM schema_migrations
		WHERE id = ?`,
		id,
	).Scan(&checksum)
	if err == nil {
		return checksum, true, nil
	}
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return "", false, fmt.Errorf("read schema migration %s: %w", id, err)
}

func checksumBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func checksumString(body string) string {
	return checksumBytes([]byte(body))
}

func ensureChunkStreamColumn(ctx context.Context, store schemaStore) error {
	hasStreamID, err := tableHasColumn(ctx, store, "chunks", "stream_id")
	if err != nil {
		return err
	}
	if !hasStreamID {
		if _, err := store.ExecContext(ctx, "ALTER TABLE chunks ADD COLUMN stream_id TEXT"); err != nil {
			return fmt.Errorf("add chunks.stream_id: %w", err)
		}
	}
	if _, err := store.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_chunks_stream_id ON chunks(stream_id)"); err != nil {
		return fmt.Errorf("create chunks stream index: %w", err)
	}
	return nil
}

func ensureChunkStreamIdentity(ctx context.Context, store schemaStore) error {
	if err := ensureChunkStreamColumn(ctx, store); err != nil {
		return err
	}
	if _, err := store.ExecContext(ctx, "UPDATE chunks SET stream_id = NULL WHERE stream_id = ''"); err != nil {
		return fmt.Errorf("normalize empty chunk stream ids: %w", err)
	}
	if _, err := store.ExecContext(ctx, "DROP TABLE IF EXISTS chunks_stream_identity_migration"); err != nil {
		return fmt.Errorf("drop stale chunks migration table: %w", err)
	}
	if _, err := store.ExecContext(ctx, `
		CREATE TABLE chunks_stream_identity_migration (
			id TEXT PRIMARY KEY,
			incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
			stream_id TEXT CHECK (stream_id IS NULL OR length(stream_id) > 0),
			chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
			media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			original_filename TEXT,
			stored_path TEXT NOT NULL,
			byte_size INTEGER NOT NULL CHECK (byte_size >= 0),
			sha256_hex TEXT NOT NULL CHECK (
				length(sha256_hex) = 64
				AND sha256_hex GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
			),
			created_at TEXT NOT NULL
		);`); err != nil {
		return fmt.Errorf("create stream identity chunks table: %w", err)
	}
	if _, err := store.ExecContext(ctx, `
		INSERT INTO chunks_stream_identity_migration (
			id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		)
		SELECT id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		FROM chunks;`); err != nil {
		return fmt.Errorf("copy chunks into stream identity table: %w", err)
	}
	if _, err := store.ExecContext(ctx, "DROP TABLE chunks"); err != nil {
		return fmt.Errorf("drop legacy chunks table: %w", err)
	}
	if _, err := store.ExecContext(ctx, "ALTER TABLE chunks_stream_identity_migration RENAME TO chunks"); err != nil {
		return fmt.Errorf("rename stream identity chunks table: %w", err)
	}
	if _, err := store.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_chunks_incident_id ON chunks(incident_id)"); err != nil {
		return fmt.Errorf("create chunks incident index: %w", err)
	}
	if _, err := store.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_chunks_stream_id ON chunks(stream_id)"); err != nil {
		return fmt.Errorf("create chunks stream index: %w", err)
	}
	if _, err := store.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_legacy_identity_unique
		ON chunks(incident_id, media_type, chunk_index)
		WHERE stream_id IS NULL`); err != nil {
		return fmt.Errorf("create legacy chunk identity index: %w", err)
	}
	if _, err := store.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_stream_identity_unique
		ON chunks(incident_id, stream_id, chunk_index)
		WHERE stream_id IS NOT NULL`); err != nil {
		return fmt.Errorf("create stream chunk identity index: %w", err)
	}
	return nil
}

func dropEmergencyTokenLastUsedColumn(ctx context.Context, store schemaStore) error {
	hasLastUsedAt, err := tableHasColumn(ctx, store, "emergency_tokens", "last_used_at")
	if err != nil {
		return err
	}
	if !hasLastUsedAt {
		return nil
	}
	if _, err := store.ExecContext(ctx, "ALTER TABLE emergency_tokens DROP COLUMN last_used_at"); err != nil {
		return fmt.Errorf("drop emergency_tokens.last_used_at: %w", err)
	}
	return nil
}

func tableHasColumn(ctx context.Context, store schemaStore, tableName, columnName string) (bool, error) {
	rows, err := store.QueryContext(ctx, "PRAGMA table_info("+tableName+")")
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
