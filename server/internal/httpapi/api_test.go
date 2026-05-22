package httpapi_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"safety-recorder/server/internal/db"
	"safety-recorder/server/internal/envelope"
	"safety-recorder/server/internal/httpapi"
	"safety-recorder/server/internal/incidents"
	"safety-recorder/server/internal/storage"
)

type testApp struct {
	privateHandler http.Handler
	publicHandler  http.Handler
	dataDir        string
	db             *sql.DB
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	return newTestAppWithMaxUploadBytes(t, 1024*1024)
}

func newTestAppWithMaxUploadBytes(t *testing.T, maxUploadBytes int64) *testApp {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newTestAppWithMaxUploadBytesAndLogger(t, maxUploadBytes, logger)
}

func newTestAppWithMaxUploadBytesAndLogger(t *testing.T, maxUploadBytes int64, logger *slog.Logger) *testApp {
	t.Helper()

	dataDir := t.TempDir()
	conn, err := db.Open(context.Background(), filepath.Join(dataDir, "safety.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	blobStore, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	repo := incidents.NewRepository(conn)
	options := httpapi.Options{
		MaxUploadBytes: maxUploadBytes,
		Logger:         logger,
	}

	return &testApp{
		privateHandler: httpapi.NewPrivate(repo, blobStore, options),
		publicHandler:  httpapi.NewPublic(repo, blobStore, options),
		dataDir:        dataDir,
		db:             conn,
	}
}

func TestCreateIncident(t *testing.T) {
	app := newTestApp(t)

	incidentID := createIncident(t, app, `{"client_label":"phone","notes":"test"}`)

	if incidentID == "" {
		t.Fatal("expected incident id")
	}
}

func TestPrivateAPIJSONSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	response, body := post(t, app, "/v1/incidents", "application/json", bytes.NewBufferString(`{}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create incident status 201, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
}

func TestPrivateAPIErrorSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	response, body := get(t, app, "/v1/incidents/inc_missing")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing incident status 404, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
	assertErrorCode(t, body, "incident_not_found")
}

func TestGetIncidentReturnsEmptyArrays(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	response, body := get(t, app, "/v1/incidents/"+incidentID)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected incident status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(`"chunks":[]`)) {
		t.Fatalf("expected chunks to be an empty array, got: %s", body)
	}
	if !bytes.Contains(body, []byte(`"checkins":[]`)) {
		t.Fatalf("expected checkins to be an empty array, got: %s", body)
	}
}

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

func TestCloseIncident(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/close", "application/json", bytes.NewBufferString(`{}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected close status 200, got %d: %s", response.StatusCode, body)
	}
	var incident incidents.Incident
	if err := json.Unmarshal(body, &incident); err != nil {
		t.Fatalf("decode incident: %v", err)
	}
	if incident.Status != incidents.StatusClosed {
		t.Fatalf("expected closed incident, got %+v", incident)
	}
}

func TestRejectUploadAfterClose(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	response, body := post(t, app, "/v1/incidents/"+incidentID+"/close", "application/json", bytes.NewBufferString(`{}`))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected close status 200, got %d: %s", response.StatusCode, body)
	}

	payload := []byte("encrypted audio data")
	response, body = uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected upload after close status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "incident_closed")
}

func TestListIncidentWithChunksAndCheckins(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{"client_label":"phone"}`)
	payload := []byte("encrypted metadata")

	response, body := uploadChunk(t, app, incidentID, 2, "metadata", payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.StatusCode, body)
	}

	checkinBody := bytes.NewBufferString(`{"device_battery_percent":82,"device_network":"wifi","latitude":-37,"longitude":145,"accuracy_meters":20}`)
	response, body = post(t, app, "/v1/incidents/"+incidentID+"/checkins", "application/json", checkinBody)
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected checkin status 201, got %d: %s", response.StatusCode, body)
	}

	response, body = get(t, app, "/v1/incidents/"+incidentID)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected incident status 200, got %d: %s", response.StatusCode, body)
	}

	var detail incidents.IncidentDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode incident detail: %v", err)
	}
	if detail.Incident.ID != incidentID {
		t.Fatalf("expected incident id %s, got %s", incidentID, detail.Incident.ID)
	}
	if len(detail.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(detail.Chunks))
	}
	if len(detail.Checkins) != 1 {
		t.Fatalf("expected 1 checkin, got %d", len(detail.Checkins))
	}
}

func TestCreateMediaStream(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "main audio recording")

	if stream.ID == "" {
		t.Fatal("expected stream id")
	}
	if stream.IncidentID != incidentID || stream.MediaType != incidents.MediaTypeAudio || stream.Status != incidents.StreamStatusOpen {
		t.Fatalf("unexpected stream: %+v", stream)
	}
	if stream.Label != "main audio recording" {
		t.Fatalf("expected stream label to round trip, got %q", stream.Label)
	}
}

func TestRejectInvalidMediaStreamType(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams", "application/json", bytes.NewBufferString(`{"media_type":"screen"}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid media type status 400, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "invalid_media_type")
}

