package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"safety-recorder/server/internal/envelope"
)

func TestParseByteSize(t *testing.T) {
	tests := map[string]int64{
		"1":     1,
		"64KiB": 64 * 1024,
		"2MiB":  2 * 1024 * 1024,
		"3MB":   3 * 1000 * 1000,
		"4 gb":  4 * 1000 * 1000 * 1000,
	}

	for input, want := range tests {
		got, err := parseByteSize(input)
		if err != nil {
			t.Fatalf("parseByteSize(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("parseByteSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseByteSizeRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"", "KiB", "12XB"} {
		if _, err := parseByteSize(input); err == nil {
			t.Fatalf("parseByteSize(%q) succeeded, want error", input)
		}
	}
}

func TestBuildViewerURL(t *testing.T) {
	got := buildViewerURL("http://localhost:8081/", "abc/123")
	want := "http://localhost:8081/e/abc%2F123"
	if got != want {
		t.Fatalf("buildViewerURL returned %q, want %q", got, want)
	}
}

func TestClientWriteRoutesUsePrivateAPIBase(t *testing.T) {
	expectedPaths := []string{
		"POST /v1/incidents",
		"POST /v1/incidents/inc_1/emergency-tokens",
		"POST /v1/incidents/inc_1/streams",
		"POST /v1/incidents/inc_1/checkins",
		"POST /v1/incidents/inc_1/streams/str_1/complete",
		"POST /v1/incidents/inc_1/close",
	}
	var gotPaths []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.example" {
			t.Fatalf("write route used host %q, want api.example", r.URL.Host)
		}
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/v1/incidents":
			return testResponse(http.StatusCreated, "application/json", `{"incident_id":"inc_1","status":"open"}`), nil
		case "/v1/incidents/inc_1/emergency-tokens":
			return testResponse(http.StatusCreated, "application/json", `{"token_id":"etk_1","incident_id":"inc_1","token":"tok_1","created_at":"2026-05-22T10:00:00Z"}`), nil
		case "/v1/incidents/inc_1/streams":
			return testResponse(http.StatusCreated, "application/json", `{"stream":{"id":"str_1","incident_id":"inc_1","media_type":"audio","status":"open","created_at":"2026-05-22T10:00:00Z","updated_at":"2026-05-22T10:00:00Z"}}`), nil
		case "/v1/incidents/inc_1/checkins":
			return testResponse(http.StatusCreated, "application/json", `{"id":"chk_1"}`), nil
		case "/v1/incidents/inc_1/streams/str_1/complete":
			return testResponse(http.StatusOK, "application/json", `{"stream":{"id":"str_1","incident_id":"inc_1","media_type":"audio","status":"complete","created_at":"2026-05-22T10:00:00Z","updated_at":"2026-05-22T10:01:00Z","completed_at":"2026-05-22T10:01:00Z"}}`), nil
		case "/v1/incidents/inc_1/close":
			return testResponse(http.StatusOK, "application/json", `{"id":"inc_1","status":"closed"}`), nil
		default:
			return testResponse(http.StatusNotFound, "application/json", `{"error":{"code":"not_found"}}`), nil
		}
	})}

	sim := client{
		httpClient: httpClient,
		apiBase:    "http://api.example",
		viewerBase: "http://viewer.example",
	}
	ctx := context.Background()

	incidentID, err := sim.createIncident(ctx)
	if err != nil {
		t.Fatalf("createIncident returned error: %v", err)
	}
	if incidentID != "inc_1" {
		t.Fatalf("incidentID = %q", incidentID)
	}
	token, err := sim.createEmergencyToken(ctx, incidentID)
	if err != nil {
		t.Fatalf("createEmergencyToken returned error: %v", err)
	}
	if token != "tok_1" {
		t.Fatalf("token = %q", token)
	}
	streamID, err := sim.createMediaStream(ctx, incidentID, "audio")
	if err != nil {
		t.Fatalf("createMediaStream returned error: %v", err)
	}
	if streamID != "str_1" {
		t.Fatalf("streamID = %q", streamID)
	}
	if err := sim.createCheckin(ctx, incidentID, 1); err != nil {
		t.Fatalf("createCheckin returned error: %v", err)
	}
	if err := sim.completeMediaStream(ctx, incidentID, streamID, 1); err != nil {
		t.Fatalf("completeMediaStream returned error: %v", err)
	}
	if err := sim.closeIncident(ctx, incidentID); err != nil {
		t.Fatalf("closeIncident returned error: %v", err)
	}

	if len(gotPaths) != len(expectedPaths) {
		t.Fatalf("got paths %v, want %v", gotPaths, expectedPaths)
	}
	for i, want := range expectedPaths {
		if gotPaths[i] != want {
			t.Fatalf("path %d = %q, want %q; all paths: %v", i, gotPaths[i], want, gotPaths)
		}
	}
}

