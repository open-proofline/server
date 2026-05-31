package migrations

import "embed"

// FS contains the embedded SQLite schema files.
//
// The schema includes constraints for incident and stream status, media type,
// chunk identity, byte-size sanity, SHA-256 shape, and hashed idempotency
// operation state so SQLite reinforces handler validation.
//
//go:embed *.sql
var FS embed.FS
