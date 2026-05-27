package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"safety-recorder/server/internal/incidents"
)

type emergencyIncidentSummary struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	ClientLabel string    `json:"client_label,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type emergencyCheckinSummary struct {
	CreatedAt            time.Time `json:"created_at"`
	DeviceBatteryPercent *int      `json:"device_battery_percent,omitempty"`
	DeviceNetwork        *string   `json:"device_network,omitempty"`
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	AccuracyMeters       *float64  `json:"accuracy_meters,omitempty"`
}

type emergencyChunkSummary struct {
	ChunkIndex       int       `json:"chunk_index"`
	MediaType        string    `json:"media_type"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	ByteSize         int64     `json:"byte_size"`
	SHA256Hex        string    `json:"sha256_hex"`
	CreatedAt        time.Time `json:"created_at"`
}

type emergencyMediaSummary struct {
	MediaType   string                 `json:"media_type"`
	ChunkCount  int                    `json:"chunk_count"`
	LatestChunk *emergencyChunkSummary `json:"latest_chunk,omitempty"`
}

type emergencyStreamSummary struct {
	ID                 string     `json:"id"`
	MediaType          string     `json:"media_type"`
	Label              string     `json:"label,omitempty"`
	Status             string     `json:"status"`
	ExpectedChunkCount *int       `json:"expected_chunk_count,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	FailedAt           *time.Time `json:"failed_at,omitempty"`
	FailureReason      string     `json:"failure_reason,omitempty"`
	ChunkCount         int        `json:"chunk_count"`
	TotalBytes         int64      `json:"total_bytes"`
}

type emergencyViewData struct {
	Incident               emergencyIncidentSummary          `json:"incident"`
	LatestCheckin          *emergencyCheckinSummary          `json:"latest_checkin,omitempty"`
	ChunkCountByMediaType  map[string]int                    `json:"chunk_count_by_media_type"`
	LatestChunkByMediaType map[string]*emergencyChunkSummary `json:"latest_chunk_by_media_type"`
	Media                  []emergencyMediaSummary           `json:"media"`
	Streams                []emergencyStreamSummary          `json:"streams"`
	CompletedStreams       []emergencyStreamSummary          `json:"completed_streams"`
	Warning                string                            `json:"warning"`
	GeneratedAt            time.Time                         `json:"generated_at"`
}

type createEmergencyTokenResponse struct {
	TokenID    string     `json:"token_id"`
	IncidentID string     `json:"incident_id"`
	Token      string     `json:"token"`
	Label      string     `json:"label,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type createEmergencyTokenRequest struct {
	Label        string     `json:"label"`
	ExpiresAt    *time.Time `json:"expires_at"`
	ExpiresAtSet bool       `json:"-"`
}

func (request *createEmergencyTokenRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name := range fields {
		if name != "label" && name != "expires_at" {
			return fmt.Errorf("unknown field %q", name)
		}
	}
	if rawLabel, ok := fields["label"]; ok {
		if err := json.Unmarshal(rawLabel, &request.Label); err != nil {
			return err
		}
	}
	rawExpiresAt, ok := fields["expires_at"]
	if !ok {
		return nil
	}
	request.ExpiresAtSet = true
	if string(rawExpiresAt) == "null" {
		request.ExpiresAt = nil
		return nil
	}
	var expiresAt time.Time
	if err := json.Unmarshal(rawExpiresAt, &expiresAt); err != nil {
		return err
	}
	request.ExpiresAt = &expiresAt
	return nil
}

const emergencyWarning = "If you are concerned about immediate safety, call emergency services now."

// createEmergencyToken is a private route that mints a read-only emergency
// capability for one incident.
func (a *API) createEmergencyToken(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if _, err := a.repo.GetIncident(r.Context(), incidentID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	} else if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	var request createEmergencyTokenRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	expiresAt := a.emergencyTokenExpiresAt(request.ExpiresAt, request.ExpiresAtSet)
	token, rawToken, err := a.repo.CreateEmergencyToken(r.Context(), incidentID, request.Label, expiresAt)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "create emergency token", err)
		return
	}

	setNoStore(w)
	// The raw token is returned only in this response; the repository stores
	// only its hash.
	writeJSON(w, http.StatusCreated, createEmergencyTokenResponse{
		TokenID:    token.ID,
		IncidentID: token.IncidentID,
		Token:      rawToken,
		Label:      token.Label,
		CreatedAt:  token.CreatedAt,
		ExpiresAt:  token.ExpiresAt,
	})
}

