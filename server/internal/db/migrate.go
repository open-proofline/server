package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"safety-recorder/server/migrations"
)

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
		migration, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		steps = append(steps, embeddedSQLMigrationStep(entry.Name(), migration))
	}

	steps = append(steps,
		compatMigrationStep("004_chunks_stream_id_compat", "add chunks.stream_id column and index when missing", ensureChunkStreamColumn),
		compatMigrationStep("005_drop_emergency_token_last_used_compat", "drop obsolete emergency_tokens.last_used_at column when present", dropEmergencyTokenLastUsedColumn),
		compatMigrationStep("006_chunks_stream_identity_compat", "rebuild chunks uniqueness around stream scoped identity", ensureChunkStreamIdentity),
	)
	return steps, nil
}

func embeddedSQLMigrationStep(id string, migration []byte) migrationStep {
	sqlText := string(migration)
	return migrationStep{
		id:       id,
		checksum: checksumBytes(migration),
		apply: func(ctx context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, sqlText); err != nil {
				return fmt.Errorf("apply sql: %w", err)
			}
			return nil
		},
	}
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
