package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/open-proofline/server/internal/incidents"
	"github.com/open-proofline/server/internal/storage"
)

func (a *API) uploadChunk(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	incident, err := a.repo.GetIncident(r.Context(), incidentID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident", err)
		return
	}
	if incident.Status == incidents.StatusClosed {
		writeError(w, http.StatusConflict, "incident_closed", "incident is closed")
		return
	}

	upload, ok := a.readChunkUpload(w, r)
	if !ok {
		return
	}
	defer upload.temp.Cleanup()

	if upload.temp.SHA256Hex != upload.sha256Hex {
		writeError(w, http.StatusBadRequest, "hash_mismatch", "computed SHA-256 did not match provided hash")
		return
	}
	if !a.validateChunkStream(w, r, incidentID, upload) {
		return
	}

	exists, err := a.repo.ChunkExists(r.Context(), incidentID, upload.streamID, upload.mediaType, upload.chunkIndex)
	if err != nil {
		a.internalError(w, "check duplicate chunk", err)
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "duplicate_chunk", "chunk_index already exists for this chunk identity")
		return
	}

	storedPath, err := a.store.CommitTemp(upload.temp, incidentID, upload.streamID, upload.mediaType, upload.chunkIndex)
	if errors.Is(err, storage.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "duplicate_chunk", "stored chunk already exists for this chunk identity")
		return
	}
	if err != nil {
		a.internalError(w, "commit upload", err)
		return
	}

	chunk, err := a.repo.CreateChunk(r.Context(), incidents.CreateChunkParams{
		IncidentID:       incidentID,
		StreamID:         upload.streamID,
		ChunkIndex:       upload.chunkIndex,
		MediaType:        upload.mediaType,
		StartedAt:        upload.startedAt,
		EndedAt:          upload.endedAt,
		OriginalFilename: upload.originalFilename,
		StoredPath:       storedPath,
		ByteSize:         upload.temp.ByteSize,
		SHA256Hex:        upload.sha256Hex,
	})
	if errors.Is(err, incidents.ErrDuplicate) {
		_ = a.store.Remove(storedPath)
		writeError(w, http.StatusConflict, "duplicate_chunk", "chunk_index already exists for this chunk identity")
		return
	}
	if errors.Is(err, incidents.ErrIncidentClosed) {
		_ = a.store.Remove(storedPath)
		writeError(w, http.StatusConflict, "incident_closed", "incident is closed")
		return
	}
	if errors.Is(err, incidents.ErrInvalidState) {
		_ = a.store.Remove(storedPath)
		writeError(w, http.StatusConflict, "stream_not_open", "media stream is not open")
		return
	}
	if errors.Is(err, incidents.ErrNotFound) {
		_ = a.store.Remove(storedPath)
		if upload.streamID != "" {
			writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
			return
		}
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		_ = a.store.Remove(storedPath)
		a.internalError(w, "insert chunk metadata", err)
		return
	}

	writeJSON(w, http.StatusCreated, chunk)
}

func (a *API) listChunks(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if !a.ensureIncidentExists(w, r, incidentID) {
		return
	}

	chunks, err := a.repo.ListChunks(r.Context(), incidentID)
	if err != nil {
		a.internalError(w, "list chunks", err)
		return
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (a *API) getChunkBytes(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	mediaType := r.PathValue("media_type")
	if !incidents.ValidMediaType(mediaType) {
		writeError(w, http.StatusBadRequest, "invalid_media_type", "media_type must be audio, video, location, or metadata")
		return
	}
	chunkIndex, err := parseChunkIndex(r.PathValue("chunk_index"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_chunk_index", "chunk_index must be a non-negative integer")
		return
	}

	chunk, err := a.repo.GetChunkByKey(r.Context(), incidentID, mediaType, chunkIndex)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "chunk_not_found", "chunk was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get chunk", err)
		return
	}

	file, err := a.store.Open(chunk.StoredPath)
	if errors.Is(err, os.ErrNotExist) {
		a.internalError(w, "open chunk bytes", fmt.Errorf("metadata exists but file is missing: %w", err))
		return
	}
	if err != nil {
		a.internalError(w, "open chunk bytes", err)
		return
	}
	defer file.Close()

	setNoSniff(w)
	setNoStore(w)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(chunk.ByteSize, 10))
	if seeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, path.Base(chunk.StoredPath), chunk.CreatedAt, seeker)
		return
	}
	if _, err := io.Copy(w, file); err != nil {
		a.logInternalError("write chunk bytes", err)
	}
}

func (a *API) validateChunkStream(w http.ResponseWriter, r *http.Request, incidentID string, upload chunkUpload) bool {
	if upload.streamID == "" {
		return true
	}

	stream, err := a.repo.GetMediaStream(r.Context(), incidentID, upload.streamID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
		return false
	}
	if err != nil {
		a.internalError(w, "get media stream", err)
		return false
	}
	if stream.Status != incidents.StreamStatusOpen {
		writeError(w, http.StatusConflict, "stream_not_open", "media stream is not open")
		return false
	}
	if stream.MediaType != upload.mediaType {
		writeError(w, http.StatusBadRequest, "stream_media_type_mismatch", "stream media_type does not match chunk media_type")
		return false
	}
	return true
}
