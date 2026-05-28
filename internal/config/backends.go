package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	MetadataBackendSQLite   = "sqlite"
	BlobBackendLocal        = "local"
	CoordinationBackendNone = "none"
)

func backendSelectionFromEnv() (BackendSelection, error) {
	metadata, err := backendFromEnv(
		"SAFE_METADATA_BACKEND",
		MetadataBackendSQLite,
		[]string{MetadataBackendSQLite},
	)
	if err != nil {
		return BackendSelection{}, err
	}
	blob, err := backendFromEnv(
		"SAFE_BLOB_BACKEND",
		BlobBackendLocal,
		[]string{BlobBackendLocal},
	)
	if err != nil {
		return BackendSelection{}, err
	}
	coordination, err := backendFromEnv(
		"SAFE_COORDINATION_BACKEND",
		CoordinationBackendNone,
		[]string{CoordinationBackendNone},
	)
	if err != nil {
		return BackendSelection{}, err
	}

	return BackendSelection{
		Metadata:     metadata,
		Blob:         blob,
		Coordination: coordination,
	}, nil
}

func backendFromEnv(name, fallback string, supported []string) (string, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	value := strings.ToLower(strings.TrimSpace(raw))
	for _, candidate := range supported {
		if value == candidate {
			return value, nil
		}
	}
	return "", UnsupportedBackendError{
		EnvName:   name,
		Supported: supported,
	}
}

// UnsupportedBackendError reports a rejected backend selector without exposing
// the rejected value.
type UnsupportedBackendError struct {
	EnvName   string
	Supported []string
}

func (e UnsupportedBackendError) Error() string {
	return fmt.Sprintf("parse %s: unsupported backend; supported values: %s",
		e.EnvName,
		strings.Join(e.Supported, ", "),
	)
}
