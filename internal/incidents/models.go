package incidents

import "time"

const (
	// StatusOpen means the incident can still accept chunks and checkins.
	StatusOpen = "open"
	// StatusClosed means the incident metadata remains readable, but new chunk
	// uploads are rejected by the HTTP layer.
	StatusClosed = "closed"

	// StreamStatusOpen means chunks can still be uploaded to a media stream.
	StreamStatusOpen = "open"
	// StreamStatusComplete means a media stream has a verified contiguous set
	// of chunks and can be downloaded as an encrypted evidence bundle.
	StreamStatusComplete = "complete"
	// StreamStatusFailed means recording stopped without a complete stream.
	StreamStatusFailed = "failed"

	// MediaTypeAudio identifies encrypted audio chunks.
	MediaTypeAudio = "audio"
	// MediaTypeVideo identifies encrypted video chunks.
	MediaTypeVideo = "video"
	// MediaTypeLocation identifies encrypted location chunks.
	MediaTypeLocation = "location"
	// MediaTypeMetadata identifies encrypted metadata chunks.
	MediaTypeMetadata = "metadata"

	// IncidentModeEmergency identifies an incident where the user chose an
	// emergency capture mode. The label alone does not grant access or notify
	// anyone.
	IncidentModeEmergency = "emergency"
	// IncidentModeInteractionRecord identifies a non-emergency interaction record.
	IncidentModeInteractionRecord = "interaction_record"
	// IncidentModeSafetyCheck identifies a timed safety-check incident.
	IncidentModeSafetyCheck = "safety_check"
	// IncidentModeEvidenceNote identifies a note or attachment-oriented incident.
	IncidentModeEvidenceNote = "evidence_note"

	// CaptureProfileAudioVideoLocation records an intent to capture audio, video,
	// and location where available.
	CaptureProfileAudioVideoLocation = "audio_video_location"
	// CaptureProfileAudioLocation records an intent to capture audio and location.
	CaptureProfileAudioLocation = "audio_location"
	// CaptureProfileLocationCheckin records a location/check-in oriented flow.
	CaptureProfileLocationCheckin = "location_checkin"
	// CaptureProfileNoteOrAttachment records a note or attachment-oriented flow.
	CaptureProfileNoteOrAttachment = "note_or_attachment"
	// CaptureProfileCustom records a future client-selected capture combination.
	CaptureProfileCustom = "custom"

	// EscalationPolicyNone records that no automatic escalation policy was chosen.
	EscalationPolicyNone = "none"
	// EscalationPolicyTrustedContactsOnStart records a future trusted-contact
	// escalation policy. The current backend does not send notifications.
	EscalationPolicyTrustedContactsOnStart = "trusted_contacts_on_start"
	// EscalationPolicyTrustedContactsOnMissedCheckin records a future missed
	// check-in escalation policy. The current backend does not run timers.
	EscalationPolicyTrustedContactsOnMissedCheckin = "trusted_contacts_on_missed_checkin"
	// EscalationPolicyUrgentTrustedContactAlert records a future urgent
	// trusted-contact policy. The current backend does not send notifications.
	EscalationPolicyUrgentTrustedContactAlert = "urgent_trusted_contact_alert"

	// SharingStatePrivate records that no sharing has been declared for the
	// incident metadata.
	SharingStatePrivate = "private"
	// SharingStateTrustedContactAccess records future trusted-contact access state
	// metadata. The current backend does not grant trusted-contact access.
	SharingStateTrustedContactAccess = "trusted_contact_access"
	// SharingStatePublicLinkCreated records public-link sharing state metadata.
	SharingStatePublicLinkCreated = "public_link_created"
	// SharingStateLegalExportCreated records legal/export sharing state metadata.
	SharingStateLegalExportCreated = "legal_export_created"
	// SharingStateRevokedOrExpired records that a sharing state was revoked or
	// expired.
	SharingStateRevokedOrExpired = "revoked_or_expired"

	// UploadOperationUploadChunk identifies the chunk-upload idempotency route.
	UploadOperationUploadChunk = "upload_chunk"

	// UploadOperationStateReserved means an idempotency key has been bound to
	// immutable upload inputs but no final chunk row is confirmed yet.
	UploadOperationStateReserved = "reserved"
	// UploadOperationStateMetadataCommitted means the upload operation has a
	// confirmed final chunk row and can be replayed safely.
	UploadOperationStateMetadataCommitted = "metadata_committed"
)

// Incident is the top-level recording session tracked by the backend.
type Incident struct {
	ID               string    `json:"id"`
	OwnerAccountID   string    `json:"owner_account_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Status           string    `json:"status"`
	ClientLabel      string    `json:"client_label,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	IncidentMode     string    `json:"incident_mode,omitempty"`
	CaptureProfile   string    `json:"capture_profile,omitempty"`
	EscalationPolicy string    `json:"escalation_policy,omitempty"`
	SharingState     string    `json:"sharing_state,omitempty"`
}

