package httpapi

import "net/http"

func (a *API) privateRoutes() http.Handler {
	mux := http.NewServeMux()
	a.registerPrivateHealthRoutes(mux)
	a.registerPrivateAuthRoutes(mux)
	a.registerPrivateAdminWebRoutes(mux)
	a.registerPrivateIncidentRoutes(mux)
	a.registerPrivateStreamRoutes(mux)
	a.registerPrivateIncidentTokenRoutes(mux)
	mux.HandleFunc("/", a.notFound)

	return a.loggingMiddleware(a.recoveryMiddleware(a.privateSecurityMiddleware(a.privateAuthMiddleware(mux))))
}

func (a *API) registerPrivateHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/health/live", a.healthLive)
	mux.HandleFunc("GET /v1/health/ready", a.healthReady)
}

func (a *API) registerPrivateAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/bootstrap/admin", a.bootstrapAdmin)
	mux.HandleFunc("POST /v1/auth/login", a.login)
	mux.HandleFunc("POST /v1/auth/logout", a.logout)
	mux.HandleFunc("GET /v1/account", a.getCurrentAccount)
	mux.HandleFunc("POST /v1/account/password", a.changeOwnPassword)
	mux.HandleFunc("GET /v1/admin/accounts", a.listAccounts)
	mux.HandleFunc("POST /v1/admin/accounts", a.createAccount)
	mux.HandleFunc("POST /v1/admin/accounts/{account_id}/password", a.resetAccountPassword)
	mux.HandleFunc("POST /v1/admin/accounts/{account_id}/sessions/revoke", a.revokeAccountSessions)
}

func (a *API) registerPrivateAdminWebRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin", a.adminWebPage)
	mux.HandleFunc("POST /admin/bootstrap", a.adminWebBootstrap)
	mux.HandleFunc("POST /admin/login", a.adminWebLogin)
	mux.HandleFunc("POST /admin/logout", a.adminWebLogout)
	mux.HandleFunc("POST /admin/password", a.adminWebChangeOwnPassword)
	mux.HandleFunc("POST /admin/accounts/{account_id}/password", a.adminWebResetAccountPassword)
	mux.Handle("GET /admin/static/", a.adminWebStaticHandler())
}

func (a *API) registerPrivateIncidentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/incidents", a.createIncident)
	mux.HandleFunc("GET /v1/incidents/{incident_id}", a.getIncident)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/chunks/reconcile", a.reconcileChunk)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/chunks", a.uploadChunk)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/chunks", a.listChunks)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}", a.getChunkBytes)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/download", a.downloadPrivateIncidentBundle)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/checkins", a.createCheckin)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/close", a.closeIncident)
}

func (a *API) registerPrivateStreamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/incidents/{incident_id}/streams", a.createMediaStream)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/streams", a.listMediaStreams)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/streams/{stream_id}", a.getMediaStream)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/streams/{stream_id}/complete", a.completeMediaStream)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/streams/{stream_id}/fail", a.failMediaStream)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/streams/{stream_id}/download", a.downloadPrivateStreamBundle)
}

func (a *API) registerPrivateIncidentTokenRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/incidents/{incident_id}/incident-tokens", a.createIncidentToken)
	mux.HandleFunc("POST /v1/incident-tokens/{token_id}/revoke", a.revokeIncidentToken)
}

func (a *API) publicRoutes() http.Handler {
	mux := http.NewServeMux()
	a.registerPublicIncidentViewerRoutes(mux)
	mux.HandleFunc("/", a.notFound)

	return a.loggingMiddleware(a.recoveryMiddleware(a.publicSecurityMiddleware(mux)))
}

func (a *API) registerPublicIncidentViewerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /i/{token}", a.incidentViewerPage)
	mux.HandleFunc("GET /i/{token}/data", a.incidentViewData)
	mux.HandleFunc("GET /i/{token}/streams/{stream_id}/download", a.downloadIncidentViewerStreamBundle)
	mux.HandleFunc("GET /i/{token}/incident/download", a.downloadIncidentViewerIncidentBundle)
	// Keep the pre-rename viewer path as a compatibility alias for already
	// shared token-bearing links. /i remains canonical for new links.
	mux.HandleFunc("GET /e/{token}", a.incidentViewerPage)
	mux.HandleFunc("GET /e/{token}/data", a.incidentViewData)
	mux.HandleFunc("GET /e/{token}/streams/{stream_id}/download", a.downloadIncidentViewerStreamBundle)
	mux.HandleFunc("GET /e/{token}/incident/download", a.downloadIncidentViewerIncidentBundle)
	// Static incident viewer assets are embedded and token-neutral; the token stays
	// in the request path handled above.
	mux.Handle("GET /static/", incidentViewerStaticHandler())
}

func (a *API) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "endpoint was not found")
}
