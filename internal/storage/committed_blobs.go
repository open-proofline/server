package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CommitTemp links a verified temp upload into its final immutable chunk path.
// It never overwrites an existing file. Streamed chunks are stored below a
// stream-specific namespace; legacy unstreamed chunks keep their original path.
func (s *Store) CommitTemp(_ context.Context, upload *TempUpload, incidentID, streamID, mediaType string, chunkIndex int) (string, error) {
	if upload == nil || upload.Path == "" {
		return "", fmt.Errorf("missing temp upload")
	}
	relPath, err := storedBlobPath(incidentID, streamID, mediaType, chunkIndex)
	if err != nil {
		return "", err
	}
	finalPath, err := s.fullPath(relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
		return "", fmt.Errorf("create incident storage directory: %w", err)
	}

	// Hard-linking to a new path gives us atomic no-overwrite behavior on the
	// same filesystem. Existing chunk files are treated as conflicts.
	if err := os.Link(upload.Path, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", ErrAlreadyExists
		}
		return "", fmt.Errorf("commit temp upload: %w", err)
	}
	_ = os.Remove(upload.Path)
	upload.Path = ""

	return relPath, nil
}

// Open opens a previously committed blob by its stored relative path.
func (s *Store) Open(_ context.Context, relPath string) (io.ReadCloser, error) {
	fullPath, err := s.fullPath(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

// Remove deletes a committed blob by its stored relative path.
func (s *Store) Remove(_ context.Context, relPath string) error {
	fullPath, err := s.fullPath(relPath)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

func (s *Store) fullPath(relPath string) (string, error) {
	clean, err := cleanStoredPath(relPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.dataDir, filepath.FromSlash(clean)), nil
}