func TestUploadChunkWithValidStreamID(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio")
	payload := []byte("encrypted audio data")

	response, body := uploadChunkWithStream(t, app, incidentID, stream.ID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected stream upload status 201, got %d: %s", response.StatusCode, body)
	}
	var chunk incidents.Chunk
	if err := json.Unmarshal(body, &chunk); err != nil {
		t.Fatalf("decode chunk: %v", err)
	}
	if chunk.StreamID != stream.ID {
		t.Fatalf("expected chunk stream_id %s, got %q", stream.ID, chunk.StreamID)
	}
}

func TestRejectChunkUploadWhereStreamBelongsToAnotherIncident(t *testing.T) {
	app := newTestApp(t)
	firstIncidentID := createIncident(t, app, `{}`)
	secondIncidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, firstIncidentID, incidents.MediaTypeAudio, "audio")
	payload := []byte("encrypted audio data")

	response, body := uploadChunkWithStream(t, app, secondIncidentID, stream.ID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected wrong-incident stream status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "stream_not_found")
}

func TestRejectChunkUploadWhereStreamMediaTypeDoesNotMatch(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio")
	payload := []byte("encrypted video data")

	response, body := uploadChunkWithStream(t, app, incidentID, stream.ID, 1, "video", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected media mismatch status 400, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "stream_media_type_mismatch")
}

func TestCompleteStreamWithContiguousChunks(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 2)

	updated := completeMediaStream(t, app, incidentID, stream.ID, 2)

	if updated.Status != incidents.StreamStatusComplete {
		t.Fatalf("expected complete stream, got %+v", updated)
	}
	if updated.ExpectedChunkCount == nil || *updated.ExpectedChunkCount != 2 {
		t.Fatalf("expected expected_chunk_count 2, got %+v", updated.ExpectedChunkCount)
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestRejectStreamCompletionWithMissingChunk(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/complete", "application/json", bytes.NewBufferString(`{"expected_chunk_count":2}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected incomplete stream status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "stream_chunks_incomplete")
}

func TestRejectDuplicateStreamCompletion(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/complete", "application/json", bytes.NewBufferString(`{"expected_chunk_count":1}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate completion status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "stream_already_complete")
}

func TestRejectChunkUploadToCompletedStream(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	payload := []byte("late encrypted audio data")

	response, body := uploadChunkWithStream(t, app, incidentID, stream.ID, 2, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected completed stream upload status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "stream_not_open")
}

func TestFailStream(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio")

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/fail", "application/json", bytes.NewBufferString(`{"failure_reason":"client stopped recording unexpectedly"}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected fail stream status 200, got %d: %s", response.StatusCode, body)
	}
	var result struct {
		Stream incidents.MediaStream `json:"stream"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode fail stream response: %v", err)
	}
	if result.Stream.Status != incidents.StreamStatusFailed || result.Stream.FailedAt == nil {
		t.Fatalf("expected failed stream, got %+v", result.Stream)
	}
}

func TestRejectDownloadOfOpenAndFailedStreams(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	openStream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "open audio")
	failedStream := createMediaStream(t, app, incidentID, incidents.MediaTypeVideo, "failed video")

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+failedStream.ID+"/fail", "application/json", bytes.NewBufferString(`{"failure_reason":"stopped"}`))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected fail stream status 200, got %d: %s", response.StatusCode, body)
	}

	for _, stream := range []incidents.MediaStream{openStream, failedStream} {
		response, body := get(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/download")
		response.Body.Close()
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("expected download status 409 for %s stream, got %d: %s", stream.Status, response.StatusCode, body)
		}
		assertErrorCode(t, body, "stream_not_complete")
	}
}

func TestDownloadCompletedStreamBundle(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 2)
	completeMediaStream(t, app, incidentID, stream.ID, 2)

	response, body := get(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/download")
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected stream bundle status 200, got %d: %s", response.StatusCode, body)
	}
	assertBundleHeaders(t, response)
	entries := readZipEntries(t, body)
	assertZipEntry(t, entries, "manifest.json")
	assertZipEntry(t, entries, "chunks/audio_000001.enc")
	assertZipEntry(t, entries, "chunks/audio_000002.enc")

	var manifest struct {
		IncidentID string `json:"incident_id"`
		StreamID   string `json:"stream_id"`
		MediaType  string `json:"media_type"`
		Status     string `json:"status"`
		ChunkCount int    `json:"chunk_count"`
		Chunks     []struct {
			ChunkIndex int    `json:"chunk_index"`
			SHA256Hex  string `json:"sha256_hex"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatalf("decode stream manifest: %v", err)
	}
	if manifest.IncidentID != incidentID || manifest.StreamID != stream.ID || manifest.Status != incidents.StreamStatusComplete || manifest.ChunkCount != 2 {
		t.Fatalf("unexpected stream manifest: %+v", manifest)
	}
	if manifest.Chunks[0].SHA256Hex != sha256Hex(entries["chunks/audio_000001.enc"]) {
		t.Fatalf("first manifest hash does not match zip chunk bytes")
	}
	if manifest.Chunks[1].SHA256Hex != sha256Hex(entries["chunks/audio_000002.enc"]) {
		t.Fatalf("second manifest hash does not match zip chunk bytes")
	}
}

