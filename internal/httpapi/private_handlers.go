package httpapi

import (
	"errors"
	"net/http"

	"github.com/open-proofline/server/internal/incidents"
)

func (a *API) createIncident(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required", "authentication is required")
		return
	}

	var request struct {
		ClientLabel string `json:"client_label"`
		Notes       string `json:"notes"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	incident, err := a.repo.CreateIncidentForAccount(r.Context(), principal.Account.ID, request.ClientLabel, request.Notes)
	if err != nil {
		a.internalError(w, "create incident", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"incident_id": incident.ID,
		"status":      incident.Status,
	})
}

func (a *API) getIncident(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authorizeIncident(w, r, r.PathValue("incident_id"), actionReadIncident, dataClassIncidentMetadata); !ok {
		return
	}
	detail, err := a.repo.GetIncidentDetail(r.Context(), r.PathValue("incident_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (a *API) createCheckin(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if !a.ensureIncidentExists(w, r, incidentID) {
		return
	}

	var request struct {
		DeviceBatteryPercent *int     `json:"device_battery_percent"`
		DeviceNetwork        *string  `json:"device_network"`
		Latitude             *float64 `json:"latitude"`
		Longitude            *float64 `json:"longitude"`
		AccuracyMeters       *float64 `json:"accuracy_meters"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	checkin, err := a.repo.CreateCheckin(r.Context(), incidentID, incidents.CreateCheckinParams{
		DeviceBatteryPercent: request.DeviceBatteryPercent,
		DeviceNetwork:        request.DeviceNetwork,
		Latitude:             request.Latitude,
		Longitude:            request.Longitude,
		AccuracyMeters:       request.AccuracyMeters,
	})
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "create checkin", err)
		return
	}

	writeJSON(w, http.StatusCreated, checkin)
}

func (a *API) closeIncident(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authorizeIncident(w, r, r.PathValue("incident_id"), actionWriteIncident, dataClassIncidentMetadata); !ok {
		return
	}
	incident, err := a.repo.CloseIncident(r.Context(), r.PathValue("incident_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "close incident", err)
		return
	}
	writeJSON(w, http.StatusOK, incident)
}
