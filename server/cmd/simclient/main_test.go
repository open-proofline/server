package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
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
