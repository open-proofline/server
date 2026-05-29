package httpapi

import (
	"context"
	"time"

	"github.com/open-proofline/server/internal/incidents"
)

// MetadataRepository is the incident metadata boundary required by the HTTP
// handlers. SQLite remains the default implementation, and optional PostgreSQL
// support must preserve token hashing, duplicate guards, state checks, and
// stream completion validation.
type MetadataRepository interface {
	CreateIncident(ctx context.Context, clientLabel, notes string) (incidents.Incident, error)
	GetIncident(ctx context.Context, id string) (incidents.Incident, error)
	GetIncidentDetail(ctx context.Context, id string) (incidents.IncidentDetail, error)
	CloseIncident(ctx context.Context, id string) (incidents.Incident, error)

	CreateCheckin(ctx context.Context, incidentID string, params incidents.CreateCheckinParams) (incidents.Checkin, error)

	ChunkExists(ctx context.Context, incidentID, streamID, mediaType string, chunkIndex int) (bool, error)
	CreateChunk(ctx context.Context, params incidents.CreateChunkParams) (incidents.Chunk, error)
	ListChunks(ctx context.Context, incidentID string) ([]incidents.Chunk, error)
	GetChunkByKey(ctx context.Context, incidentID, mediaType string, chunkIndex int) (incidents.Chunk, error)

	CreateMediaStream(ctx context.Context, incidentID, mediaType, label string) (incidents.MediaStream, error)
	ListMediaStreams(ctx context.Context, incidentID string) ([]incidents.MediaStream, error)
	ListCompletedMediaStreams(ctx context.Context, incidentID string) ([]incidents.MediaStream, error)
	GetMediaStream(ctx context.Context, incidentID, streamID string) (incidents.MediaStream, error)
	ListStreamChunks(ctx context.Context, incidentID, streamID string) ([]incidents.Chunk, error)
	CompleteMediaStream(ctx context.Context, incidentID, streamID string, expectedChunkCount int) (incidents.MediaStream, error)
	FailMediaStream(ctx context.Context, incidentID, streamID, reason string) (incidents.MediaStream, error)

	CreateIncidentToken(ctx context.Context, incidentID, label string, expiresAt *time.Time) (incidents.IncidentToken, string, error)
	LookupIncidentToken(ctx context.Context, rawToken string) (incidents.IncidentToken, error)
	RevokeIncidentToken(ctx context.Context, tokenID string) error
}

var _ MetadataRepository = (*incidents.Repository)(nil)