func TestEncryptedEnvelopeChunkRoundTripsThroughOpaqueBackendBundle(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio recording")
	key, err := envelope.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	ctx := envelope.ChunkContext{
		IncidentID: incidentID,
		StreamID:   stream.ID,
		MediaType:  incidents.MediaTypeAudio,
		ChunkIndex: 1,
	}
	payload, err := envelope.EncryptChunk(key, ctx, []byte("plaintext stays client-side"))
	if err != nil {
		t.Fatalf("encrypt chunk: %v", err)
	}

	response, body := uploadChunkWithStream(t, app, incidentID, stream.ID, 1, incidents.MediaTypeAudio, payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected encrypted envelope upload status 201, got %d: %s", response.StatusCode, body)
	}
	completeMediaStream(t, app, incidentID, stream.ID, 1)

	response, body = get(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/download")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected stream bundle status 200, got %d: %s", response.StatusCode, body)
	}
	entries := readZipEntries(t, body)
	bundledChunk := entries["chunks/audio_000001.enc"]
	if !bytes.Equal(bundledChunk, payload) {
		t.Fatal("backend changed encrypted envelope bytes")
	}
	plaintext, err := envelope.DecryptChunk(key, ctx, bundledChunk)
	if err != nil {
		t.Fatalf("decrypt bundled chunk: %v", err)
	}
	if string(plaintext) != "plaintext stays client-side" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}

	var manifest struct {
		Encryption struct {
			Expected       string `json:"expected"`
			Scheme         string `json:"scheme"`
			ServerDecrypts bool   `json:"server_decrypts"`
		} `json:"encryption"`
	}
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Encryption.Expected != "client-side" || manifest.Encryption.Scheme != envelope.SchemeV1 || manifest.Encryption.ServerDecrypts {
		t.Fatalf("unexpected encryption hint: %+v", manifest.Encryption)
	}
}

func TestEmergencyTokenCanDownloadCompletedStreamBundle(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/e/"+token.Token+"/streams/"+stream.ID+"/download")
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency stream download status 200, got %d: %s", response.StatusCode, body)
	}
	assertBundleHeaders(t, response)
	entries := readZipEntries(t, body)
	assertZipEntry(t, entries, "manifest.json")
	assertZipEntry(t, entries, "chunks/audio_000001.enc")
}

func TestInvalidExpiredRevokedEmergencyTokenCannotDownloadBundle(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)

	expiredAt := time.Now().UTC().Add(-time.Minute)
	expired := createEmergencyToken(t, app, incidentID, "expired", &expiredAt)
	revoked := createEmergencyToken(t, app, incidentID, "revoked", nil)
	response, body := post(t, app, "/v1/emergency-tokens/"+revoked.TokenID+"/revoke", "application/json", bytes.NewBufferString(`{}`))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected revoke status 200, got %d: %s", response.StatusCode, body)
	}

	for _, rawToken := range []string{"invalid-token", expired.Token, revoked.Token} {
		response, body := getPublic(t, app, "/e/"+rawToken+"/streams/"+stream.ID+"/download")
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("expected token rejection status 404, got %d: %s", response.StatusCode, body)
		}
		assertErrorCode(t, body, "emergency_token_invalid")
	}
}

func TestEmergencyViewerShowsDownloadButtonsOnlyForCompletedStreams(t *testing.T) {
	app := newTestApp(t)
	incidentID, completed := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, completed.ID, 1)
	failed := createMediaStream(t, app, incidentID, incidents.MediaTypeVideo, "failed video")
	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+failed.ID+"/fail", "application/json", bytes.NewBufferString(`{"failure_reason":"stopped"}`))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected fail stream status 200, got %d: %s", response.StatusCode, body)
	}
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body = getPublic(t, app, "/e/"+token.Token)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency page status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte("Download encrypted bundle")) {
		t.Fatalf("expected completed stream download button: %s", body)
	}
	if !bytes.Contains(body, []byte(completed.Label)) {
		t.Fatalf("expected completed stream label: %s", body)
	}
	if bytes.Contains(body, []byte(failed.Label)) {
		t.Fatalf("failed stream should not have a completed download row: %s", body)
	}
}

func TestEmergencyTokenCanDownloadIncidentBundle(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/e/"+token.Token+"/incident/download")
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency incident download status 200, got %d: %s", response.StatusCode, body)
	}
	assertBundleHeaders(t, response)
	entries := readZipEntries(t, body)
	assertZipEntry(t, entries, "manifest.json")
	assertZipEntry(t, entries, "streams/"+stream.ID+"/manifest.json")
	assertZipEntry(t, entries, "streams/"+stream.ID+"/chunks/audio_000001.enc")
}

