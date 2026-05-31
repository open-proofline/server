package httpapi_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-proofline/server/internal/incidents"
)

func TestUploadValidChunk(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	payload := []byte("encrypted audio data")
	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.StatusCode, body)
	}
	var chunk incidents.Chunk
	if err := json.Unmarshal(body, &chunk); err != nil {
		t.Fatalf("decode chunk: %v", err)
	}
	if chunk.MediaType != "audio" || chunk.ChunkIndex != 1 || chunk.ByteSize != int64(len(payload)) {
		t.Fatalf("unexpected chunk response: %+v", chunk)
	}

	storedPath := filepath.Join(app.dataDir, "incidents", incidentID, "audio_000001.enc")
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored chunk: %v", err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("stored payload mismatch")
	}

	response, body = get(t, app, "/v1/incidents/"+incidentID+"/chunks/audio/1")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected chunk bytes status 200, got %d: %s", response.StatusCode, body)
	}
	assertNoSniff(t, response)
	assertNoStore(t, response)
}

func TestLegacyUnstreamedChunkIndexZeroIsAccepted(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("legacy encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 0, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected legacy zero-index upload status 201, got %d: %s", response.StatusCode, body)
	}
	var chunk incidents.Chunk
	if err := json.Unmarshal(body, &chunk); err != nil {
		t.Fatalf("decode chunk: %v", err)
	}
	if chunk.StreamID != "" || chunk.ChunkIndex != 0 {
		t.Fatalf("unexpected legacy chunk response: %+v", chunk)
	}
}

func TestRejectDuplicateChunkIndex(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")
	duplicatePayload := []byte("different encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first upload status 201, got %d: %s", response.StatusCode, body)
	}

	response, body = uploadChunk(t, app, incidentID, 1, "audio", duplicatePayload, sha256Hex(duplicatePayload))
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "duplicate_chunk")

	storedPath := filepath.Join(app.dataDir, "incidents", incidentID, "audio_000001.enc")
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored chunk: %v", err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("duplicate upload overwrote stored payload")
	}
	assertTempDirEmpty(t, app)
}

func TestUploadIdempotencyKeyEquivalentRetryReturnsSuccess(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio recording")
	payload := []byte("encrypted stream audio data")
	key := "chunk-upload-key-1"

	response, body := uploadChunkWithIdempotencyKey(t, app, incidentID, stream.ID, 1, incidents.MediaTypeAudio, payload, sha256Hex(payload), "chunk.enc", key)
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first upload status 201, got %d: %s", response.StatusCode, body)
	}
	var first incidents.Chunk
	if err := json.Unmarshal(body, &first); err != nil {
		t.Fatalf("decode first chunk: %v", err)
	}

	response, body = uploadChunkWithIdempotencyKey(t, app, incidentID, stream.ID, 1, incidents.MediaTypeAudio, payload, sha256Hex(payload), "chunk.enc", key)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected idempotent retry status 200, got %d: %s", response.StatusCode, body)
	}
	if response.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected Idempotency-Replayed header, got %q", response.Header.Get("Idempotency-Replayed"))
	}
	var replayed incidents.Chunk
	if err := json.Unmarshal(body, &replayed); err != nil {
		t.Fatalf("decode replayed chunk: %v", err)
	}
	if replayed.ID != first.ID || replayed.StoredPath != first.StoredPath {
		t.Fatalf("expected replayed chunk to match first response: first=%+v replayed=%+v", first, replayed)
	}

	storedPath := filepath.Join(app.dataDir, "incidents", incidentID, "streams", stream.ID, "audio_000001.enc")
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored chunk: %v", err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("idempotent retry changed stored payload")
	}

	var storedHash string
	var operationState string
	if err := app.db.QueryRowContext(t.Context(), `
		SELECT idempotency_key_hash, state
		FROM upload_operations
		WHERE chunk_id = ?`,
		first.ID,
	).Scan(&storedHash, &operationState); err != nil {
		t.Fatalf("read upload operation: %v", err)
	}
	if storedHash == key || len(storedHash) != 64 {
		t.Fatalf("idempotency key was not stored as a 64-character hash")
	}
	if operationState != incidents.UploadOperationStateMetadataCommitted {
		t.Fatalf("operation state = %q, want %q", operationState, incidents.UploadOperationStateMetadataCommitted)
	}
	assertTempDirEmpty(t, app)
}

func TestUploadIdempotencyKeyReuseWithDifferentInputsConflicts(t *testing.T) {
	var logs bytes.Buffer
	app := newTestAppWithMaxUploadBytesAndLogger(t, 1024*1024, slog.New(slog.NewTextHandler(&logs, nil)))
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")
	key := "raw-idempotency-key-secret"

	response, body := uploadChunkWithIdempotencyKey(t, app, incidentID, "", 1, incidents.MediaTypeAudio, payload, sha256Hex(payload), "chunk.enc", key)
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first upload status 201, got %d: %s", response.StatusCode, body)
	}

	response, body = uploadChunkWithIdempotencyKey(t, app, incidentID, "", 1, incidents.MediaTypeAudio, payload, sha256Hex(payload), "other-name.enc", key)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected idempotency conflict status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "idempotency_conflict")
	for _, disallowed := range []string{key, "other-name.enc", string(payload), app.dataDir} {
		if bytes.Contains(body, []byte(disallowed)) {
			t.Fatalf("idempotency conflict response exposed %q: %s", disallowed, body)
		}
		if strings.TrimSpace(disallowed) != "" && bytes.Contains(logs.Bytes(), []byte(disallowed)) {
			t.Fatalf("idempotency conflict logs exposed %q: %s", disallowed, logs.String())
		}
	}
}

func TestDuplicateChunkWithoutIdempotencyKeyKeepsExistingBehavior(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 1, incidents.MediaTypeAudio, payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first upload status 201, got %d: %s", response.StatusCode, body)
	}
	response, body = uploadChunk(t, app, incidentID, 1, incidents.MediaTypeAudio, payload, sha256Hex(payload))
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "duplicate_chunk")

	var operations int
	if err := app.db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM upload_operations`,
	).Scan(&operations); err != nil {
		t.Fatalf("count upload operations: %v", err)
	}
	if operations != 0 {
		t.Fatalf("unexpected upload operation rows for no-key upload: %d", operations)
	}
}

func TestRejectHashMismatchRemovesTempFile(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, stringsOf("0", 64))
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected hash mismatch status 400, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "hash_mismatch")
	assertNoStoredFile(t, app, incidentID, "audio_000001.enc")
	assertTempDirEmpty(t, app)
}

func TestRejectUploadTooLargeRemovesTempFile(t *testing.T) {
	app := newTestAppWithMaxUploadBytes(t, 8)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("this encrypted payload is too large")

	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected upload too large status 413, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "upload_too_large")
	assertNoStoredFile(t, app, incidentID, "audio_000001.enc")
	assertTempDirEmpty(t, app)
}

func TestHugeConfiguredUploadLimitDoesNotOverflowRequestLimit(t *testing.T) {
	app := newTestAppWithMaxUploadBytes(t, int64(1<<63-1))
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.StatusCode, body)
	}
}

func TestRejectUploadToMissingIncident(t *testing.T) {
	app := newTestApp(t)
	payload := []byte("encrypted audio data")

	response, body := uploadChunk(t, app, "inc_missing", 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing incident status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "incident_not_found")
}
