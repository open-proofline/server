package db

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
)

func TestMigrateCreatesSchemaMigrationsAndRecordsSteps(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !hasTable(t, ctx, conn, "schema_migrations") {
		t.Fatal("expected schema_migrations table")
	}

	got := appliedMigrationIDs(t, ctx, conn)
	want := expectedMigrationIDs(t)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("applied migrations = %v, want %v", got, want)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	first := appliedMigrationIDs(t, ctx, conn)

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	second := appliedMigrationIDs(t, ctx, conn)
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("second migration IDs = %v, want %v", second, first)
	}
}

func TestMigrateRejectsRecordedChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `
		UPDATE schema_migrations
		SET checksum = 'bad'
		WHERE id = '001_init.sql'`); err != nil {
		t.Fatalf("update checksum: %v", err)
	}

	err := Migrate(ctx, conn)
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if !strings.Contains(err.Error(), "001_init.sql checksum mismatch") {
		t.Fatalf("expected checksum mismatch for 001_init.sql, got %v", err)
	}
}

func TestOpenMemoryCreatesMigrationTable(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open :memory:: %v", err)
	}
	defer conn.Close()

	if !hasTable(t, ctx, conn, "schema_migrations") {
		t.Fatal("expected schema_migrations table")
	}
}

func TestMigrateDropsEmergencyTokenLastUsedAt(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	_, err := conn.ExecContext(ctx, `
		CREATE TABLE incidents (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			status TEXT NOT NULL,
			client_label TEXT,
			notes TEXT
		);
		CREATE TABLE emergency_tokens (
			id TEXT PRIMARY KEY,
			incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			label TEXT,
			created_at TEXT NOT NULL,
			expires_at TEXT,
			revoked_at TEXT,
			last_used_at TEXT
		);
		INSERT INTO incidents (id, created_at, updated_at, status)
		VALUES ('inc_existing', '2026-05-21T10:00:00Z', '2026-05-21T10:00:00Z', 'open');
		INSERT INTO emergency_tokens (
			id, incident_id, token_hash, label, created_at, expires_at, revoked_at, last_used_at
		)
		VALUES (
			'etk_existing',
			'inc_existing',
			'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
			'trusted contact',
			'2026-05-21T10:00:00Z',
			NULL,
			NULL,
			'2026-05-21T10:05:00Z'
		);`)
	if err != nil {
		t.Fatalf("create old schema: %v", err)
	}

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if hasColumn(t, ctx, conn, "emergency_tokens", "last_used_at") {
		t.Fatal("expected emergency_tokens.last_used_at to be dropped")
	}

	var count int
	if err := conn.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM emergency_tokens
		WHERE id = 'etk_existing'
			AND incident_id = 'inc_existing'
			AND token_hash = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
			AND label = 'trusted contact'
			AND created_at = '2026-05-21T10:00:00Z'
			AND expires_at IS NULL
			AND revoked_at IS NULL`).Scan(&count); err != nil {
		t.Fatalf("count preserved token: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected existing token row to be preserved, got count %d", count)
	}
	if !hasTable(t, ctx, conn, "schema_migrations") {
		t.Fatal("expected existing database migration to create schema_migrations")
	}
}

func TestMigrateAddsMissingChunkStreamIDColumn(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	createLegacyChunksSchema(t, ctx, conn, false)

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !hasColumn(t, ctx, conn, "chunks", "stream_id") {
		t.Fatal("expected chunks.stream_id column")
	}
}

func TestMigrateAllowsExistingChunkStreamIDColumn(t *testing.T) {
	ctx := context.Background()
	conn := openMemoryDB(t)
	defer conn.Close()

	createLegacyChunksSchema(t, ctx, conn, true)

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !hasColumn(t, ctx, conn, "chunks", "stream_id") {
		t.Fatal("expected chunks.stream_id column")
	}
}

func openMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return conn
}

func createLegacyChunksSchema(t *testing.T, ctx context.Context, conn *sql.DB, includeStreamID bool) {
	t.Helper()

	streamIDColumn := ""
	if includeStreamID {
		streamIDColumn = "stream_id TEXT,"
	}
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE incidents (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			status TEXT NOT NULL,
			client_label TEXT,
			notes TEXT
		);
		CREATE TABLE chunks (
			id TEXT PRIMARY KEY,
			incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
			`+streamIDColumn+`
			chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
			media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			original_filename TEXT,
			stored_path TEXT NOT NULL,
			byte_size INTEGER NOT NULL CHECK (byte_size >= 0),
			sha256_hex TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE (incident_id, media_type, chunk_index)
		);`)
	if err != nil {
		t.Fatalf("create legacy chunks schema: %v", err)
	}
}

func hasTable(t *testing.T, ctx context.Context, conn *sql.DB, tableName string) bool {
	t.Helper()
	var name string
	err := conn.QueryRowContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = ?`,
		tableName,
	).Scan(&name)
	if err == nil {
		return true
	}
	if err == sql.ErrNoRows {
		return false
	}
	t.Fatalf("query table %s: %v", tableName, err)
	return false
}

func appliedMigrationIDs(t *testing.T, ctx context.Context, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.QueryContext(ctx, `
		SELECT id
		FROM schema_migrations
		ORDER BY id`)
	if err != nil {
		t.Fatalf("query schema migrations: %v", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan migration id: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migration ids: %v", err)
	}
	return ids
}

func expectedMigrationIDs(t *testing.T) []string {
	t.Helper()
	steps, err := migrationSteps()
	if err != nil {
		t.Fatalf("migrationSteps: %v", err)
	}
	ids := make([]string, 0, len(steps))
	for _, step := range steps {
		ids = append(ids, step.id)
	}
	return ids
}

func hasColumn(t *testing.T, ctx context.Context, conn *sql.DB, tableName, columnName string) bool {
	t.Helper()
	hasColumn, err := tableHasColumn(ctx, conn, tableName, columnName)
	if err != nil {
		t.Fatalf("inspect %s.%s: %v", tableName, columnName, err)
	}
	return hasColumn
}