func TestCreateEmergencyToken(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	expiresAt := time.Now().UTC().Add(time.Hour)

	token := createEmergencyToken(t, app, incidentID, "trusted contact", &expiresAt)

	if token.TokenID == "" {
		t.Fatal("expected token id")
	}
	if token.Token == "" {
		t.Fatal("expected raw token to be returned once")
	}
	if token.IncidentID != incidentID {
		t.Fatalf("expected incident id %s, got %s", incidentID, token.IncidentID)
	}
	if token.Label != "trusted contact" {
		t.Fatalf("expected label to round trip, got %q", token.Label)
	}
}

func TestEmergencyRawTokenIsNotStored(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	var tokenHash string
	if err := app.db.QueryRowContext(context.Background(), `
		SELECT token_hash
		FROM emergency_tokens
		WHERE id = ?`,
		token.TokenID,
	).Scan(&tokenHash); err != nil {
		t.Fatalf("read token hash: %v", err)
	}
	if tokenHash == token.Token {
		t.Fatal("raw token was stored in token_hash")
	}
	if len(tokenHash) != 64 {
		t.Fatalf("expected SHA-256 hex token hash, got %q", tokenHash)
	}

	var rawMatches int
	if err := app.db.QueryRowContext(context.Background(), `
		SELECT COUNT(*)
		FROM emergency_tokens
		WHERE token_hash = ?`,
		token.Token,
	).Scan(&rawMatches); err != nil {
		t.Fatalf("count raw token rows: %v", err)
	}
	if rawMatches != 0 {
		t.Fatalf("raw token matched %d stored rows", rawMatches)
	}
}

func TestPublicServerDoesNotMountPrivateRoutes(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		method string
		target string
	}{
		{http.MethodPost, "/v1/incidents"},
		{http.MethodGet, "/v1/incidents/inc_missing"},
		{http.MethodPost, "/v1/incidents/inc_missing/chunks"},
		{http.MethodGet, "/v1/incidents/inc_missing/chunks"},
		{http.MethodGet, "/v1/incidents/inc_missing/chunks/audio/0"},
		{http.MethodGet, "/v1/incidents/inc_missing/download"},
		{http.MethodPost, "/v1/incidents/inc_missing/streams"},
		{http.MethodGet, "/v1/incidents/inc_missing/streams"},
		{http.MethodGet, "/v1/incidents/inc_missing/streams/str_missing"},
		{http.MethodPost, "/v1/incidents/inc_missing/streams/str_missing/complete"},
		{http.MethodPost, "/v1/incidents/inc_missing/streams/str_missing/fail"},
		{http.MethodGet, "/v1/incidents/inc_missing/streams/str_missing/download"},
		{http.MethodPost, "/v1/incidents/inc_missing/checkins"},
		{http.MethodPost, "/v1/incidents/inc_missing/close"},
		{http.MethodPost, "/v1/incidents/inc_missing/emergency-tokens"},
		{http.MethodPost, "/v1/emergency-tokens/etk_missing/revoke"},
	}

	for _, tt := range tests {
		response, body := request(t, app.publicHandler, tt.method, tt.target, "application/json", bytes.NewBufferString(`{}`))
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("%s %s: expected public server status 404, got %d: %s", tt.method, tt.target, response.StatusCode, body)
		}
	}
}

func TestPublicNotFoundUsesSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	response, body := getPublic(t, app, "/missing")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected public 404 status, got %d: %s", response.StatusCode, body)
	}
	assertPublicBrowserSecurityHeaders(t, response)
	assertNoStore(t, response)
	assertErrorCode(t, body, "not_found")
}

func TestPrivateServerDoesNotMountPublicEmergencyRoutes(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	for _, target := range []string{
		"/e/" + token.Token,
		"/e/" + token.Token + "/data",
		"/e/" + token.Token + "/streams/str_missing/download",
		"/e/" + token.Token + "/incident/download",
	} {
		response, body := get(t, app, target)
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("GET %s: expected private server status 404, got %d: %s", target, response.StatusCode, body)
		}
	}
}

