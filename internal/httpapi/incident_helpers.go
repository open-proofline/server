package httpapi

import (
	"errors"
	"net/http"

	"github.com/open-proofline/server/internal/incidents"
)

func (a *API) ensureIncidentExists(w http.ResponseWriter, r *http.Request, incidentID string) bool {
	if _, err := a.repo.GetIncident(r.Context(), incidentID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return false
	} else if err != nil {
		a.internalError(w, "get incident", err)
		return false
	}
	return true
}
