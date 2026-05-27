package db

import (
	"context"
	"fmt"
)

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
