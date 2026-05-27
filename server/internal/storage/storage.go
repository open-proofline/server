package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrTooLarge indicates that an upload exceeded its configured byte limit.
	ErrTooLarge = errors.New("upload too large")
	// ErrAlreadyExists indicates that committing a temp upload would overwrite
	// an existing immutable chunk file.
	ErrAlreadyExists = errors.New("stored chunk already exists")
	// ErrUnsafePath indicates that a requested storage path could escape the
	// configured data directory.
	ErrUnsafePath = errors.New("unsafe storage path")
)

// Store manages temporary and final blob files under one data directory.
type Store struct {
	dataDir string
	tempDir string
}

// New creates the data and temporary directories used for encrypted blob
// storage.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	tempDir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	return &Store{dataDir: dataDir, tempDir: tempDir}, nil
}