// MediaStream groups encrypted chunks that belong to one recording stream.
type MediaStream struct {
	ID                 string     `json:"id"`
	IncidentID         string     `json:"incident_id"`
	MediaType          string     `json:"media_type"`
	Label              string     `json:"label,omitempty"`
	Status             string     `json:"status"`
	ExpectedChunkCount *int       `json:"expected_chunk_count,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	FailedAt           *time.Time `json:"failed_at,omitempty"`
	FailureReason      string     `json:"failure_reason,omitempty"`
}

// Chunk records metadata for an accepted encrypted upload.
type Chunk struct {
	ID               string    `json:"id"`
	IncidentID       string    `json:"incident_id"`
	StreamID         string    `json:"stream_id,omitempty"`
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
	Incident Incident      `json:"incident"`
	Streams  []MediaStream `json:"streams"`
	Chunks   []Chunk       `json:"chunks"`
	Checkins []Checkin     `json:"checkins"`
}

// CreateIncidentParams contains optional metadata stored with a new incident.
// Incident mode fields are metadata only; they do not grant access, send
// notifications, change retention, or change key custody.
type CreateIncidentParams struct {
	ClientLabel      string
	Notes            string
	IncidentMode     string
	CaptureProfile   string
	EscalationPolicy string
	SharingState     string
}

// CreateChunkParams contains metadata saved after a chunk file has been safely
// written and hash-verified.
type CreateChunkParams struct {
	IncidentID       string
	StreamID         string
	ChunkIndex       int
	MediaType        string
	StartedAt        time.Time
	EndedAt          time.Time
	OriginalFilename string
	StoredPath       string
	ByteSize         int64
	SHA256Hex        string
}

// UploadOperation records durable idempotency state for one private write
// operation. IdempotencyKeyHash stores a SHA-256 hash, never the raw key.
type UploadOperation struct {
	ID                 string
	Operation          string
	IdempotencyKeyHash string
	IncidentID         string
	StreamID           string
	ChunkIndex         int
	MediaType          string
	StartedAt          time.Time
	EndedAt            time.Time
	OriginalFilename   string
	ByteSize           int64
	SHA256Hex          string
	FingerprintHash    string
	State              string
	ChunkID            string
	StoredPath         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// UploadOperationParams contains the key hash, normalized chunk identity, and
// immutable request fingerprint fields used to reserve an idempotent upload.
type UploadOperationParams struct {
	Operation          string
	IdempotencyKeyHash string
	IncidentID         string
	StreamID           string
	ChunkIndex         int
	MediaType          string
	StartedAt          time.Time
	EndedAt            time.Time
	OriginalFilename   string
	ByteSize           int64
	SHA256Hex          string
	FingerprintHash    string
}

// CreateCheckinParams contains optional device metadata for a checkin.
type CreateCheckinParams struct {
	DeviceBatteryPercent *int
	DeviceNetwork        *string
	Latitude             *float64
	Longitude            *float64
	AccuracyMeters       *float64
}

// IncidentToken records read-only incident viewer access scoped to one incident.
// TokenHash is stored instead of the raw token and is not exposed in API JSON.
type IncidentToken struct {
	ID         string     `json:"id"`
	IncidentID string     `json:"incident_id"`
	TokenHash  string     `json:"-"`
	Label      string     `json:"label,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
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

// ValidIncidentMode reports whether value is one of the server-supported
// incident-mode identifiers.
func ValidIncidentMode(value string) bool {
	switch value {
	case IncidentModeEmergency, IncidentModeInteractionRecord, IncidentModeSafetyCheck, IncidentModeEvidenceNote:
		return true
	default:
		return false
	}
}

// ValidCaptureProfile reports whether value is one of the server-supported
// capture-profile identifiers.
func ValidCaptureProfile(value string) bool {
	switch value {
	case CaptureProfileAudioVideoLocation, CaptureProfileAudioLocation, CaptureProfileLocationCheckin, CaptureProfileNoteOrAttachment, CaptureProfileCustom:
		return true
	default:
		return false
	}
}

// ValidEscalationPolicy reports whether value is one of the server-supported
// escalation-policy identifiers.
func ValidEscalationPolicy(value string) bool {
	switch value {
	case EscalationPolicyNone, EscalationPolicyTrustedContactsOnStart, EscalationPolicyTrustedContactsOnMissedCheckin, EscalationPolicyUrgentTrustedContactAlert:
		return true
	default:
		return false
	}
}

// ValidSharingState reports whether value is one of the server-supported
// sharing-state identifiers.
func ValidSharingState(value string) bool {
	switch value {
	case SharingStatePrivate, SharingStateTrustedContactAccess, SharingStatePublicLinkCreated, SharingStateLegalExportCreated, SharingStateRevokedOrExpired:
		return true
	default:
		return false
	}
}
