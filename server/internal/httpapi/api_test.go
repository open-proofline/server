package httpapi_test

import (
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
	db      *sql.DB
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	return newTestAppWithMaxUploadBytes(t, 1024*1024)
}

func newTestAppWithMaxUploadBytes(t *testing.T, maxUploadBytes int64) *testApp {
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
		MaxUploadBytes: maxUploadBytes,
		Logger:         logger,
	})

	return &testApp{
		handler: handler,
		dataDir: dataDir,
		db:      conn,
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

func TestValidEmergencyTokenCanReadIncidentData(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{"client_label":"iphone"}`)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := get(t, app, "/e/"+token.Token)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency page status 200, got %d: %s", response.StatusCode, body)
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

	response, body := get(t, app, "/static/styles.css")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected stylesheet status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(".warning")) {
		t.Fatalf("expected stylesheet content, got: %s", body)
	}

	response, body = get(t, app, "/static/scripts.js")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected script status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte("setInterval")) {
		t.Fatalf("expected script content, got: %s", body)
	}
}

func TestExpiredEmergencyTokenIsRejected(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	expiresAt := time.Now().UTC().Add(-time.Minute)
	token := createEmergencyToken(t, app, incidentID, "expired", &expiresAt)

	response, body := get(t, app, "/e/"+token.Token+"/data")
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

	response, body = get(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected revoked token status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "emergency_token_invalid")
}

func TestInvalidEmergencyTokenIsRejected(t *testing.T) {
	app := newTestApp(t)

	response, body := get(t, app, "/e/not-a-real-token/data")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected invalid token status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "emergency_token_invalid")
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
		response, body := post(t, app, target, "application/json", bytes.NewBufferString(`{"device_network":"cell"}`))
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

	response, body = get(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency data status 200, got %d: %s", response.StatusCode, body)
	}
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
