// Package db opens SQLite connections and applies the embedded schema.
//
// It centralizes SQLite safety settings such as foreign keys and WAL mode so
// the HTTP layer can rely on database constraints as a second line of defense.
package db
