ALTER TABLE incidents ADD COLUMN deletion_state TEXT NOT NULL DEFAULT 'active' CHECK (
  deletion_state IN ('active', 'deletion_pending', 'deleting', 'deletion_failed', 'deleted')
);

CREATE TABLE IF NOT EXISTS incident_deletion_decisions (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('account_request', 'admin_request', 'retention_policy')),
  reason_code TEXT,
  actor_account_id TEXT,
  allow_open INTEGER NOT NULL CHECK (allow_open IN (0, 1)),
  state TEXT NOT NULL CHECK (
    state IN ('deletion_pending', 'deleting', 'deletion_failed', 'deleted')
  ),
  item_count INTEGER NOT NULL DEFAULT 0 CHECK (item_count >= 0),
  error_code TEXT,
  requested_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  started_at TEXT,
  completed_at TEXT,
  UNIQUE (incident_id)
);

CREATE TABLE IF NOT EXISTS incident_deletion_items (
  id TEXT PRIMARY KEY,
  decision_id TEXT NOT NULL REFERENCES incident_deletion_decisions(id) ON DELETE CASCADE,
  incident_id TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  stored_path TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('pending', 'deleted', 'failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  error_code TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_attempt_at TEXT,
  completed_at TEXT,
  UNIQUE (decision_id, stored_path)
);

CREATE INDEX IF NOT EXISTS idx_incidents_deletion_state ON incidents(deletion_state);
CREATE INDEX IF NOT EXISTS idx_incident_deletion_decisions_state ON incident_deletion_decisions(state);
CREATE INDEX IF NOT EXISTS idx_incident_deletion_items_decision_id ON incident_deletion_items(decision_id);
CREATE INDEX IF NOT EXISTS idx_incident_deletion_items_state ON incident_deletion_items(state);
