package httpapi_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"

	"github.com/open-proofline/server/internal/incidents"
)

func TestPrivateIncidentBundleFailsClosedWhenCompletedStreamChunkFileMissing(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	removeStoredStreamChunkFile(t, app, incidentID, stream.ID, incidents.MediaTypeAudio, 1)

	response, body := get(t, app, "/v1/incidents/"+incidentID+"/download")
	defer response.Body.Close()

	assertIncidentBundleInconsistent(t, response, body)
	assertIncidentBundleErrorDoesNotExposeStorageDetails(t, body, app.dataDir, stream.ID, "audio_000001.enc")
}

func TestPrivateStreamBundleStorageFailureLogDoesNotExposeStoragePath(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	app := newTestAppWithMaxUploadBytesAndLogger(t, 1024*1024, logger)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	removeStoredStreamChunkFile(t, app, incidentID, stream.ID, incidents.MediaTypeAudio, 1)

	response, body := get(t, app, "/v1/incidents/"+incidentID+"/streams/"+stream.ID+"/download")
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected stream bundle storage failure status 500, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "internal_error")
	assertIncidentBundleErrorDoesNotExposeStorageDetails(t, body, app.dataDir, stream.ID, "audio_000001.enc")
	assertLogDoesNotExposeStorageDetails(t, logs.Bytes(), app.dataDir, "incidents/"+incidentID+"/streams/"+stream.ID+"/audio_000001.enc", "audio_000001.enc")
	if !bytes.Contains(logs.Bytes(), []byte("error_category=not_found")) {
		t.Fatalf("expected safe not_found error category in logs: %s", logs.String())
	}
}

func TestPrivateIncidentBundleFailsClosedWhenCompletedStreamChunksAreNonContiguous(t *testing.T) {
	app := newTestApp(t)
	incidentID, brokenStream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, brokenStream.ID, 1)
	validStream := createMediaStream(t, app, incidentID, incidents.MediaTypeVideo, "video recording")
	payload := []byte("encrypted video data")
	response, body := uploadChunkWithStream(t, app, incidentID, validStream.ID, 1, incidents.MediaTypeVideo, payload, sha256Hex(payload))
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected video stream chunk upload status 201, got %d: %s", response.StatusCode, body)
	}
	completeMediaStream(t, app, incidentID, validStream.ID, 1)
	updateStoredStreamChunkIndex(t, app, incidentID, brokenStream.ID, 1, 2)

	response, body = get(t, app, "/v1/incidents/"+incidentID+"/download")
	defer response.Body.Close()

	assertIncidentBundleInconsistent(t, response, body)
	assertIncidentBundleErrorDoesNotExposeStorageDetails(t, body, app.dataDir, brokenStream.ID, "audio_000001.enc")
	if bytes.Contains(body, []byte(validStream.ID)) {
		t.Fatalf("incident bundle inconsistency error exposed valid stream ID: %s", body)
	}
}

func TestIncidentViewerIncidentBundleFailsClosedWhenCompletedStreamChunksAreNonContiguous(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	updateStoredStreamChunkIndex(t, app, incidentID, stream.ID, 1, 2)
	token := createIncidentToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/i/"+token.Token+"/incident/download")
	defer response.Body.Close()

	assertIncidentBundleInconsistent(t, response, body)
	assertIncidentViewerPrivacyHeaders(t, response)
	assertIncidentBundleErrorDoesNotExposeStorageDetails(t, body, app.dataDir, stream.ID, "audio_000001.enc")
	if bytes.Contains(body, []byte(token.Token)) {
		t.Fatalf("incident bundle inconsistency error exposed raw token: %s", body)
	}
}

func assertIncidentBundleInconsistent(t *testing.T, response *http.Response, body []byte) {
	t.Helper()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected incident bundle inconsistency status 409, got %d: %s", response.StatusCode, body)
	}
	assertErrorCode(t, body, "incident_bundle_inconsistent")
}

func assertIncidentBundleErrorDoesNotExposeStorageDetails(t *testing.T, body []byte, dataDir, streamID, chunkFilename string) {
	t.Helper()

	for _, disallowed := range []string{dataDir, streamID, chunkFilename} {
		if bytes.Contains(body, []byte(disallowed)) {
			t.Fatalf("incident bundle inconsistency error exposed %q: %s", disallowed, body)
		}
	}
}

func assertLogDoesNotExposeStorageDetails(t *testing.T, logs []byte, dataDir, storedPath, chunkFilename string) {
	t.Helper()

	for _, disallowed := range []string{dataDir, storedPath, chunkFilename} {
		if bytes.Contains(logs, []byte(disallowed)) {
			t.Fatalf("internal logs exposed %q: %s", disallowed, logs)
		}
	}
}
