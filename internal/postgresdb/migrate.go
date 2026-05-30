package postgresdb

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

	postgresmigrations "github.com/open-proofline/server/migrations/postgres"
)

const createSchemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  id TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL
);`

const (
	postgresMigrationAdvisoryLockNamespace int32 = 0x70726f6f // "proo"
	postgresMigrationAdvisoryLockID        int32 = 0x6c696e65 // "line"
	postgresMigrationUnlockTimeout               = 5 * time.Second
)

type migrationStep struct {
	id       string
	checksum string
	sqlText  string
}

type migrationStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// Migrate applies every embedded PostgreSQL migration in lexical order and
// records completed steps in schema_migrations.
func Migrate(ctx context.Context, db *sql.DB) (err error) {
	steps, err := migrationSteps()
	if err != nil {
		return err
	}

	migrationConn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve postgres migration connection: %w", err)
	}
	defer func() {
		if closeErr := migrationConn.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close postgres migration connection: %w", closeErr)
		}
	}()

	if err := lockPostgresMigrations(ctx, migrationConn); err != nil {
		return err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), postgresMigrationUnlockTimeout)
		defer cancel()
		if unlockErr := unlockPostgresMigrations(unlockCtx, migrationConn); err == nil && unlockErr != nil {
			err = unlockErr
		}
	}()

	return migrateLocked(ctx, migrationConn, steps)
}

func migrateLocked(ctx context.Context, store migrationStore, steps []migrationStep) error {
	if _, err := store.ExecContext(ctx, createSchemaMigrationsSQL); err != nil {
		return fmt.Errorf("create postgres schema_migrations: %w", err)
	}

	for _, step := range steps {
		if err := applyMigrationStep(ctx, store, step); err != nil {
			return err
		}
	}
	return nil
}

func migrationSteps() ([]migrationStep, error) {
	entries, err := fs.ReadDir(postgresmigrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read postgres migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	steps := []migrationStep{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := postgresmigrations.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read postgres migration %s: %w", entry.Name(), err)
		}
		steps = append(steps, migrationStep{
			id:       entry.Name(),
			checksum: checksumBytes(body),
			sqlText:  string(body),
		})
	}
	return steps, nil
}

func applyMigrationStep(ctx context.Context, store migrationStore, step migrationStep) error {
	recordedChecksum, recorded, err := migrationRecord(ctx, store, step.id)
	if err != nil {
		return err
	}
	if recorded {
		if recordedChecksum != step.checksum {
			return fmt.Errorf("postgres migration %s checksum mismatch: recorded %s, current %s", step.id, recordedChecksum, step.checksum)
		}
		return nil
	}

	tx, err := store.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin postgres migration %s: %w", step.id, err)
	}
	if _, err := tx.ExecContext(ctx, step.sqlText); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply postgres migration %s: %w", step.id, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (id, checksum, applied_at)
		VALUES ($1, $2, $3)`,
		step.id,
		step.checksum,
		time.Now().UTC(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record postgres migration %s: %w", step.id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit postgres migration %s: %w", step.id, err)
	}
	return nil
}

func lockPostgresMigrations(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, `
		SELECT pg_advisory_lock($1, $2)`,
		postgresMigrationAdvisoryLockNamespace,
		postgresMigrationAdvisoryLockID,
	); err != nil {
		return fmt.Errorf("lock postgres migrations: %w", err)
	}
	return nil
}

func unlockPostgresMigrations(ctx context.Context, conn *sql.Conn) error {
	var unlocked bool
	err := conn.QueryRowContext(ctx, `
		SELECT pg_advisory_unlock($1, $2)`,
		postgresMigrationAdvisoryLockNamespace,
		postgresMigrationAdvisoryLockID,
	).Scan(&unlocked)
	if err != nil {
		return fmt.Errorf("unlock postgres migrations: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("unlock postgres migrations: lock was not held")
	}
	return nil
}

func migrationRecord(ctx context.Context, store migrationStore, id string) (string, bool, error) {
	var checksum string
	err := store.QueryRowContext(ctx, `
		SELECT checksum
		FROM schema_migrations
		WHERE id = $1`,
		id,
	).Scan(&checksum)
	if err == nil {
		return checksum, true, nil
	}
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return "", false, fmt.Errorf("read postgres schema migration %s: %w", id, err)
}

func checksumBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
