package incidents

import "time"

const (
	// StatusOpen means the incident can still accept chunks and checkins.
	StatusOpen = "open"
	// StatusClosed means the incident metadata remains readable, but new chunk
	// uploads are rejected by the HTTP layer.
	StatusClosed = "closed"

	// MediaTypeAudio identifies encrypted audio chunks.
	MediaTypeAudio = "audio"
	// MediaTypeVideo identifies encrypted video chunks.
	MediaTypeVideo = "video"
	// MediaTypeLocation identifies encrypted location chunks.
	MediaTypeLocation = "location"
	// MediaTypeMetadata identifies encrypted metadata chunks.
	MediaTypeMetadata = "metadata"
)

// Incident is the top-level recording session tracked by the backend.
type Incident struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
	ClientLabel string    `json:"client_label,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

// Chunk records metadata for an accepted encrypted upload.
type Chunk struct {
	ID               string    `json:"id"`
	IncidentID       string    `json:"incident_id"`
	ChunkIndex       int       `json:"chunk_index"`
	MediaType        string    `json:"media_type"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	StoredPath       string    `json:"stored_path"`
	ByteSize         int64     `json:"byte_size"`
	SHA256Hex        string    `json:"sha256_hex"`
	CreatedAt        time.Time `json:"created_at"`
}

// Checkin records optional device status and location metadata for an incident.
type Checkin struct {
	ID                   string    `json:"id"`
	IncidentID           string    `json:"incident_id"`
	CreatedAt            time.Time `json:"created_at"`
	DeviceBatteryPercent *int      `json:"device_battery_percent,omitempty"`
	DeviceNetwork        *string   `json:"device_network,omitempty"`
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	AccuracyMeters       *float64  `json:"accuracy_meters,omitempty"`
}

// IncidentDetail combines one incident with its chunk and checkin metadata.
type IncidentDetail struct {
	Incident Incident  `json:"incident"`
	Chunks   []Chunk   `json:"chunks"`
	Checkins []Checkin `json:"checkins"`
}

// CreateChunkParams contains metadata saved after a chunk file has been safely
// written and hash-verified.
type CreateChunkParams struct {
	IncidentID       string
	ChunkIndex       int
	MediaType        string
	StartedAt        time.Time
	EndedAt          time.Time
	OriginalFilename string
	StoredPath       string
	ByteSize         int64
	SHA256Hex        string
}

// CreateCheckinParams contains optional device metadata for a checkin.
type CreateCheckinParams struct {
	DeviceBatteryPercent *int
	DeviceNetwork        *string
	Latitude             *float64
	Longitude            *float64
	AccuracyMeters       *float64
}

// ValidMediaType reports whether mediaType is one of the supported chunk
// categories.
func ValidMediaType(mediaType string) bool {
	switch mediaType {
	case MediaTypeAudio, MediaTypeVideo, MediaTypeLocation, MediaTypeMetadata:
		return true
	default:
		return false
	}
}
