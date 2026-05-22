package httpapi

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"safety-recorder/server/internal/incidents"
)

type streamBundleData struct {
	Stream   incidents.MediaStream
	Chunks   []incidents.Chunk
	Manifest streamBundleManifest
}

type streamBundleManifest struct {
	IncidentID  string                `json:"incident_id"`
	StreamID    string                `json:"stream_id"`
	MediaType   string                `json:"media_type"`
	Label       string                `json:"label,omitempty"`
	Status      string                `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	ChunkCount  int                   `json:"chunk_count"`
	TotalBytes  int64                 `json:"total_bytes"`
	Chunks      []bundleChunkManifest `json:"chunks"`
}

type bundleChunkManifest struct {
	ChunkIndex       int       `json:"chunk_index"`
	MediaType        string    `json:"media_type"`
	ByteSize         int64     `json:"byte_size"`
	SHA256Hex        string    `json:"sha256_hex"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	OriginalFilename string    `json:"original_filename,omitempty"`
}

type incidentBundleManifest struct {
	Incident      emergencyIncidentSummary `json:"incident"`
	LatestCheckin *emergencyCheckinSummary `json:"latest_checkin,omitempty"`
	Streams       []streamBundleManifest   `json:"streams"`
	StreamCount   int                      `json:"stream_count"`
	TotalBytes    int64                    `json:"total_bytes"`
	GeneratedAt   time.Time                `json:"generated_at"`
}

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
	if !a.recordEmergencyTokenUse(w, r, token.ID) {
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
	if !a.recordEmergencyTokenUse(w, r, token.ID) {
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
		if errors.Is(err, incidents.ErrInvalidState) || errors.Is(err, incidents.ErrNotFound) {
			continue
		}
		if err != nil {
			a.internalError(w, "build stream bundle", err)
			return nil, false
		}
		bundles = append(bundles, bundle)
	}
	return bundles, true
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

func writeStreamBundle(w io.Writer, openChunk func(string) (readSeekCloser, error), bundle streamBundleData, prefix string) error {
	zipWriter := zip.NewWriter(w)
	if err := writeStreamBundleToZip(zipWriter, openChunk, bundle, prefix); err != nil {
		_ = zipWriter.Close()
		return err
	}
	return zipWriter.Close()
}

func writeStreamBundleToZip(zipWriter *zip.Writer, openChunk func(string) (readSeekCloser, error), bundle streamBundleData, prefix string) error {
	if err := writeJSONZipEntry(zipWriter, prefix+"manifest.json", bundle.Manifest, bundle.Stream.UpdatedAt); err != nil {
		return err
	}
	for _, chunk := range bundle.Chunks {
		entryName := fmt.Sprintf("%schunks/%s_%06d.enc", prefix, safeZipSegment(chunk.MediaType), chunk.ChunkIndex)
		if err := writeChunkZipEntry(zipWriter, openChunk, entryName, chunk); err != nil {
			return err
		}
	}
	return nil
}

func validStreamBundleChunks(stream incidents.MediaStream, chunks []incidents.Chunk) bool {
	if stream.ExpectedChunkCount != nil && len(chunks) != *stream.ExpectedChunkCount {
		return false
	}
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i+1 || chunk.MediaType != stream.MediaType {
			return false
		}
	}
	return true
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

func (a *API) openBundleChunk(relPath string) (readSeekCloser, error) {
	return a.store.Open(relPath)
}

func writeChunkZipEntry(zipWriter *zip.Writer, openChunk func(string) (readSeekCloser, error), entryName string, chunk incidents.Chunk) error {
	header := &zip.FileHeader{
		Name:     entryName,
		Method:   zip.Deflate,
		Modified: chunk.CreatedAt,
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create chunk zip entry: %w", err)
	}

	file, err := openChunk(chunk.StoredPath)
	if err != nil {
		return fmt.Errorf("open chunk: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("copy chunk: %w", err)
	}
	return nil
}

func writeJSONZipEntry(zipWriter *zip.Writer, name string, value any, modified time.Time) error {
	header := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: modified,
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create json zip entry: %w", err)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json zip entry: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write json zip entry: %w", err)
	}
	return nil
}

func makeStreamBundleManifest(stream incidents.MediaStream, chunks []incidents.Chunk) streamBundleManifest {
	manifestChunks := make([]bundleChunkManifest, 0, len(chunks))
	var totalBytes int64
	for _, chunk := range chunks {
		totalBytes += chunk.ByteSize
		manifestChunks = append(manifestChunks, bundleChunkManifest{
			ChunkIndex:       chunk.ChunkIndex,
			MediaType:        chunk.MediaType,
			ByteSize:         chunk.ByteSize,
			SHA256Hex:        chunk.SHA256Hex,
			StartedAt:        chunk.StartedAt,
			EndedAt:          chunk.EndedAt,
			OriginalFilename: chunk.OriginalFilename,
		})
	}
	return streamBundleManifest{
		IncidentID:  stream.IncidentID,
		StreamID:    stream.ID,
		MediaType:   stream.MediaType,
		Label:       stream.Label,
		Status:      stream.Status,
		CreatedAt:   stream.CreatedAt,
		CompletedAt: stream.CompletedAt,
		ChunkCount:  len(chunks),
		TotalBytes:  totalBytes,
		Chunks:      manifestChunks,
	}
}

func makeIncidentBundleManifest(detail incidents.IncidentDetail, bundles []streamBundleData) incidentBundleManifest {
	var latestCheckin *emergencyCheckinSummary
	if len(detail.Checkins) > 0 {
		checkin := detail.Checkins[len(detail.Checkins)-1]
		latestCheckin = &emergencyCheckinSummary{
			CreatedAt:            checkin.CreatedAt,
			DeviceBatteryPercent: checkin.DeviceBatteryPercent,
			DeviceNetwork:        checkin.DeviceNetwork,
			Latitude:             checkin.Latitude,
			Longitude:            checkin.Longitude,
			AccuracyMeters:       checkin.AccuracyMeters,
		}
	}

	manifests := make([]streamBundleManifest, 0, len(bundles))
	var totalBytes int64
	for _, bundle := range bundles {
		manifests = append(manifests, bundle.Manifest)
		totalBytes += bundle.Manifest.TotalBytes
	}
	return incidentBundleManifest{
		Incident: emergencyIncidentSummary{
			ID:          detail.Incident.ID,
			Status:      detail.Incident.Status,
			ClientLabel: detail.Incident.ClientLabel,
			CreatedAt:   detail.Incident.CreatedAt,
			UpdatedAt:   detail.Incident.UpdatedAt,
		},
		LatestCheckin: latestCheckin,
		Streams:       manifests,
		StreamCount:   len(manifests),
		TotalBytes:    totalBytes,
		GeneratedAt:   time.Now().UTC(),
	}
}

func setBundleHeaders(w http.ResponseWriter, filename string) {
	setPublicBrowserSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	setNoStore(w)
}

func safeDownloadFilename(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('_')
	}
	filename := builder.String()
	if filename == "" || filename == "." || filename == ".." {
		return "evidence.zip"
	}
	return filename
}

func safeZipSegment(value string) string {
	return strings.Trim(safeDownloadFilename(value), ".")
}
