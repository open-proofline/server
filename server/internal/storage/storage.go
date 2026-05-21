package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrTooLarge      = errors.New("upload too large")
	ErrAlreadyExists = errors.New("stored chunk already exists")
	ErrUnsafePath    = errors.New("unsafe storage path")
)

type Store struct {
	dataDir string
	tempDir string
}

type TempUpload struct {
	Path      string
	ByteSize  int64
	SHA256Hex string
}

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

func (s *Store) CommitTemp(upload *TempUpload, incidentID, mediaType string, chunkIndex int) (string, error) {
	if upload == nil || upload.Path == "" {
		return "", fmt.Errorf("missing temp upload")
	}
	if chunkIndex < 0 || !safePathSegment(incidentID) || !safePathSegment(mediaType) {
		return "", ErrUnsafePath
	}

	relPath := path.Join("incidents", incidentID, fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex))
	finalPath, err := s.fullPath(relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
		return "", fmt.Errorf("create incident storage directory: %w", err)
	}

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

func (s *Store) Open(relPath string) (*os.File, error) {
	fullPath, err := s.fullPath(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

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

func (u *TempUpload) Cleanup() {
	if u == nil || u.Path == "" {
		return
	}
	_ = os.Remove(u.Path)
	u.Path = ""
}