func TestValidEmergencyTokenCanReadIncidentData(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{"client_label":"iphone"}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/e/"+token.Token)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency page status 200, got %d: %s", response.StatusCode, body)
	}
	assertContentTypePrefix(t, response, "text/html")
	assertEmergencyPrivacyHeaders(t, response)
	if !bytes.Contains(body, []byte(`name="referrer" content="no-referrer"`)) {
		t.Fatalf("expected no-referrer meta tag in response: %s", body)
	}
	if !bytes.Contains(body, []byte("Emergency Incident Viewer")) {
		t.Fatalf("expected emergency page title in response: %s", body)
	}
	if !bytes.Contains(body, []byte(`/static/styles.css`)) {
		t.Fatalf("expected static stylesheet link in response: %s", body)
	}
	if !bytes.Contains(body, []byte(`/static/scripts.js`)) {
		t.Fatalf("expected static script tag in response: %s", body)
	}
	if bytes.Contains(body, []byte("<style>")) || bytes.Contains(body, []byte("setInterval(function")) {
		t.Fatalf("expected no inline style or script in emergency page: %s", body)
	}
	if !bytes.Contains(body, []byte("iphone")) {
		t.Fatalf("expected client label in response: %s", body)
	}
	if !bytes.Contains(body, []byte("Last updated")) || !bytes.Contains(body, []byte("just now")) {
		t.Fatalf("expected human-friendly relative timestamp in response: %s", body)
	}
	if !bytes.Contains(body, []byte("call emergency services")) {
		t.Fatalf("expected emergency warning in response: %s", body)
	}
}

func TestEmergencyStaticAssetsAreServed(t *testing.T) {
	app := newTestApp(t)

	response, body := getPublic(t, app, "/static/styles.css")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected stylesheet status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(".warning")) {
		t.Fatalf("expected stylesheet content, got: %s", body)
	}
	assertContentTypePrefix(t, response, "text/css")
	assertPublicBrowserSecurityHeaders(t, response)
	assertNoStrictTransportSecurity(t, response)

	response, body = getPublic(t, app, "/static/scripts.js")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected script status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte("setInterval")) {
		t.Fatalf("expected script content, got: %s", body)
	}
	assertContentTypeContains(t, response, "javascript")
	assertPublicBrowserSecurityHeaders(t, response)
	assertNoStrictTransportSecurity(t, response)
}

func TestExpiredEmergencyTokenIsRejected(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	expiresAt := time.Now().UTC().Add(-time.Minute)
	token := createEmergencyToken(t, app, incidentID, "expired", &expiresAt)

	response, body := getPublic(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected expired token status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "emergency_token_invalid")
}

func TestRevokedEmergencyTokenIsRejected(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := post(t, app, "/v1/emergency-tokens/"+token.TokenID+"/revoke", "application/json", bytes.NewBufferString(`{}`))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected revoke status 200, got %d: %s", response.StatusCode, body)
	}

	response, body = getPublic(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected revoked token status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "emergency_token_invalid")
}

func TestInvalidEmergencyTokenIsRejected(t *testing.T) {
	app := newTestApp(t)

	response, body := getPublic(t, app, "/e/not-a-real-token/data")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected invalid token status 404, got %d: %s", response.StatusCode, body)
	}
	assertEmergencyPrivacyHeaders(t, response)
	assertErrorCode(t, body, "emergency_token_invalid")
}

