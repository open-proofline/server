package main

import "testing"

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

func TestParseConfigRejectsDownloadWithoutCompleteStream(t *testing.T) {
	if _, err := parseConfig([]string{"--download-bundle", "--complete-stream=false"}); err == nil {
		t.Fatal("expected --download-bundle without --complete-stream to fail")
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
