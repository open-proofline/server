package httpapi

import "net/http"

func (a *API) ensureIncidentExists(w http.ResponseWriter, r *http.Request, incidentID string) bool {
	_, ok := a.authorizeIncident(w, r, incidentID, actionReadIncident, dataClassIncidentMetadata)
	return ok
}
