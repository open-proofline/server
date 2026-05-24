package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"safety-recorder/server/internal/incidents"
)

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
	if token.ExpiresAt == nil {
		t.Fatal("expected explicit expiry to round trip")
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected explicit expiry %s, got %s", expiresAt, token.ExpiresAt)
	}
}

func TestCreateEmergencyTokenAppliesDefaultExpiry(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	before := time.Now().UTC()
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)
	after := time.Now().UTC()

	if token.ExpiresAt == nil {
		t.Fatal("expected omitted expires_at to receive default expiry")
	}
	earliest := before.Add(24 * time.Hour)
	latest := after.Add(24 * time.Hour)
	if token.ExpiresAt.Before(earliest) || token.ExpiresAt.After(latest) {
		t.Fatalf("default expiry = %s, want between %s and %s", token.ExpiresAt, earliest, latest)
	}
}

func TestCreateEmergencyTokenPreservesExplicitNullExpiry(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	response, body := post(t, app, "/v1/incidents/"+incidentID+"/emergency-tokens", "application/json", bytes.NewBufferString(`{"label":"trusted contact","expires_at":null}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create emergency token status 201, got %d: %s", response.StatusCode, body)
	}

	var token emergencyTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		t.Fatalf("decode create emergency token response: %v", err)
	}
	if token.ExpiresAt != nil {
		t.Fatalf("expected explicit null expires_at to remain unset, got %s", token.ExpiresAt)
	}
}

func TestCreateEmergencyTokenCanDisableDefaultExpiry(t *testing.T) {
	app := newTestAppWithDefaultEmergencyTokenTTL(t, 0)
	incidentID := createIncident(t, app, `{}`)

	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	if token.ExpiresAt != nil {
		t.Fatalf("expected omitted expires_at to remain unset when default expiry is disabled, got %s", token.ExpiresAt)
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

func TestEmergencyDataLatestChunkUsesReceivedTimeAcrossStreamScopedIndexes(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)
	firstStream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "first audio")
	secondStream := createMediaStream(t, app, incidentID, incidents.MediaTypeAudio, "second audio")
	firstPayload := []byte("first stream encrypted audio")
	firstLaterIndexPayload := []byte("first stream encrypted audio index two")
	secondPayload := []byte("second stream encrypted audio")

	response, body := uploadChunkWithStream(t, app, incidentID, firstStream.ID, 1, incidents.MediaTypeAudio, firstPayload, sha256Hex(firstPayload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first stream chunk 1 upload status 201, got %d: %s", response.StatusCode, body)
	}
	response, body = uploadChunkWithStream(t, app, incidentID, firstStream.ID, 2, incidents.MediaTypeAudio, firstLaterIndexPayload, sha256Hex(firstLaterIndexPayload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected first stream chunk 2 upload status 201, got %d: %s", response.StatusCode, body)
	}
	response, body = uploadChunkWithStream(t, app, incidentID, secondStream.ID, 1, incidents.MediaTypeAudio, secondPayload, sha256Hex(secondPayload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected second stream chunk 1 upload status 201, got %d: %s", response.StatusCode, body)
	}

	baseTime := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	setChunkCreatedAt(t, app, firstStream.ID, 1, baseTime)
	setChunkCreatedAt(t, app, firstStream.ID, 2, baseTime.Add(time.Second))
	setChunkCreatedAt(t, app, secondStream.ID, 1, baseTime.Add(2*time.Second))

	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)
	response, body = getPublic(t, app, "/e/"+token.Token+"/data")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected emergency data status 200, got %d: %s", response.StatusCode, body)
	}

	var data struct {
		LatestChunkByMediaType map[string]struct {
			ChunkIndex int    `json:"chunk_index"`
			ByteSize   int64  `json:"byte_size"`
			SHA256Hex  string `json:"sha256_hex"`
		} `json:"latest_chunk_by_media_type"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("decode emergency data: %v", err)
	}
	latestAudio := data.LatestChunkByMediaType[incidents.MediaTypeAudio]
	if latestAudio.ChunkIndex != 1 {
		t.Fatalf("expected latest audio chunk to use later stream-local index 1, got %+v", latestAudio)
	}
	if latestAudio.ByteSize != int64(len(secondPayload)) || latestAudio.SHA256Hex != sha256Hex(secondPayload) {
		t.Fatalf("expected latest audio chunk to match second stream payload, got %+v", latestAudio)
	}
}

func setChunkCreatedAt(t *testing.T, app *testApp, streamID string, chunkIndex int, createdAt time.Time) {
	t.Helper()
	result, err := app.db.ExecContext(context.Background(), `
		UPDATE chunks
		SET created_at = ?
		WHERE stream_id = ? AND chunk_index = ?`,
		createdAt.Format(time.RFC3339Nano),
		streamID,
		chunkIndex,
	)
	if err != nil {
		t.Fatalf("update chunk created_at: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("read update rows affected: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("expected one updated chunk row, got %d", rowsAffected)
	}
}
