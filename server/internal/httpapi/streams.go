package httpapi

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"safety-recorder/server/internal/incidents"
)

func (a *API) createMediaStream(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if !a.ensureIncidentExists(w, r, incidentID) {
		return
	}

	var request struct {
		MediaType string `json:"media_type"`
		Label     string `json:"label"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if !incidents.ValidMediaType(request.MediaType) {
		writeError(w, http.StatusBadRequest, "invalid_media_type", "media_type must be audio, video, location, or metadata")
		return
	}

	stream, err := a.repo.CreateMediaStream(r.Context(), incidentID, request.MediaType, strings.TrimSpace(request.Label))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "create media stream", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]incidents.MediaStream{"stream": stream})
}

func (a *API) listMediaStreams(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if !a.ensureIncidentExists(w, r, incidentID) {
		return
	}

	streams, err := a.repo.ListMediaStreams(r.Context(), incidentID)
	if err != nil {
		a.internalError(w, "list media streams", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]incidents.MediaStream{"streams": streams})
}

func (a *API) getMediaStream(w http.ResponseWriter, r *http.Request) {
	stream, ok := a.loadMediaStream(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]incidents.MediaStream{"stream": stream})
}

func (a *API) completeMediaStream(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	stream, ok := a.loadMediaStream(w, r)
	if !ok {
		return
	}
	if stream.Status == incidents.StreamStatusComplete {
		writeError(w, http.StatusConflict, "stream_already_complete", "media stream is already complete")
		return
	}
	if stream.Status == incidents.StreamStatusFailed {
		writeError(w, http.StatusConflict, "stream_failed", "failed media stream cannot be completed")
		return
	}

	var request struct {
		ExpectedChunkCount *int `json:"expected_chunk_count"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.ExpectedChunkCount == nil || *request.ExpectedChunkCount <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_expected_chunk_count", "expected_chunk_count must be a positive integer")
		return
	}
	chunks, err := a.repo.ListStreamChunks(r.Context(), incidentID, stream.ID)
	if err != nil {
		a.internalError(w, "list stream chunks", err)
		return
	}
	if !a.validateCompleteStreamChunks(w, chunks, *request.ExpectedChunkCount) {
		return
	}

	updated, err := a.repo.CompleteMediaStream(r.Context(), incidentID, stream.ID, *request.ExpectedChunkCount)
	if errors.Is(err, incidents.ErrInvalidState) {
		writeError(w, http.StatusConflict, "stream_not_open", "media stream is not open")
		return
	}
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
		return
	}
	if err != nil {
		a.internalError(w, "complete media stream", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]incidents.MediaStream{"stream": updated})
}

func (a *API) failMediaStream(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	stream, ok := a.loadMediaStream(w, r)
	if !ok {
		return
	}
	if stream.Status != incidents.StreamStatusOpen {
		writeError(w, http.StatusConflict, "stream_not_open", "media stream is not open")
		return
	}

	var request struct {
		FailureReason string `json:"failure_reason"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	updated, err := a.repo.FailMediaStream(r.Context(), incidentID, stream.ID, strings.TrimSpace(request.FailureReason))
	if errors.Is(err, incidents.ErrInvalidState) {
		writeError(w, http.StatusConflict, "stream_not_open", "media stream is not open")
		return
	}
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
		return
	}
	if err != nil {
		a.internalError(w, "fail media stream", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]incidents.MediaStream{"stream": updated})
}

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

func (a *API) loadMediaStream(w http.ResponseWriter, r *http.Request) (incidents.MediaStream, bool) {
	incidentID := r.PathValue("incident_id")
	if !a.ensureIncidentExists(w, r, incidentID) {
		return incidents.MediaStream{}, false
	}
	stream, err := a.repo.GetMediaStream(r.Context(), incidentID, r.PathValue("stream_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
		return incidents.MediaStream{}, false
	}
	if err != nil {
		a.internalError(w, "get media stream", err)
		return incidents.MediaStream{}, false
	}
	return stream, true
}

func (a *API) validateCompleteStreamChunks(w http.ResponseWriter, chunks []incidents.Chunk, expectedChunkCount int) bool {
	if len(chunks) != expectedChunkCount {
		writeError(w, http.StatusConflict, "stream_chunks_incomplete", "stream does not have the expected number of chunks")
		return false
	}
	for i, chunk := range chunks {
		expectedIndex := i + 1
		if chunk.ChunkIndex != expectedIndex {
			writeError(w, http.StatusConflict, "stream_chunks_not_contiguous", "stream chunks must be contiguous from 1 to expected_chunk_count")
			return false
		}
		file, err := a.store.Open(chunk.StoredPath)
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusConflict, "stream_chunk_file_missing", "stream chunk file is missing")
			return false
		}
		if err != nil {
			a.internalError(w, "open stream chunk", err)
			return false
		}
		if err := file.Close(); err != nil {
			a.internalError(w, "close stream chunk", err)
			return false
		}
	}
	return true
}
