package migrations

import "embed"

// FS contains the embedded SQLite schema files.
//
// The schema includes constraints for status, media type, chunk identity,
// byte-size sanity, and SHA-256 shape so SQLite reinforces handler validation.
//
//go:embed *.sql
var FS embed.FS
