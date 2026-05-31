package storage

import (
	"context"
	"io"
)

// BlobStore is the storage boundary used by the HTTP API for encrypted chunk
// bytes.
//
// Implementations must preserve these semantics:
//   - SaveTemp stages opaque encrypted bytes, enforces maxBytes, and returns a
//     SHA-256 hash over exactly the accepted bytes.
//   - CommitTemp moves a verified staged upload to a server-controlled stored
//     path and must never overwrite an existing committed blob.
//   - Open accepts only previously stored server-controlled paths from metadata.
//   - Remove deletes only server-controlled stored paths from metadata or
//     rollback state; it must not accept client-provided paths.
type BlobStore interface {
	Check(ctx context.Context) error
	SaveTemp(ctx context.Context, reader io.Reader, maxBytes int64) (*TempUpload, error)
	CommitTemp(ctx context.Context, upload *TempUpload, incidentID, streamID, mediaType string, chunkIndex int) (string, error)
	Open(ctx context.Context, storedPath string) (io.ReadCloser, error)
	Remove(ctx context.Context, storedPath string) error
}

var _ BlobStore = (*Store)(nil)
