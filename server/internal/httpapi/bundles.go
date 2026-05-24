package httpapi

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"safety-recorder/server/internal/incidents"
)

func (a *API) downloadPrivateStreamBundle(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	bundle, ok := a.loadCompletedStreamBundle(w, r, incidentID, r.PathValue("stream_id"))
	if !ok {
		return
	}
	a.serveStreamBundle(w, bundle)
}

func (a *API) downloadEmergencyStreamBundle(w http.ResponseWriter, r *http.Request) {
	token, ok := a.loadEmergencyToken(w, r)
	if !ok {
		return
	}
	bundle, ok := a.loadCompletedStreamBundle(w, r, token.IncidentID, r.PathValue("stream_id"))
	if !ok {
		return
	}
	a.serveStreamBundle(w, bundle)
}

func (a *API) downloadPrivateIncidentBundle(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	detail, err := a.repo.GetIncidentDetail(r.Context(), incidentID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident detail", err)
		return
	}
	bundles, ok := a.loadCompletedIncidentBundles(w, r, detail.Incident.ID)
	if !ok {
		return
	}
	a.serveIncidentBundle(w, detail, bundles)
}

func (a *API) downloadEmergencyIncidentBundle(w http.ResponseWriter, r *http.Request) {
	token, ok := a.loadEmergencyToken(w, r)
	if !ok {
		return
	}
	detail, err := a.repo.GetIncidentDetail(r.Context(), token.IncidentID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "emergency_token_invalid", "emergency token is invalid, expired, or revoked")
		return
	}
	if err != nil {
		a.internalError(w, "get incident detail", err)
		return
	}
	bundles, ok := a.loadCompletedIncidentBundles(w, r, token.IncidentID)
	if !ok {
		return
	}
	a.serveIncidentBundle(w, detail, bundles)
}

func (a *API) loadCompletedStreamBundle(w http.ResponseWriter, r *http.Request, incidentID, streamID string) (streamBundleData, bool) {
	bundle, err := a.buildCompletedStreamBundle(r.Context(), incidentID, streamID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "stream_not_found", "media stream was not found")
		return streamBundleData{}, false
	}
	if errors.Is(err, incidents.ErrInvalidState) {
		writeError(w, http.StatusConflict, "stream_not_complete", "media stream is not complete")
		return streamBundleData{}, false
	}
	if err != nil {
		a.internalError(w, "build stream bundle", err)
		return streamBundleData{}, false
	}
	return bundle, true
}

func (a *API) loadCompletedIncidentBundles(w http.ResponseWriter, r *http.Request, incidentID string) ([]streamBundleData, bool) {
	streams, err := a.repo.ListCompletedMediaStreams(r.Context(), incidentID)
	if err != nil {
		a.internalError(w, "list completed streams", err)
		return nil, false
	}

	bundles := make([]streamBundleData, 0, len(streams))
	for _, stream := range streams {
		bundle, err := a.buildCompletedStreamBundle(r.Context(), incidentID, stream.ID)
		if isIncidentBundleInconsistency(err) {
			writeError(w, http.StatusConflict, "incident_bundle_inconsistent", "completed media stream could not be included in incident bundle")
			return nil, false
		}
		if err != nil {
			a.internalError(w, "build stream bundle", err)
			return nil, false
		}
		bundles = append(bundles, bundle)
	}
	return bundles, true
}

func isIncidentBundleInconsistency(err error) bool {
	return errors.Is(err, incidents.ErrInvalidState) ||
		errors.Is(err, incidents.ErrNotFound) ||
		errors.Is(err, os.ErrNotExist)
}

func (a *API) buildCompletedStreamBundle(ctx context.Context, incidentID, streamID string) (streamBundleData, error) {
	stream, err := a.repo.GetMediaStream(ctx, incidentID, streamID)
	if err != nil {
		return streamBundleData{}, err
	}
	if stream.Status != incidents.StreamStatusComplete {
		return streamBundleData{}, incidents.ErrInvalidState
	}

	chunks, err := a.repo.ListStreamChunks(ctx, incidentID, streamID)
	if err != nil {
		return streamBundleData{}, err
	}
	if len(chunks) == 0 {
		return streamBundleData{}, incidents.ErrInvalidState
	}
	if !validStreamBundleChunks(stream, chunks) {
		return streamBundleData{}, incidents.ErrInvalidState
	}
	for _, chunk := range chunks {
		file, err := a.store.Open(chunk.StoredPath)
		if err != nil {
			return streamBundleData{}, fmt.Errorf("open chunk for bundle: %w", err)
		}
		if err := file.Close(); err != nil {
			return streamBundleData{}, fmt.Errorf("close chunk for bundle: %w", err)
		}
	}

	manifest := makeStreamBundleManifest(stream, chunks)
	return streamBundleData{
		Stream:   stream,
		Chunks:   chunks,
		Manifest: manifest,
	}, nil
}

func (a *API) serveStreamBundle(w http.ResponseWriter, bundle streamBundleData) {
	filename := safeDownloadFilename(fmt.Sprintf("incident_%s_%s_%s.zip", bundle.Stream.IncidentID, bundle.Stream.MediaType, bundle.Stream.ID))
	setBundleHeaders(w, filename)
	if err := writeStreamBundle(w, a.openBundleChunk, bundle, ""); err != nil {
		a.logger.Error("write stream bundle", "err", err)
	}
}

func (a *API) serveIncidentBundle(w http.ResponseWriter, detail incidents.IncidentDetail, bundles []streamBundleData) {
	filename := safeDownloadFilename(fmt.Sprintf("incident_%s_evidence.zip", detail.Incident.ID))
	setBundleHeaders(w, filename)

	manifest := makeIncidentBundleManifest(detail, bundles)
	zipWriter := zip.NewWriter(w)
	if err := writeJSONZipEntry(zipWriter, "manifest.json", manifest, manifest.GeneratedAt); err != nil {
		a.logger.Error("write incident manifest", "err", err)
		_ = zipWriter.Close()
		return
	}
	for _, bundle := range bundles {
		prefix := "streams/" + safeZipSegment(bundle.Stream.ID) + "/"
		if err := writeStreamBundleToZip(zipWriter, a.openBundleChunk, bundle, prefix); err != nil {
			a.logger.Error("write incident stream bundle", "err", err)
			_ = zipWriter.Close()
			return
		}
	}
	if err := zipWriter.Close(); err != nil {
		a.logger.Error("close incident bundle", "err", err)
	}
}
