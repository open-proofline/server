CREATE TABLE IF NOT EXISTS incidents (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  client_label TEXT,
  notes TEXT
);

-- Chunks are immutable once accepted. The unique constraint rejects duplicate
-- uploads for the same incident, media type, and client-supplied chunk index.
CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
  media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL,
  original_filename TEXT,
  stored_path TEXT NOT NULL,
  byte_size INTEGER NOT NULL CHECK (byte_size >= 0),
  -- Store only lowercase SHA-256 hex; the backend verifies the bytes before
  -- metadata is inserted.
  sha256_hex TEXT NOT NULL CHECK (
    length(sha256_hex) = 64
    AND sha256_hex GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
  ),
  created_at TEXT NOT NULL,
  UNIQUE (incident_id, media_type, chunk_index)
);

CREATE TABLE IF NOT EXISTS checkins (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  device_battery_percent INTEGER,
  device_network TEXT,
  latitude REAL,
  longitude REAL,
  accuracy_meters REAL
);

CREATE INDEX IF NOT EXISTS idx_chunks_incident_id ON chunks(incident_id);
CREATE INDEX IF NOT EXISTS idx_checkins_incident_id ON checkins(incident_id);
