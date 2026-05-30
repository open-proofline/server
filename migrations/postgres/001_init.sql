CREATE TABLE IF NOT EXISTS incidents (
  id TEXT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  client_label TEXT,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS media_streams (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
  label TEXT,
  status TEXT NOT NULL CHECK (status IN ('open', 'complete', 'failed')),
  expected_chunk_count INTEGER CHECK (expected_chunk_count IS NULL OR expected_chunk_count > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ,
  failure_reason TEXT,
  UNIQUE (incident_id, id, media_type)
);

CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
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
  stored_path TEXT NOT NULL,
  byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
  sha256_hex TEXT NOT NULL CHECK (sha256_hex ~ '^[0-9a-f]{64}$'),
  created_at TIMESTAMPTZ NOT NULL,
  FOREIGN KEY (incident_id, stream_id, media_type)
    REFERENCES media_streams(incident_id, id, media_type)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS checkins (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL,
  device_battery_percent INTEGER,
  device_network TEXT,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  accuracy_meters DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS incident_tokens (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
  label TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_media_streams_incident_id ON media_streams(incident_id);
CREATE INDEX IF NOT EXISTS idx_media_streams_status ON media_streams(status);
CREATE INDEX IF NOT EXISTS idx_chunks_incident_id ON chunks(incident_id);
CREATE INDEX IF NOT EXISTS idx_chunks_stream_id ON chunks(stream_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_legacy_identity_unique
  ON chunks(incident_id, media_type, chunk_index)
  WHERE stream_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_chunks_stream_identity_unique
  ON chunks(incident_id, stream_id, chunk_index)
  WHERE stream_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_checkins_incident_id ON checkins(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_tokens_incident_id ON incident_tokens(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_tokens_token_hash ON incident_tokens(token_hash);
