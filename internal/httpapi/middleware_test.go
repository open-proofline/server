package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddlewareLogDoesNotExposePanicValue(t *testing.T) {
	var logs bytes.Buffer
	api := &API{logger: slog.New(slog.NewTextHandler(&logs, nil))}
	handler := api.recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("raw-token-like-value /tmp/proofline/private/data")
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("panic response status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	for _, disallowed := range []string{"raw-token-like-value", "/tmp/proofline/private/data"} {
		if bytes.Contains(logs.Bytes(), []byte(disallowed)) {
			t.Fatalf("panic log exposed %q: %s", disallowed, logs.String())
		}
	}
	if !bytes.Contains(logs.Bytes(), []byte("panic_type=string")) {
		t.Fatalf("panic log omitted safe panic type: %s", logs.String())
	}
}