func TestClientUploadChunkUsesPrivateAPIBaseAndStreamID(t *testing.T) {
	body := []byte("encrypted bytes")
	upload := buildChunkUpload("inc_1", "str_1", 2, "audio", time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC), body)

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.example" {
			t.Fatalf("upload used host %q, want api.example", r.URL.Host)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/incidents/inc_1/chunks" {
			t.Fatalf("unexpected upload route %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("ParseMultipartForm returned error: %v", err)
		}
		if got := r.FormValue("stream_id"); got != "str_1" {
			t.Fatalf("stream_id = %q", got)
		}
		if got := r.FormValue("chunk_index"); got != "2" {
			t.Fatalf("chunk_index = %q", got)
		}
		if got := r.FormValue("sha256_hex"); got != sha256Hex(body) {
			t.Fatalf("sha256_hex = %q", got)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile returned error: %v", err)
		}
		defer file.Close()
		fileBody, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		if !bytes.Equal(fileBody, body) {
			t.Fatalf("uploaded file body = %q, want %q", fileBody, body)
		}
		return testResponse(http.StatusCreated, "application/json", `{"id":"chunk_1"}`), nil
	})}

	sim := client{
		httpClient: httpClient,
		apiBase:    "http://api.example",
		viewerBase: "http://viewer.example",
	}
	if err := sim.uploadChunk(context.Background(), upload); err != nil {
		t.Fatalf("uploadChunk returned error: %v", err)
	}
}

func TestClientDownloadStreamBundleUsesViewerBase(t *testing.T) {
	bundleBytes := []byte("zip bytes")
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "viewer.example" {
			t.Fatalf("bundle download used host %q, want viewer.example", r.URL.Host)
		}
		if r.Method != http.MethodGet || r.URL.Path != "/e/tok_1/streams/str_1/download" {
			t.Fatalf("unexpected viewer route %s %s", r.Method, r.URL.Path)
		}
		return testResponse(http.StatusOK, "application/zip", string(bundleBytes)), nil
	})}

	sim := client{
		httpClient: httpClient,
		apiBase:    "http://api.example",
		viewerBase: "http://viewer.example",
	}
	got, err := sim.downloadStreamBundle(context.Background(), "tok_1", "str_1")
	if err != nil {
		t.Fatalf("downloadStreamBundle returned error: %v", err)
	}
	if !bytes.Equal(got, bundleBytes) {
		t.Fatalf("bundle body = %q, want %q", got, bundleBytes)
	}
}

func TestParseConfigStreamFlags(t *testing.T) {
	cfg, err := parseConfig([]string{"--chunks", "2", "--interval", "0", "--download-bundle"})
	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if !cfg.completeStream {
		t.Fatal("expected complete-stream to default true")
	}
	if !cfg.downloadBundle {
		t.Fatal("expected download-bundle flag to be set")
	}
}

func TestParseConfigEncryptionDefaults(t *testing.T) {
	cfg, err := parseConfig([]string{"--chunks", "2", "--interval", "0"})
	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if !cfg.encrypt {
		t.Fatal("expected encryption to default true")
	}
	if !cfg.verifyBundleDecrypt {
		t.Fatal("expected bundle decryption verification to default true")
	}
	if cfg.keyFile != "" {
		t.Fatalf("keyFile = %q, want empty", cfg.keyFile)
	}
}

func TestParseConfigAllowsEncryptionToBeDisabled(t *testing.T) {
	cfg, err := parseConfig([]string{"--encrypt=false"})
	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if cfg.encrypt {
		t.Fatal("expected encryption to be disabled")
	}
}

func TestParseConfigCleansBasesAndChunkSize(t *testing.T) {
	cfg, err := parseConfig([]string{
		"--api", "http://private.example/",
		"--viewer", "http://public.example/",
		"--chunk-size", "2KiB",
	})
	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if cfg.apiBase != "http://private.example" {
		t.Fatalf("apiBase = %q", cfg.apiBase)
	}
	if cfg.viewerBase != "http://public.example" {
		t.Fatalf("viewerBase = %q", cfg.viewerBase)
	}
	if cfg.chunkSize != 2048 {
		t.Fatalf("chunkSize = %d", cfg.chunkSize)
	}
}

