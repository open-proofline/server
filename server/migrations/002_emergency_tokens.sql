-- Emergency tokens are read-only bearer capabilities scoped to one incident.
-- The raw token is returned once by the API; SQLite stores only the hash.
CREATE TABLE IF NOT EXISTS emergency_tokens (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  -- Keep token hashes unique and shaped like lowercase SHA-256 hex.
  token_hash TEXT NOT NULL UNIQUE CHECK (
    length(token_hash) = 64
    AND token_hash GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
  ),
  label TEXT,
  created_at TEXT NOT NULL,
  -- expires_at and revoked_at are the validity gates; last_used_at records
  -- successful reads without granting extra access.
  expires_at TEXT,
  revoked_at TEXT,
  last_used_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_emergency_tokens_incident_id ON emergency_tokens(incident_id);
CREATE INDEX IF NOT EXISTS idx_emergency_tokens_token_hash ON emergency_tokens(token_hash);
