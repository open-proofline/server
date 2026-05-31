package httpapi

import (
	"context"
	"net/http"
	"time"
)

const readinessCheckTimeout = 2 * time.Second

type healthResponse struct {
	Status string                `json:"status"`
	Checks []readinessCheckState `json:"checks,omitempty"`
}

type readinessCheckState struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Status  string `json:"status"`
}

func (a *API) healthLive(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (a *API) healthReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithReadinessTimeout(r)
	defer cancel()

	response := healthResponse{
		Status: "ok",
		Checks: make([]readinessCheckState, 0, len(a.readinessChecks)),
	}
	status := http.StatusOK
	for _, check := range a.readinessChecks {
		checkStatus := "ok"
		if check.Check == nil || check.Check(ctx) != nil {
			checkStatus = "unavailable"
			response.Status = "unavailable"
			status = http.StatusServiceUnavailable
		}
		response.Checks = append(response.Checks, readinessCheckState{
			Name:    check.Name,
			Backend: check.Backend,
			Status:  checkStatus,
		})
	}

	writeJSON(w, status, response)
}

func contextWithReadinessTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), readinessCheckTimeout)
}
