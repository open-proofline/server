package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/open-proofline/server/internal/incidents"
)

type incidentDeletionRequest struct {
	ReasonCode string `json:"reason_code"`
	AllowOpen  bool   `json:"allow_open"`
}

func (a *API) requestIncidentDeletion(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	incident, ok := a.authorizeIncident(w, r, incidentID, actionDeleteIncident, dataClassIncidentMetadata)
	if !ok {
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required", "authentication is required")
		return
	}
	request, ok := a.decodeIncidentDeletionRequest(w, r)
	if !ok {
		return
	}
	if incident.OwnerAccountID != principal.Account.ID {
		writeError(w, http.StatusForbidden, "forbidden", "account is not authorized for this incident")
		return
	}

	status, err := a.repo.RequestIncidentDeletion(r.Context(), incidents.IncidentDeletionRequest{
		IncidentID:     incident.ID,
		Source:         incidents.IncidentDeletionSourceAccountRequest,
		ReasonCode:     request.ReasonCode,
		ActorAccountID: principal.Account.ID,
		AllowOpen:      request.AllowOpen,
		RequireOwnerID: principal.Account.ID,
	})
	if !a.writeIncidentDeletionResult(w, err, status) {
		return
	}
}

func (a *API) requestAdminIncidentDeletion(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required", "authentication is required")
		return
	}
	request, ok := a.decodeIncidentDeletionRequest(w, r)
	if !ok {
		return
	}

	status, err := a.repo.RequestIncidentDeletion(r.Context(), incidents.IncidentDeletionRequest{
		IncidentID:     r.PathValue("incident_id"),
		Source:         incidents.IncidentDeletionSourceAdminRequest,
		ReasonCode:     request.ReasonCode,
		ActorAccountID: principal.Account.ID,
		AllowOpen:      request.AllowOpen,
	})
	if !a.writeIncidentDeletionResult(w, err, status) {
		return
	}
}

func (a *API) getIncidentDeletion(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if _, ok := a.authorizeIncident(w, r, incidentID, actionReadIncident, dataClassIncidentMetadata); !ok {
		return
	}
	status, err := a.repo.GetIncidentDeletionStatus(r.Context(), incidentID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_deletion_not_found", "incident deletion was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident deletion", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]incidents.IncidentDeletionStatus{"deletion": status})
}

func (a *API) getAdminIncidentDeletion(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	status, err := a.repo.GetIncidentDeletionStatus(r.Context(), r.PathValue("incident_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_deletion_not_found", "incident deletion was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get admin incident deletion", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]incidents.IncidentDeletionStatus{"deletion": status})
}

func (a *API) decodeIncidentDeletionRequest(w http.ResponseWriter, r *http.Request) (incidentDeletionRequest, bool) {
	var request incidentDeletionRequest
	if !decodeJSON(w, r, &request) {
		return incidentDeletionRequest{}, false
	}
	reasonCode, ok := normalizeDeletionReasonCode(w, request.ReasonCode)
	if !ok {
		return incidentDeletionRequest{}, false
	}
	request.ReasonCode = reasonCode
	return request, true
}

func (a *API) writeIncidentDeletionResult(w http.ResponseWriter, err error, status incidents.IncidentDeletionStatus) bool {
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return false
	}
	if errors.Is(err, incidents.ErrInvalidState) {
		writeError(w, http.StatusConflict, "incident_deletion_invalid_state", "incident cannot be deleted in its current state")
		return false
	}
	if err != nil {
		a.internalError(w, "request incident deletion", err)
		return false
	}
	writeJSON(w, http.StatusAccepted, map[string]incidents.IncidentDeletionStatus{"deletion": status})
	return true
}

func normalizeDeletionReasonCode(w http.ResponseWriter, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len(value) > 64 {
		writeError(w, http.StatusBadRequest, "invalid_reason_code", "reason_code must be a short non-sensitive code")
		return "", false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.' || r == ':' {
			continue
		}
		writeError(w, http.StatusBadRequest, "invalid_reason_code", "reason_code must be a short non-sensitive code")
		return "", false
	}
	return value, true
}