func TestEmergencyTokenIsRedactedFromRequestLogs(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	app := newTestAppWithMaxUploadBytesAndLogger(t, 1024*1024, logger)
	incidentID := createIncident(t, app, `{}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/e/"+token.Token)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency page status 200, got %d: %s", response.StatusCode, body)
	}

	if bytes.Contains(logs.Bytes(), []byte(token.Token)) {
		t.Fatalf("request logs exposed raw token: %s", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("/e/{token}")) {
		t.Fatalf("expected redacted emergency path in request logs: %s", logs.String())
	}
}

func TestEmergencyTokenCannotMutateIncidentChunkOrCheckinData(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")
	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.StatusCode, body)
	}
	createCheckin(t, app, incidentID)
	before := getIncidentDetail(t, app, incidentID)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	for _, target := range []string{"/e/" + token.Token, "/e/" + token.Token + "/data", "/e/" + token.Token + "/checkins"} {
		response, body := postPublic(t, app, target, "application/json", bytes.NewBufferString(`{"device_network":"cell"}`))
		response.Body.Close()
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			t.Fatalf("expected POST %s to fail, got %d: %s", target, response.StatusCode, body)
		}
	}

	after := getIncidentDetail(t, app, incidentID)
	if before.Incident.Status != after.Incident.Status {
		t.Fatalf("incident status changed from %s to %s", before.Incident.Status, after.Incident.Status)
	}
	if len(before.Chunks) != len(after.Chunks) {
		t.Fatalf("chunk count changed from %d to %d", len(before.Chunks), len(after.Chunks))
	}
	if len(before.Checkins) != len(after.Checkins) {
		t.Fatalf("checkin count changed from %d to %d", len(before.Checkins), len(after.Checkins))
	}
}

func TestEmergencyViewerReadsDoNotMutateEmergencyTokenRows(t *testing.T) {
	app := newTestApp(t)
	assertEmergencyTokenColumnMissing(t, app, "last_used_at")

	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)
	before := emergencyTokenRows(t, app)

	targets := []string{
		"/e/" + token.Token,
		"/e/" + token.Token + "/data",
		"/e/" + token.Token + "/streams/" + stream.ID + "/download",
		"/e/" + token.Token + "/incident/download",
	}
	for _, target := range targets {
		response, body := getPublic(t, app, target)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: expected status 200, got %d: %s", target, response.StatusCode, body)
		}
	}

	after := emergencyTokenRows(t, app)
	if len(before) != len(after) {
		t.Fatalf("emergency token row count changed from %d to %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("emergency token row changed from %+v to %+v", before[i], after[i])
		}
	}
}

func TestEmergencyDataReturnsExpectedReadOnlyJSON(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{"client_label":"iphone"}`)
	payload := []byte("encrypted metadata")
	response, body := uploadChunk(t, app, incidentID, 2, "metadata", payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.StatusCode, body)
	}
	createCheckin(t, app, incidentID)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body = getPublic(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency data status 200, got %d: %s", response.StatusCode, body)
	}
	assertEmergencyPrivacyHeaders(t, response)
	if bytes.Contains(body, []byte("stored_path")) {
		t.Fatalf("emergency data exposed storage path: %s", body)
	}
	if bytes.Contains(body, []byte(token.Token)) {
		t.Fatalf("emergency data exposed raw token: %s", body)
	}

	var data struct {
		Incident struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			ClientLabel string `json:"client_label"`
		} `json:"incident"`
		LatestCheckin *struct {
			DeviceBatteryPercent *int    `json:"device_battery_percent"`
			DeviceNetwork        *string `json:"device_network"`
		} `json:"latest_checkin"`
		ChunkCountByMediaType map[string]int `json:"chunk_count_by_media_type"`
		Media                 []struct {
			MediaType  string `json:"media_type"`
			ChunkCount int    `json:"chunk_count"`
		} `json:"media"`
		Warning string `json:"warning"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("decode emergency data: %v", err)
	}
	if data.Incident.ID != incidentID || data.Incident.Status != incidents.StatusOpen || data.Incident.ClientLabel != "iphone" {
		t.Fatalf("unexpected incident summary: %+v", data.Incident)
	}
	if data.LatestCheckin == nil || data.LatestCheckin.DeviceBatteryPercent == nil || *data.LatestCheckin.DeviceBatteryPercent != 82 {
		t.Fatalf("unexpected latest checkin: %+v", data.LatestCheckin)
	}
	if data.ChunkCountByMediaType["metadata"] != 1 {
		t.Fatalf("expected one metadata chunk, got %+v", data.ChunkCountByMediaType)
	}
	if data.Warning == "" {
		t.Fatal("expected emergency warning")
	}
}

func createIncident(t *testing.T, app *testApp, requestBody string) string {
	t.Helper()

	response, body := post(t, app, "/v1/incidents", "application/json", bytes.NewBufferString(requestBody))
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create incident status 201, got %d: %s", response.StatusCode, body)
	}

	var result struct {
		IncidentID string `json:"incident_id"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode create incident response: %v", err)
	}
	if result.Status != incidents.StatusOpen {
		t.Fatalf("expected status open, got %q", result.Status)
	}
	return result.IncidentID
}

type emergencyTokenResponse struct {
	TokenID    string     `json:"token_id"`
	IncidentID string     `json:"incident_id"`
	Token      string     `json:"token"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func createEmergencyToken(t *testing.T, app *testApp, incidentID string, label string, expiresAt *time.Time) emergencyTokenResponse {
	t.Helper()

	requestBody, err := json.Marshal(struct {
		Label     string     `json:"label"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}{
		Label:     label,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("marshal emergency token request: %v", err)
	}

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/emergency-tokens", "application/json", bytes.NewReader(requestBody))
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create emergency token status 201, got %d: %s", response.StatusCode, body)
	}
	assertNoStore(t, response)

	var result emergencyTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode create emergency token response: %v", err)
	}
	return result
}

func createCheckin(t *testing.T, app *testApp, incidentID string) {
	t.Helper()

	checkinBody := bytes.NewBufferString(`{"device_battery_percent":82,"device_network":"wifi","latitude":-37,"longitude":145,"accuracy_meters":20}`)
	response, body := post(t, app, "/v1/incidents/"+incidentID+"/checkins", "application/json", checkinBody)
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected checkin status 201, got %d: %s", response.StatusCode, body)
	}
}

func getIncidentDetail(t *testing.T, app *testApp, incidentID string) incidents.IncidentDetail {
	t.Helper()

	response, body := get(t, app, "/v1/incidents/"+incidentID)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected incident status 200, got %d: %s", response.StatusCode, body)
	}
	var detail incidents.IncidentDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode incident detail: %v", err)
	}
	return detail
}

