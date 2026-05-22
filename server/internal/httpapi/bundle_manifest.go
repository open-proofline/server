package httpapi

import (
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
