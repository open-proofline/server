package postgres

import "embed"

// FS contains the embedded PostgreSQL schema files.
//
// PostgreSQL migrations are separate from SQLite migrations because PostgreSQL
// uses backend-specific DDL, constraints, and advisory-lock behavior.
//
//go:embed *.sql
var FS embed.FS
