package postgres

import "embed"

// FS contains the embedded PostgreSQL schema files.
//
// PostgreSQL migrations are separate from SQLite migrations because the first
// PostgreSQL schema models the current canonical end state directly.
//
//go:embed *.sql
var FS embed.FS