func TestParseConfigRejectsDownloadWithoutCompleteStream(t *testing.T) {
	if _, err := parseConfig([]string{"--download-bundle", "--complete-stream=false"}); err == nil {
		t.Fatal("expected --download-bundle without --complete-stream to fail")
	}
}

func TestParseConfigRejectsDownloadWithoutChunks(t *testing.T) {
	if _, err := parseConfig([]string{"--chunks", "0", "--download-bundle"}); err == nil {
		t.Fatal("expected --download-bundle without chunks to fail")
	}
}

func TestParseConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "negative chunks",
			args: []string{"--chunks", "-1"},
		},
		{
			name: "negative interval",
			args: []string{"--interval", "-1s"},
		},
		{
			name: "invalid media type",
			args: []string{"--media-type", "screen"},
		},
		{
			name: "zero chunk size",
			args: []string{"--chunk-size", "0"},
		},
		{
			name: "negative failure interval",
			args: []string{"--simulate-failure-every", "-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseConfig(tt.args); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBadHashFor(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	bad := badHashFor(hash)
	if bad == hash {
		t.Fatal("badHashFor returned original hash")
	}
	if len(bad) != 64 {
		t.Fatalf("badHashFor returned length %d, want 64", len(bad))
	}
}

func TestShouldSimulateFailure(t *testing.T) {
	if !shouldSimulateFailure(4, 4) {
		t.Fatal("expected every fourth chunk to simulate failure")
	}
	if shouldSimulateFailure(3, 4) {
		t.Fatal("did not expect third chunk to simulate failure")
	}
	if shouldSimulateFailure(4, 0) {
		t.Fatal("did not expect failure when disabled")
	}
}

func TestShouldSendCheckin(t *testing.T) {
	tests := []struct {
		index int
		want  bool
	}{
		{index: 1, want: true},
		{index: 2, want: false},
		{index: 3, want: true},
		{index: 4, want: false},
		{index: 6, want: true},
	}

	for _, tt := range tests {
		if got := shouldSendCheckin(tt.index); got != tt.want {
			t.Fatalf("shouldSendCheckin(%d) = %v, want %v", tt.index, got, tt.want)
		}
	}
}

func TestNewChunkUploadIncludesStreamAndHashMetadata(t *testing.T) {
	startedAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upload, err := newChunkUpload("incident-test", "stream-test", 2, "audio", 16, startedAt)
	if err != nil {
		t.Fatalf("newChunkUpload returned error: %v", err)
	}

	if upload.incidentID != "incident-test" {
		t.Fatalf("incidentID = %q", upload.incidentID)
	}
	if upload.streamID != "stream-test" {
		t.Fatalf("streamID = %q", upload.streamID)
	}
	if upload.mediaType != "audio" {
		t.Fatalf("mediaType = %q", upload.mediaType)
	}
	if upload.chunkIndex != 2 {
		t.Fatalf("chunkIndex = %d", upload.chunkIndex)
	}
	if upload.filename != "audio_000002.enc" {
		t.Fatalf("filename = %q", upload.filename)
	}
	if len(upload.body) != 16 {
		t.Fatalf("body length = %d", len(upload.body))
	}
	if upload.startedAt != startedAt.Add(chunkDuration) {
		t.Fatalf("startedAt = %s", upload.startedAt)
	}
	if upload.endedAt != upload.startedAt.Add(chunkDuration) {
		t.Fatalf("endedAt = %s, startedAt = %s", upload.endedAt, upload.startedAt)
	}

	sum := sha256.Sum256(upload.body)
	if upload.sha256Hex != hex.EncodeToString(sum[:]) {
		t.Fatalf("sha256Hex = %q", upload.sha256Hex)
	}
}

func TestLoadOrCreateSimulatorKeyCreatesAndLoadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sim.key.json")

	created, err := loadOrCreateSimulatorKey(path)
	if err != nil {
		t.Fatalf("loadOrCreateSimulatorKey create returned error: %v", err)
	}
	if created.KeyID == "" || len(created.Key) != 32 {
		t.Fatalf("unexpected created key: %+v", created)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected key file to be created: %v", err)
	}

	loaded, err := loadOrCreateSimulatorKey(path)
	if err != nil {
		t.Fatalf("loadOrCreateSimulatorKey load returned error: %v", err)
	}
	if loaded.KeyID != created.KeyID || !bytes.Equal(loaded.Key, created.Key) {
		t.Fatal("loaded key did not match created key")
	}
}

