package incidents

import "time"

const (
	StatusOpen   = "open"
	StatusClosed = "closed"

	MediaTypeAudio    = "audio"
	MediaTypeVideo    = "video"
	MediaTypeLocation = "location"
	MediaTypeMetadata = "metadata"
)

type Incident struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
	ClientLabel string    `json:"client_label,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

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

type IncidentDetail struct {
	Incident Incident  `json:"incident"`
	Chunks   []Chunk   `json:"chunks"`
	Checkins []Checkin `json:"checkins"`
}

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

type CreateCheckinParams struct {
	DeviceBatteryPercent *int
	DeviceNetwork        *string
	Latitude             *float64
	Longitude            *float64
	AccuracyMeters       *float64
}

func ValidMediaType(mediaType string) bool {
	switch mediaType {
	case MediaTypeAudio, MediaTypeVideo, MediaTypeLocation, MediaTypeMetadata:
		return true
	default:
		return false
	}
}
