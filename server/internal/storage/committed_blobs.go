package storage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CommitTemp links a verified temp upload into its final immutable chunk path.
// It never overwrites an existing file. Streamed chunks are stored below a
// stream-specific namespace; legacy unstreamed chunks keep their original path.
func (s *Store) CommitTemp(upload *TempUpload, incidentID, streamID, mediaType string, chunkIndex int) (string, error) {
	if upload == nil || upload.Path == "" {
		return "", fmt.Errorf("missing temp upload")
	}
	if chunkIndex < 0 || !safePathSegment(incidentID) || !safePathSegment(mediaType) {
		return "", ErrUnsafePath
	}
	if streamID != "" && !safePathSegment(streamID) {
		return "", ErrUnsafePath
	}

	relPath := path.Join("incidents", incidentID, fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex))
	if streamID != "" {
		relPath = path.Join("incidents", incidentID, "streams", streamID, fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex))
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
func (s *Store) Open(relPath string) (*os.File, error) {
	fullPath, err := s.fullPath(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

// Remove deletes a committed blob by its stored relative path.
func (s *Store) Remove(relPath string) error {
	fullPath, err := s.fullPath(relPath)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

func (s *Store) fullPath(relPath string) (string, error) {
	if relPath == "" || path.IsAbs(relPath) || strings.Contains(relPath, "\\") {
		return "", ErrUnsafePath
	}
	clean := path.Clean(relPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", ErrUnsafePath
	}
	return filepath.Join(s.dataDir, filepath.FromSlash(clean)), nil
}

func safePathSegment(value string) bool {
	return value != "" &&
		value != "." &&
		value != ".." &&
		!strings.Contains(value, "/") &&
		!strings.Contains(value, "\\")
}
