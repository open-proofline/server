// Package storage manages encrypted blob files on local disk.
//
// It streams uploads to temporary files, hashes bytes while writing, and commits
// verified chunks into immutable final paths without overwriting existing blobs.
package storage