func TestNewEncryptedChunkUploadCanDecryptEnvelope(t *testing.T) {
	key, err := envelope.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	startedAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upload, err := newEncryptedChunkUpload(key, "incident-test", "stream-test", 1, "audio", 16, startedAt)
	if err != nil {
		t.Fatalf("newEncryptedChunkUpload returned error: %v", err)
	}
	if len(upload.body) <= 16 {
		t.Fatalf("expected encrypted envelope to be larger than plaintext, got %d bytes", len(upload.body))
	}
	header, err := envelope.ParseHeader(upload.body)
	if err != nil {
		t.Fatalf("ParseHeader returned error: %v", err)
	}
	if header.KeyID != key.KeyID {
		t.Fatalf("header key id = %q, want %q", header.KeyID, key.KeyID)
	}
	plaintext, err := envelope.DecryptChunk(key, chunkContext(upload.incidentID, upload.streamID, upload.mediaType, upload.chunkIndex), upload.body)
	if err != nil {
		t.Fatalf("DecryptChunk returned error: %v", err)
	}
	if len(plaintext) != 16 {
		t.Fatalf("decrypted plaintext length = %d, want 16", len(plaintext))
	}
	if upload.sha256Hex != sha256Hex(upload.body) {
		t.Fatalf("sha256Hex = %q, want ciphertext hash %q", upload.sha256Hex, sha256Hex(upload.body))
	}
}

func TestVerifyStreamBundleDecryption(t *testing.T) {
	key, err := envelope.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	incidentID := "inc_bundle"
	streamID := "str_bundle"
	mediaType := "audio"
	first := mustEncryptTestChunk(t, key, incidentID, streamID, mediaType, 1, []byte("first plaintext"))
	second := mustEncryptTestChunk(t, key, incidentID, streamID, mediaType, 2, []byte("second plaintext"))
	bundleBytes := makeTestBundle(t, streamBundleManifest{
		IncidentID: incidentID,
		StreamID:   streamID,
		MediaType:  mediaType,
		ChunkCount: 2,
		Chunks: []bundleChunkManifest{
			{ChunkIndex: 1, MediaType: mediaType, SHA256Hex: sha256Hex(first)},
			{ChunkIndex: 2, MediaType: mediaType, SHA256Hex: sha256Hex(second)},
		},
	}, map[string][]byte{
		"chunks/audio_000001.enc": first,
		"chunks/audio_000002.enc": second,
	})

	verified, err := verifyStreamBundleDecryption(bundleBytes, key, incidentID, streamID, mediaType)
	if err != nil {
		t.Fatalf("verifyStreamBundleDecryption returned error: %v", err)
	}
	if verified != 2 {
		t.Fatalf("verified = %d, want 2", verified)
	}
}

func TestVerifyStreamBundleDecryptionRejectsWrongMetadata(t *testing.T) {
	key, err := envelope.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	incidentID := "inc_bundle"
	streamID := "str_bundle"
	mediaType := "audio"
	chunk := mustEncryptTestChunk(t, key, incidentID, streamID, mediaType, 1, []byte("plaintext"))
	bundleBytes := makeTestBundle(t, streamBundleManifest{
		IncidentID: incidentID,
		StreamID:   streamID,
		MediaType:  mediaType,
		ChunkCount: 1,
		Chunks: []bundleChunkManifest{
			{ChunkIndex: 1, MediaType: mediaType, SHA256Hex: sha256Hex(chunk)},
		},
	}, map[string][]byte{
		"chunks/audio_000001.enc": chunk,
	})

	if _, err := verifyStreamBundleDecryption(bundleBytes, key, "inc_changed", streamID, mediaType); err == nil {
		t.Fatal("verifyStreamBundleDecryption succeeded with wrong incident id")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func testResponse(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func mustEncryptTestChunk(t *testing.T, key envelope.Key, incidentID, streamID, mediaType string, chunkIndex int, plaintext []byte) []byte {
	t.Helper()

	body, err := envelope.EncryptChunk(key, chunkContext(incidentID, streamID, mediaType, chunkIndex), plaintext)
	if err != nil {
		t.Fatalf("EncryptChunk returned error: %v", err)
	}
	return body
}

func makeTestBundle(t *testing.T, manifest streamBundleManifest, entries map[string][]byte) []byte {
	t.Helper()

	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	manifestWriter, err := writer.Create("manifest.json")
	if err != nil {
		t.Fatalf("create manifest entry: %v", err)
	}
	if err := json.NewEncoder(manifestWriter).Encode(manifest); err != nil {
		t.Fatalf("write manifest entry: %v", err)
	}
	for name, entryBody := range entries {
		entryWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entryWriter.Write(entryBody); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close test zip: %v", err)
	}
	return body.Bytes()
}
