package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"safety-recorder/server/internal/incidents"
)

func TestCreateIncident(t *testing.T) {
	app := newTestApp(t)

	incidentID := createIncident(t, app, `{"client_label":"phone","notes":"test"}`)

	if incidentID == "" {
		t.Fatal("expected incident id")
	}
}

func TestPrivateAPIJSONSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	response, body := post(t, app, "/v1/incidents", "application/json", bytes.NewBufferString(`{}`))
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create incident status 201, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
}

func TestPrivateAPIErrorSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	response, body := get(t, app, "/v1/incidents/inc_missing")
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing incident status 404, got %d: %s", response.StatusCode, body)
	}
	assertPrivateJSONSecurityHeaders(t, response)
	assertErrorCode(t, body, "incident_not_found")
}

func TestGetIncidentReturnsEmptyArrays(t *testing.T) {
	app := newTestApp(t)
	incidentID := createIncident(t, app, `{}`)

	response, body := get(t, app, "/v1/incidents/"+incidentID)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected incident status 200, got %d: %s", response.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(`"chunks":[]`)) {
		t.Fatalf("expected chunks to be an empty array, got: %s", body)
	}
	if !bytes.Contains(body, []byte(`"checkins":[]`)) {
		t.Fatalf("expected checkins to be an empty array, got: %s", body)
	}
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
