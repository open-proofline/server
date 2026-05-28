package storage

import (
	"fmt"
	"path"
	"strings"
)

func storedBlobPath(incidentID, streamID, mediaType string, chunkIndex int) (string, error) {
	if chunkIndex < 0 || !safePathSegment(incidentID) || !safePathSegment(mediaType) {
		return "", ErrUnsafePath
	}
	if streamID != "" && !safePathSegment(streamID) {
		return "", ErrUnsafePath
	}

	filename := fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex)
	if streamID != "" {
		return path.Join("incidents", incidentID, "streams", streamID, filename), nil
	}
	return path.Join("incidents", incidentID, filename), nil
}

func cleanStoredPath(storedPath string) (string, error) {
	if storedPath == "" || path.IsAbs(storedPath) || strings.Contains(storedPath, "\\") {
		return "", ErrUnsafePath
	}
	for _, segment := range strings.Split(storedPath, "/") {
		if !safePathSegment(segment) {
			return "", ErrUnsafePath
		}
	}
	return path.Clean(storedPath), nil
}

func safePathSegment(value string) bool {
	return value != "" &&
		value != "." &&
		value != ".." &&
		!strings.Contains(value, "/") &&
		!strings.Contains(value, "\\")
}