func (a *API) emergencyTokenExpiresAt(requestExpiresAt *time.Time, requestExpiresAtSet bool) *time.Time {
	if requestExpiresAtSet || a.defaultEmergencyTokenTTL <= 0 {
		return requestExpiresAt
	}
	expiresAt := time.Now().UTC().Add(a.defaultEmergencyTokenTTL)
	return &expiresAt
}

// revokeEmergencyToken is a private route that disables an emergency token
// without deleting its audit metadata.
func (a *API) revokeEmergencyToken(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("token_id")
	if err := a.repo.RevokeEmergencyToken(r.Context(), tokenID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "emergency_token_not_found", "emergency token was not found")
		return
	} else if err != nil {
		a.internalError(w, "revoke emergency token", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token_id": tokenID,
		"revoked":  true,
	})
}

// emergencyPage renders the public read-only HTML view after token validation.
func (a *API) emergencyPage(w http.ResponseWriter, r *http.Request) {
	setEmergencyPrivacyHeaders(w)
	data, ok := a.loadEmergencyData(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := emergencyPageTemplate.Execute(w, data); err != nil {
		a.logger.Error("render emergency page", "err", err)
	}
}

// emergencyData returns the same read-only summary as JSON for page polling.
func (a *API) emergencyData(w http.ResponseWriter, r *http.Request) {
	setEmergencyPrivacyHeaders(w)
	data, ok := a.loadEmergencyData(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func setEmergencyPrivacyHeaders(w http.ResponseWriter) {
	setPublicBrowserSecurityHeaders(w)
	setNoStore(w)
}

// loadEmergencyData collapses invalid, expired, and revoked tokens into one
// public error so callers cannot distinguish token state.
func (a *API) loadEmergencyData(w http.ResponseWriter, r *http.Request) (emergencyViewData, bool) {
	token, ok := a.loadEmergencyToken(w, r)
	if !ok {
		return emergencyViewData{}, false
	}
	data, err := a.buildEmergencyData(r.Context(), token)
	if err != nil {
		a.internalError(w, "load emergency data", err)
		return emergencyViewData{}, false
	}
	return data, true
}

// buildEmergencyData loads incident metadata only after token validation.
func (a *API) buildEmergencyData(ctx context.Context, token incidents.EmergencyToken) (emergencyViewData, error) {
	detail, err := a.repo.GetIncidentDetail(ctx, token.IncidentID)
	if err != nil {
		return emergencyViewData{}, err
	}
	return summarizeEmergencyData(detail), nil
}

func (a *API) loadEmergencyToken(w http.ResponseWriter, r *http.Request) (incidents.EmergencyToken, bool) {
	setEmergencyPrivacyHeaders(w)
	token, err := a.repo.LookupEmergencyToken(r.Context(), r.PathValue("token"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "emergency_token_invalid", "emergency token is invalid, expired, or revoked")
		return incidents.EmergencyToken{}, false
	}
	if err != nil {
		a.internalError(w, "lookup emergency token", err)
		return incidents.EmergencyToken{}, false
	}
	return token, true
}

// summarizeEmergencyData prepares viewer-safe incident data without exposing
// stored paths or encrypted file bytes.
func summarizeEmergencyData(detail incidents.IncidentDetail) emergencyViewData {
	chunkStats := collectEmergencyChunkStats(detail.Chunks)
	streams, completedStreams := summarizeEmergencyStreams(detail.Streams, chunkStats)

	return emergencyViewData{
		Incident:               summarizeIncident(detail.Incident),
		LatestCheckin:          summarizeLatestCheckin(detail.Checkins),
		ChunkCountByMediaType:  chunkStats.chunkCountByMediaType,
		LatestChunkByMediaType: chunkStats.latestChunkByMediaType,
		Media:                  summarizeEmergencyMedia(chunkStats),
		Streams:                streams,
		CompletedStreams:       completedStreams,
		Warning:                emergencyWarning,
		GeneratedAt:            time.Now().UTC(),
	}
}

type emergencyChunkStats struct {
	chunkCountByMediaType  map[string]int
	latestChunkByMediaType map[string]*emergencyChunkSummary
	chunkCountByStreamID   map[string]int
	byteCountByStreamID    map[string]int64
}

func collectEmergencyChunkStats(chunks []incidents.Chunk) emergencyChunkStats {
	stats := emergencyChunkStats{
		chunkCountByMediaType:  make(map[string]int),
		latestChunkByMediaType: make(map[string]*emergencyChunkSummary),
		chunkCountByStreamID:   make(map[string]int),
		byteCountByStreamID:    make(map[string]int64),
	}
	for _, chunk := range chunks {
		stats.chunkCountByMediaType[chunk.MediaType]++
		if chunk.StreamID != "" {
			stats.chunkCountByStreamID[chunk.StreamID]++
			stats.byteCountByStreamID[chunk.StreamID] += chunk.ByteSize
		}

		summary := summarizeChunk(chunk)
		current := stats.latestChunkByMediaType[chunk.MediaType]
		if current == nil || chunkReceivedAfter(summary, *current) {
			stats.latestChunkByMediaType[chunk.MediaType] = &summary
		}
	}
	return stats
}

func summarizeIncident(incident incidents.Incident) emergencyIncidentSummary {
	return emergencyIncidentSummary{
		ID:          incident.ID,
		Status:      incident.Status,
		ClientLabel: incident.ClientLabel,
		CreatedAt:   incident.CreatedAt,
		UpdatedAt:   incident.UpdatedAt,
	}
}

func summarizeLatestCheckin(checkins []incidents.Checkin) *emergencyCheckinSummary {
	if len(checkins) == 0 {
		return nil
	}
	summary := summarizeCheckin(checkins[len(checkins)-1])
	return &summary
}

func summarizeCheckin(checkin incidents.Checkin) emergencyCheckinSummary {
	return emergencyCheckinSummary{
		CreatedAt:            checkin.CreatedAt,
		DeviceBatteryPercent: checkin.DeviceBatteryPercent,
		DeviceNetwork:        checkin.DeviceNetwork,
		Latitude:             checkin.Latitude,
		Longitude:            checkin.Longitude,
		AccuracyMeters:       checkin.AccuracyMeters,
	}
}

func summarizeEmergencyMedia(stats emergencyChunkStats) []emergencyMediaSummary {
	mediaTypes := []string{
		incidents.MediaTypeAudio,
		incidents.MediaTypeVideo,
		incidents.MediaTypeLocation,
		incidents.MediaTypeMetadata,
	}
	media := make([]emergencyMediaSummary, 0, len(mediaTypes))
	for _, mediaType := range mediaTypes {
		media = append(media, emergencyMediaSummary{
			MediaType:   mediaType,
			ChunkCount:  stats.chunkCountByMediaType[mediaType],
			LatestChunk: stats.latestChunkByMediaType[mediaType],
		})
	}
	return media
}

func summarizeEmergencyStreams(streams []incidents.MediaStream, stats emergencyChunkStats) ([]emergencyStreamSummary, []emergencyStreamSummary) {
	summaries := make([]emergencyStreamSummary, 0, len(streams))
	completed := []emergencyStreamSummary{}
	for _, stream := range streams {
		summary := summarizeEmergencyStream(stream, stats)
		summaries = append(summaries, summary)
		if stream.Status == incidents.StreamStatusComplete {
			completed = append(completed, summary)
		}
	}
	return summaries, completed
}

func summarizeEmergencyStream(stream incidents.MediaStream, stats emergencyChunkStats) emergencyStreamSummary {
	return emergencyStreamSummary{
		ID:                 stream.ID,
		MediaType:          stream.MediaType,
		Label:              stream.Label,
		Status:             stream.Status,
		ExpectedChunkCount: stream.ExpectedChunkCount,
		CompletedAt:        stream.CompletedAt,
		FailedAt:           stream.FailedAt,
		FailureReason:      stream.FailureReason,
		ChunkCount:         stats.chunkCountByStreamID[stream.ID],
		TotalBytes:         stats.byteCountByStreamID[stream.ID],
	}
}

// summarizeChunk copies only metadata that is safe for the emergency viewer.
func summarizeChunk(chunk incidents.Chunk) emergencyChunkSummary {
	return emergencyChunkSummary{
		ChunkIndex:       chunk.ChunkIndex,
		MediaType:        chunk.MediaType,
		StartedAt:        chunk.StartedAt,
		EndedAt:          chunk.EndedAt,
		OriginalFilename: chunk.OriginalFilename,
		ByteSize:         chunk.ByteSize,
		SHA256Hex:        chunk.SHA256Hex,
		CreatedAt:        chunk.CreatedAt,
	}
}

func chunkReceivedAfter(candidate, current emergencyChunkSummary) bool {
	if candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.ChunkIndex > current.ChunkIndex
	}
	return candidate.CreatedAt.After(current.CreatedAt)
}