func createMediaStream(t *testing.T, app *testApp, incidentID, mediaType, label string) incidents.MediaStream {
	t.Helper()

	requestBody, err := json.Marshal(struct {
		MediaType string `json:"media_type"`
		Label     string `json:"label"`
	}{
		MediaType: mediaType,
		Label:     label,
	})
	if err != nil {
		t.Fatalf("marshal media stream request: %v", err)
	}
	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams", "application/json", bytes.NewReader(requestBody))
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create media stream status 201, got %d: %s", response.StatusCode, body)
	}

	var result struct {
		Stream incidents.MediaStream `json:"stream"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode create stream response: %v", err)
	}
	return result.Stream
}

func completeMediaStream(t *testing.T, app *testApp, incidentID, streamID string, expectedChunkCount int) incidents.MediaStream {
	t.Helper()

	requestBody, err := json.Marshal(struct {
		ExpectedChunkCount int `json:"expected_chunk_count"`
	}{ExpectedChunkCount: expectedChunkCount})
	if err != nil {
		t.Fatalf("marshal complete stream request: %v", err)
	}
	response, body := post(t, app, "/v1/incidents/"+incidentID+"/streams/"+streamID+"/complete", "application/json", bytes.NewReader(requestBody))
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected complete media stream status 200, got %d: %s", response.StatusCode, body)
	}

	var result struct {
		Stream incidents.MediaStream `json:"stream"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode complete stream response: %v", err)
	}
	return result.Stream
}

func createIncidentStreamWithChunks(t *testing.T, app *testApp, chunkCount int) (string, incidents.MediaStream) {
	t.Helper()

	incidentID := createIncident(t, app, `{}`)
	stream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "audio recording")
	for index := 1; index <= chunkCount; index++ {
		payload := []byte("encrypted audio data " + strconv.Itoa(index))
		response, body := uploadChunkWithStream(t, app, incidentID, stream.ID, index, incidents.MediaTypeAudio, payload, sha256Hex(payload))
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("expected stream chunk upload status 201, got %d: %s", response.StatusCode, body)
		}
	}
	return incidentID, stream
}

func uploadChunk(t *testing.T, app *testApp, incidentID string, index int, mediaType string, payload []byte, hash string) (*http.Response, []byte) {
	t.Helper()

	return uploadChunkWithStream(t, app, incidentID, "", index, mediaType, payload, hash)
}

func uploadChunkWithStream(t *testing.T, app *testApp, incidentID string, streamID string, index int, mediaType string, payload []byte, hash string) (*http.Response, []byte) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if streamID != "" {
		must(t, writer.WriteField("stream_id", streamID))
	}
	must(t, writer.WriteField("chunk_index", strconv.Itoa(index)))
	must(t, writer.WriteField("media_type", mediaType))
	startedAt := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	must(t, writer.WriteField("started_at", startedAt.Format(time.RFC3339Nano)))
	must(t, writer.WriteField("ended_at", startedAt.Add(time.Second).Format(time.RFC3339Nano)))
	must(t, writer.WriteField("sha256_hex", hash))
	must(t, writer.WriteField("original_filename", "chunk.enc"))
	fileWriter, err := writer.CreateFormFile("file", "upload.enc")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	must(t, writer.Close())

	return post(t, app, "/v1/incidents/"+incidentID+"/chunks", writer.FormDataContentType(), &body)
}

func post(t *testing.T, app *testApp, target string, contentType string, body io.Reader) (*http.Response, []byte) {
	t.Helper()

	return request(t, app.privateHandler, http.MethodPost, target, contentType, body)
}

func postPublic(t *testing.T, app *testApp, target string, contentType string, body io.Reader) (*http.Response, []byte) {
	t.Helper()

	return request(t, app.publicHandler, http.MethodPost, target, contentType, body)
}

func get(t *testing.T, app *testApp, target string) (*http.Response, []byte) {
	t.Helper()

	return request(t, app.privateHandler, http.MethodGet, target, "", nil)
}

func getPublic(t *testing.T, app *testApp, target string) (*http.Response, []byte) {
	t.Helper()

	return request(t, app.publicHandler, http.MethodGet, target, "", nil)
}

func request(t *testing.T, handler http.Handler, method string, target string, contentType string, body io.Reader) (*http.Response, []byte) {
	t.Helper()

	request := httptest.NewRequest(method, target, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return response, responseBody
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func stringsOf(value string, count int) string {
	var builder bytes.Buffer
	for range count {
		builder.WriteString(value)
	}
	return builder.String()
}

func assertEmergencyPrivacyHeaders(t *testing.T, response *http.Response) {
	t.Helper()

	assertPublicBrowserSecurityHeaders(t, response)
	assertNoStore(t, response)
}

func assertPublicBrowserSecurityHeaders(t *testing.T, response *http.Response) {
	t.Helper()

	assertNoSniff(t, response)
	if response.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected no-referrer policy, got %q", response.Header.Get("Referrer-Policy"))
	}
	csp := response.Header.Get("Content-Security-Policy")
	for _, directive := range []string{"default-src 'self'", "base-uri 'none'", "frame-ancestors 'none'", "form-action 'self'", "object-src 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("expected CSP directive %q in %q", directive, csp)
		}
	}
	if response.Header.Get("Permissions-Policy") != "geolocation=(), microphone=(), camera=()" {
		t.Fatalf("expected restricted permissions policy, got %q", response.Header.Get("Permissions-Policy"))
	}
	if response.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", response.Header.Get("X-Frame-Options"))
	}
}

