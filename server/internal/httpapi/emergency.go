package httpapi

import (
	"context"
	"errors"
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

type emergencyViewData struct {
	Incident               emergencyIncidentSummary          `json:"incident"`
	LatestCheckin          *emergencyCheckinSummary          `json:"latest_checkin,omitempty"`
	ChunkCountByMediaType  map[string]int                    `json:"chunk_count_by_media_type"`
	LatestChunkByMediaType map[string]*emergencyChunkSummary `json:"latest_chunk_by_media_type"`
	Media                  []emergencyMediaSummary           `json:"media"`
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

const emergencyWarning = "If you are concerned about immediate safety, call emergency services now."

func (a *API) createEmergencyToken(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if _, err := a.repo.GetIncident(r.Context(), incidentID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	} else if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	var request struct {
		Label     string     `json:"label"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	token, rawToken, err := a.repo.CreateEmergencyToken(r.Context(), incidentID, request.Label, request.ExpiresAt)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "create emergency token", err)
		return
	}

	writeJSON(w, http.StatusCreated, createEmergencyTokenResponse{
		TokenID:    token.ID,
		IncidentID: token.IncidentID,
		Token:      rawToken,
		Label:      token.Label,
		CreatedAt:  token.CreatedAt,
		ExpiresAt:  token.ExpiresAt,
	})
}

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

func (a *API) emergencyPage(w http.ResponseWriter, r *http.Request) {
	data, ok := a.loadEmergencyData(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := emergencyPageTemplate.Execute(w, data); err != nil {
		a.logger.Error("render emergency page", "err", err)
	}
}

func (a *API) emergencyData(w http.ResponseWriter, r *http.Request) {
	data, ok := a.loadEmergencyData(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (a *API) loadEmergencyData(w http.ResponseWriter, r *http.Request) (emergencyViewData, bool) {
	data, err := a.buildEmergencyData(r.Context(), r.PathValue("token"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "emergency_token_invalid", "emergency token is invalid, expired, or revoked")
		return emergencyViewData{}, false
	}
	if err != nil {
		a.internalError(w, "load emergency data", err)
		return emergencyViewData{}, false
	}
	return data, true
}

func (a *API) buildEmergencyData(ctx context.Context, rawToken string) (emergencyViewData, error) {
	token, err := a.repo.LookupEmergencyToken(ctx, rawToken)
	if err != nil {
		return emergencyViewData{}, err
	}
	detail, err := a.repo.GetIncidentDetail(ctx, token.IncidentID)
	if err != nil {
		return emergencyViewData{}, err
	}
	if err := a.repo.UpdateEmergencyTokenLastUsed(ctx, token.ID); err != nil {
		return emergencyViewData{}, err
	}

	return summarizeEmergencyData(detail), nil
}

func summarizeEmergencyData(detail incidents.IncidentDetail) emergencyViewData {
	chunkCounts := make(map[string]int)
	latestChunks := make(map[string]*emergencyChunkSummary)
	for _, chunk := range detail.Chunks {
		chunkCounts[chunk.MediaType]++
		summary := summarizeChunk(chunk)
		current := latestChunks[chunk.MediaType]
		if current == nil || summary.ChunkIndex > current.ChunkIndex {
			latestChunks[chunk.MediaType] = &summary
		}
	}

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
			ChunkCount:  chunkCounts[mediaType],
			LatestChunk: latestChunks[mediaType],
		})
	}

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

	return emergencyViewData{
		Incident: emergencyIncidentSummary{
			ID:          detail.Incident.ID,
			Status:      detail.Incident.Status,
			ClientLabel: detail.Incident.ClientLabel,
			CreatedAt:   detail.Incident.CreatedAt,
			UpdatedAt:   detail.Incident.UpdatedAt,
		},
		LatestCheckin:          latestCheckin,
		ChunkCountByMediaType:  chunkCounts,
		LatestChunkByMediaType: latestChunks,
		Media:                  media,
		Warning:                emergencyWarning,
		GeneratedAt:            time.Now().UTC(),
	}
}

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
