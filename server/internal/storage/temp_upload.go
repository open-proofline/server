package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// TempUpload describes a staged upload and the hash computed while it was read.
type TempUpload struct {
	Path      string
	ByteSize  int64
	SHA256Hex string
}

// SaveTemp streams reader into a temporary file, enforcing maxBytes and
// computing SHA-256 without buffering the upload in memory.
func (s *Store) SaveTemp(reader io.Reader, maxBytes int64) (*TempUpload, error) {
	file, err := os.CreateTemp(s.tempDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("create temp upload: %w", err)
	}
	tempPath := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}

	hash := sha256.New()
	// Read one byte past the limit so an oversized upload is detected without
	// accepting a truncated file.
	limited := &io.LimitedReader{R: reader, N: maxBytes + 1}
	byteSize, copyErr := io.Copy(io.MultiWriter(file, hash), limited)
	if copyErr != nil {
		cleanup()
		return nil, fmt.Errorf("write temp upload: %w", copyErr)
	}
	if byteSize > maxBytes {
		cleanup()
		return nil, ErrTooLarge
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return nil, fmt.Errorf("sync temp upload: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close temp upload: %w", err)
	}

	return &TempUpload{
		Path:      tempPath,
		ByteSize:  byteSize,
		SHA256Hex: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

// Cleanup removes the staged file if it has not already been committed.
func (u *TempUpload) Cleanup() {
	if u == nil || u.Path == "" {
		return
	}
	_ = os.Remove(u.Path)
	u.Path = ""
}
