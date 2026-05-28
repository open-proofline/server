package storage

import "io"

// BlobStore is the storage boundary used by the HTTP API for encrypted chunk
// bytes.
//
// Implementations must preserve these semantics:
//   - SaveTemp stages opaque encrypted bytes, enforces maxBytes, and returns a
//     SHA-256 hash over exactly the accepted bytes.
//   - CommitTemp moves a verified staged upload to a server-controlled stored
//     path and must never overwrite an existing committed blob.
//   - Open accepts only previously stored server-controlled paths from metadata.
//   - Remove is for rollback of a just-committed blob when metadata insertion
//     fails; it must not accept client-provided paths.
type BlobStore interface {
	SaveTemp(reader io.Reader, maxBytes int64) (*TempUpload, error)
	CommitTemp(upload *TempUpload, incidentID, streamID, mediaType string, chunkIndex int) (string, error)
	Open(storedPath string) (io.ReadCloser, error)
	Remove(storedPath string) error
}

var _ BlobStore = (*Store)(nil)
