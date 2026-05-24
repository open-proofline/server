package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"

	"safety-recorder/server/internal/incidents"
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

func TestEmergencyIncidentBundleFailsClosedWhenCompletedStreamChunksAreNonContiguous(t *testing.T) {
	app := newTestApp(t)
	incidentID, stream := createIncidentStreamWithChunks(t, app, 1)
	completeMediaStream(t, app, incidentID, stream.ID, 1)
	updateStoredStreamChunkIndex(t, app, incidentID, stream.ID, 1, 2)
	token := createEmergencyToken(t, app, incidentID, "trusted contact", nil)

	response, body := getPublic(t, app, "/e/"+token.Token+"/incident/download")
	defer response.Body.Close()

	assertIncidentBundleInconsistent(t, response, body)
	assertEmergencyPrivacyHeaders(t, response)
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
