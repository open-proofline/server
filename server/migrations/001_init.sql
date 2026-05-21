CREATE TABLE IF NOT EXISTS incidents (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  client_label TEXT,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL,
  media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL,
  original_filename TEXT,
  stored_path TEXT NOT NULL,
  byte_size INTEGER NOT NULL,
  sha256_hex TEXT NOT NULL,
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
