package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/open-proofline/server/internal/httpapi"
)

func TestPrivateHealthRoutesAreUnauthenticatedAndNotPublic(t *testing.T) {
	app := newTestApp(t)

	response, body := getUnauthenticated(t, app, "/v1/health/live")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected live status 200, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
	assertHealthStatus(t, body, "ok")

	response, body = getUnauthenticated(t, app, "/v1/health/ready")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected ready status 200, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
	assertHealthStatus(t, body, "ok")

	response, body = getPublic(t, app, "/v1/health/ready")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected public ready status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "not_found")

	response, body = getPublic(t, app, "/v1/health/live")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected public live status 404, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "not_found")
}

func TestPrivateReadinessResponseIsCoarseAndRedacted(t *testing.T) {
	app := newTestAppWithOptions(t, httpapi.Options{
		ReadinessChecks: []httpapi.ReadinessCheck{
			{
				Name:    "metadata",
				Backend: "postgresql",
				Check: func(context.Context) error {
					return errors.New("dial postgres://secret@10.0.0.5/proofline failed")
				},
			},
			{
				Name:    "blob",
				Backend: "s3",
				Check: func(context.Context) error {
					return errors.New("bucket proofline-evidence object incidents/inc_secret/audio.enc unavailable")
				},
			},
			{
				Name:    "coordination",
				Backend: "valkey",
				Check: func(context.Context) error {
					return nil
				},
			},
		},
	})

	response, body := getUnauthenticated(t, app, "/v1/health/ready")
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected ready status 503, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
	for _, disallowed := range [][]byte{
		[]byte("secret"),
		[]byte("10.0.0.5"),
		[]byte("proofline-evidence"),
		[]byte("incidents/"),
		[]byte("audio.enc"),
	} {
		if bytes.Contains(body, disallowed) {
			t.Fatalf("readiness response exposed %q: %s", disallowed, body)
		}
	}

	var result struct {
		Status string `json:"status"`
		Checks []struct {
			Name    string `json:"name"`
			Backend string `json:"backend"`
			Status  string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if result.Status != "unavailable" {
		t.Fatalf("readiness status = %q, want unavailable", result.Status)
	}
	want := map[string]struct {
		backend string
		status  string
	}{
		"metadata":     {backend: "postgresql", status: "unavailable"},
		"blob":         {backend: "s3", status: "unavailable"},
		"coordination": {backend: "valkey", status: "ok"},
	}
	if len(result.Checks) != len(want) {
		t.Fatalf("readiness checks = %+v, want %d checks", result.Checks, len(want))
	}
	for _, check := range result.Checks {
		expected, ok := want[check.Name]
		if !ok {
			t.Fatalf("unexpected readiness check %+v", check)
		}
		if check.Backend != expected.backend || check.Status != expected.status {
			t.Fatalf("check %q = backend %q status %q, want backend %q status %q",
				check.Name,
				check.Backend,
				check.Status,
				expected.backend,
				expected.status,
			)
		}
	}
}

func TestPrivateReadinessDoesNotLogCheckErrors(t *testing.T) {
	var logs bytes.Buffer
	app := newTestAppWithOptions(t, httpapi.Options{
		Logger: slog.New(slog.NewTextHandler(&logs, nil)),
		ReadinessChecks: []httpapi.ReadinessCheck{
			{
				Name:    "metadata",
				Backend: "postgresql",
				Check: func(context.Context) error {
					return errors.New("postgres://secret@10.0.0.5/proofline")
				},
			},
		},
	})

	response, body := getUnauthenticated(t, app, "/v1/health/ready")
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected ready status 503, got %d: %s", response.StatusCode, body)
	}
	for _, disallowed := range [][]byte{
		[]byte("secret"),
		[]byte("10.0.0.5"),
		[]byte("proofline"),
	} {
		if bytes.Contains(logs.Bytes(), disallowed) {
			t.Fatalf("readiness logs exposed %q: %s", disallowed, logs.String())
		}
	}
}

func assertHealthStatus(t *testing.T, body []byte, want string) {
	t.Helper()
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if result.Status != want {
		t.Fatalf("health status = %q, want %q", result.Status, want)
	}
}
