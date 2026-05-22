package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestMigrateDropsEmergencyTokenLastUsedAt(t *testing.T) {
	ctx := context.Background()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `
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
}

func hasColumn(t *testing.T, ctx context.Context, conn *sql.DB, tableName, columnName string) bool {
	t.Helper()
	hasColumn, err := tableHasColumn(ctx, conn, tableName, columnName)
	if err != nil {
		t.Fatalf("inspect %s.%s: %v", tableName, columnName, err)
	}
	return hasColumn
}
