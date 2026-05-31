CREATE TABLE IF NOT EXISTS upload_operations (
  id TEXT PRIMARY KEY,
  operation TEXT NOT NULL CHECK (operation IN ('upload_chunk')),
  idempotency_key_hash TEXT NOT NULL CHECK (idempotency_key_hash ~ '^[0-9a-f]{64}$'),
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  stream_id TEXT CHECK (stream_id IS NULL OR length(stream_id) > 0),
  chunk_index INTEGER NOT NULL CHECK (
    chunk_index >= 0
    AND (stream_id IS NULL OR chunk_index > 0)
  ),
  media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
  started_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ NOT NULL,
  original_filename TEXT,
  byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
  sha256_hex TEXT NOT NULL CHECK (sha256_hex ~ '^[0-9a-f]{64}$'),
  fingerprint_hash TEXT NOT NULL CHECK (fingerprint_hash ~ '^[0-9a-f]{64}$'),
  state TEXT NOT NULL CHECK (state IN ('reserved', 'metadata_committed')),
  chunk_id TEXT REFERENCES chunks(id) ON DELETE SET NULL,
  stored_path TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (operation, idempotency_key_hash)
);

CREATE INDEX IF NOT EXISTS idx_upload_operations_incident_id ON upload_operations(incident_id);
CREATE INDEX IF NOT EXISTS idx_upload_operations_chunk_identity ON upload_operations(incident_id, stream_id, media_type, chunk_index);
