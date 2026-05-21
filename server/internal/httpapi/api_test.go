package httpapi_test

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"testing"
	"time"

	"safety-recorder/server/internal/db"
	"safety-recorder/server/internal/httpapi"
	"safety-recorder/server/internal/incidents"
	"safety-recorder/server/internal/storage"
)

type testApp struct {
	handler http.Handler
	dataDir string
}

func newTestApp(t *testing.T) *testApp {
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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := httpapi.New(repo, blobStore, httpapi.Options{
		MaxUploadBytes: 1024 * 1024,
		Logger:         logger,
	})

	return &testApp{
		handler: handler,
		dataDir: dataDir,
	}
}

func TestCreateIncident(t *testing.T) {
	app := newTestApp(t)

	incidentID := createIncident(t, app, `{"client_label":"phone","notes":"test"}`)

	if incidentID == "" {
		t.Fatal("expected incident id")
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
}

func TestRejectDuplicateChunkIndex(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	payload := []byte("encrypted audio data")

	response, body := uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first upload status 201, got %d: %s", response.StatusCode, body)
	}

	response, body = uploadChunk(t, app, incidentID, 1, "audio", payload, sha256Hex(payload))
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "duplicate_chunk")
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

	entries, err := os.ReadDir(filepath.Join(app.dataDir, "tmp"))
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected temp dir to be empty, found %d entries", len(entries))
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

func uploadChunk(t *testing.T, app *testApp, incidentID string, index int, mediaType string, payload []byte, hash string) (*http.Response, []byte) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
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

	request := httptest.NewRequest(http.MethodPost, target, body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	app.handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return response, responseBody
}

func get(t *testing.T, app *testApp, target string) (*http.Response, []byte) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	app.handler.ServeHTTP(recorder, request)
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

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