func assertPrivateJSONSecurityHeaders(t *testing.T, response *http.Response) {
	t.Helper()

	if response.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %q", response.Header.Get("Content-Type"))
	}
	assertNoSniff(t, response)
	assertNoStore(t, response)
}

func assertContentTypePrefix(t *testing.T, response *http.Response, prefix string) {
	t.Helper()

	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, prefix) {
		t.Fatalf("expected Content-Type prefix %q, got %q", prefix, contentType)
	}
}

func assertContentTypeContains(t *testing.T, response *http.Response, value string) {
	t.Helper()

	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, value) {
		t.Fatalf("expected Content-Type containing %q, got %q", value, contentType)
	}
}

func assertNoSniff(t *testing.T, response *http.Response) {
	t.Helper()

	if response.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff, got %q", response.Header.Get("X-Content-Type-Options"))
	}
}

func assertNoStore(t *testing.T, response *http.Response) {
	t.Helper()

	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache policy, got %q", response.Header.Get("Cache-Control"))
	}
}

func assertNoStrictTransportSecurity(t *testing.T, response *http.Response) {
	t.Helper()

	if value := response.Header.Get("Strict-Transport-Security"); value != "" {
		t.Fatalf("expected no app-level Strict-Transport-Security header in local/dev HTTP mode, got %q", value)
	}
}

func assertBundleHeaders(t *testing.T, response *http.Response) {
	t.Helper()

	if response.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("expected application/zip, got %q", response.Header.Get("Content-Type"))
	}
	if !strings.HasPrefix(response.Header.Get("Content-Disposition"), `attachment; filename="incident_`) {
		t.Fatalf("expected attachment content disposition, got %q", response.Header.Get("Content-Disposition"))
	}
	assertEmergencyPrivacyHeaders(t, response)
}

func assertErrorCode(t *testing.T, body []byte, expected string) {
	t.Helper()

	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, response.Error.Code)
	}
}

func readZipEntries(t *testing.T, body []byte) map[string][]byte {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("open zip response: %v", err)
	}
	entries := make(map[string][]byte)
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}
		entryBody, err := io.ReadAll(handle)
		if closeErr := handle.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatalf("read zip entry %s: %v", file.Name, err)
		}
		entries[file.Name] = entryBody
	}
	return entries
}

func assertZipEntry(t *testing.T, entries map[string][]byte, name string) {
	t.Helper()

	if _, ok := entries[name]; !ok {
		t.Fatalf("expected zip entry %q, got entries %+v", name, entries)
	}
}

type emergencyTokenDBRow struct {
	ID         string
	IncidentID string
	TokenHash  string
	Label      string
	CreatedAt  string
	ExpiresAt  string
	RevokedAt  string
}

func emergencyTokenRows(t *testing.T, app *testApp) []emergencyTokenDBRow {
	t.Helper()

	rows, err := app.db.QueryContext(context.Background(), `
		SELECT id, incident_id, token_hash, COALESCE(label, ''),
			created_at, COALESCE(expires_at, ''), COALESCE(revoked_at, '')
		FROM emergency_tokens
		ORDER BY id`)
	if err != nil {
		t.Fatalf("query emergency token rows: %v", err)
	}
	defer rows.Close()

	tokenRows := []emergencyTokenDBRow{}
	for rows.Next() {
		var row emergencyTokenDBRow
		if err := rows.Scan(
			&row.ID,
			&row.IncidentID,
			&row.TokenHash,
			&row.Label,
			&row.CreatedAt,
			&row.ExpiresAt,
			&row.RevokedAt,
		); err != nil {
			t.Fatalf("scan emergency token row: %v", err)
		}
		tokenRows = append(tokenRows, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate emergency token rows: %v", err)
	}
	return tokenRows
}

func assertEmergencyTokenColumnMissing(t *testing.T, app *testApp, column string) {
	t.Helper()

	rows, err := app.db.QueryContext(context.Background(), `PRAGMA table_info(emergency_tokens)`)
	if err != nil {
		t.Fatalf("query emergency token schema: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan emergency token schema: %v", err)
		}
		if name == column {
			t.Fatalf("emergency_tokens still has removed column %q", column)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate emergency token schema: %v", err)
	}
}

func assertNoStoredFile(t *testing.T, app *testApp, incidentID, filename string) {
	t.Helper()

	storedPath := filepath.Join(app.dataDir, "incidents", incidentID, filename)
	if _, err := os.Stat(storedPath); !os.IsNotExist(err) {
		t.Fatalf("expected no stored file at %s, got err %v", storedPath, err)
	}
}

func assertTempDirEmpty(t *testing.T, app *testApp) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(app.dataDir, "tmp"))
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected temp dir to be empty, found %d entries", len(entries))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
