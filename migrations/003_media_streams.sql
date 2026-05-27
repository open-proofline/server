CREATE TABLE IF NOT EXISTS media_streams (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  media_type TEXT NOT NULL CHECK (media_type IN ('audio', 'video', 'location', 'metadata')),
  label TEXT,
  status TEXT NOT NULL CHECK (status IN ('open', 'complete', 'failed')),
  expected_chunk_count INTEGER CHECK (expected_chunk_count IS NULL OR expected_chunk_count > 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT,
  failed_at TEXT,
  failure_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_media_streams_incident_id ON media_streams(incident_id);
CREATE INDEX IF NOT EXISTS idx_media_streams_status ON media_streams(status);
