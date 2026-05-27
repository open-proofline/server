CREATE TABLE IF NOT EXISTS incident_tokens (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  -- Keep token hashes unique and shaped like lowercase SHA-256 hex.
  token_hash TEXT NOT NULL UNIQUE CHECK (
    length(token_hash) = 64
    AND token_hash GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
  ),
  label TEXT,
  created_at TEXT NOT NULL,
  expires_at TEXT,
  revoked_at TEXT
);

INSERT OR IGNORE INTO incident_tokens (
  id, incident_id, token_hash, label, created_at, expires_at, revoked_at
)
SELECT id, incident_id, token_hash, label, created_at, expires_at, revoked_at
FROM emergency_tokens;

DROP TABLE IF EXISTS emergency_tokens;

CREATE INDEX IF NOT EXISTS idx_incident_tokens_incident_id ON incident_tokens(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_tokens_token_hash ON incident_tokens(token_hash);
