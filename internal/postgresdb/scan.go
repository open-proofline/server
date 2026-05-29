package postgresdb

import (
	"database/sql"
	"time"

	"github.com/open-proofline/server/internal/incidents"
)

type scanner interface {
	Scan(dest ...any) error
}

func scanIncident(s scanner) (incidents.Incident, error) {
	var incident incidents.Incident
	var clientLabel sql.NullString
	var notes sql.NullString
	if err := s.Scan(&incident.ID, &incident.CreatedAt, &incident.UpdatedAt, &incident.Status, &clientLabel, &notes); err != nil {
		return incidents.Incident{}, err
	}
	incident.CreatedAt = incident.CreatedAt.UTC()
	incident.UpdatedAt = incident.UpdatedAt.UTC()
	if clientLabel.Valid {
		incident.ClientLabel = clientLabel.String
	}
	if notes.Valid {
		incident.Notes = notes.String
	}
	return incident, nil
}

func scanChunk(s scanner) (incidents.Chunk, error) {
	var chunk incidents.Chunk
	var streamID sql.NullString
	var originalFilename sql.NullString
	if err := s.Scan(
		&chunk.ID,
		&chunk.IncidentID,
		&streamID,
		&chunk.ChunkIndex,
		&chunk.MediaType,
		&chunk.StartedAt,
		&chunk.EndedAt,
		&originalFilename,
		&chunk.StoredPath,
		&chunk.ByteSize,
		&chunk.SHA256Hex,
		&chunk.CreatedAt,
	); err != nil {
		return incidents.Chunk{}, err
	}
	chunk.StartedAt = chunk.StartedAt.UTC()
	chunk.EndedAt = chunk.EndedAt.UTC()
	chunk.CreatedAt = chunk.CreatedAt.UTC()
	if streamID.Valid {
		chunk.StreamID = streamID.String
	}
	if originalFilename.Valid {
		chunk.OriginalFilename = originalFilename.String
	}
	return chunk, nil
}

func scanMediaStream(s scanner) (incidents.MediaStream, error) {
	var stream incidents.MediaStream
	var label sql.NullString
	var expectedChunkCount sql.NullInt64
	var completedAt sql.NullTime
	var failedAt sql.NullTime
	var failureReason sql.NullString
	if err := s.Scan(
		&stream.ID,
		&stream.IncidentID,
		&stream.MediaType,
		&label,
		&stream.Status,
		&expectedChunkCount,
		&stream.CreatedAt,
		&stream.UpdatedAt,
		&completedAt,
		&failedAt,
		&failureReason,
	); err != nil {
		return incidents.MediaStream{}, err
	}
	stream.CreatedAt = stream.CreatedAt.UTC()
	stream.UpdatedAt = stream.UpdatedAt.UTC()
	if label.Valid {
		stream.Label = label.String
	}
	if expectedChunkCount.Valid {
		count := int(expectedChunkCount.Int64)
		stream.ExpectedChunkCount = &count
	}
	stream.CompletedAt = nullableDBTime(completedAt)
	stream.FailedAt = nullableDBTime(failedAt)
	if failureReason.Valid {
		stream.FailureReason = failureReason.String
	}
	return stream, nil
}

func scanCheckin(s scanner) (incidents.Checkin, error) {
	var checkin incidents.Checkin
	var battery sql.NullInt64
	var network sql.NullString
	var latitude sql.NullFloat64
	var longitude sql.NullFloat64
	var accuracy sql.NullFloat64
	if err := s.Scan(
		&checkin.ID,
		&checkin.IncidentID,
		&checkin.CreatedAt,
		&battery,
		&network,
		&latitude,
		&longitude,
		&accuracy,
	); err != nil {
		return incidents.Checkin{}, err
	}
	checkin.CreatedAt = checkin.CreatedAt.UTC()
	if battery.Valid {
		value := int(battery.Int64)
		checkin.DeviceBatteryPercent = &value
	}
	if network.Valid {
		value := network.String
		checkin.DeviceNetwork = &value
	}
	if latitude.Valid {
		value := latitude.Float64
		checkin.Latitude = &value
	}
	if longitude.Valid {
		value := longitude.Float64
		checkin.Longitude = &value
	}
	if accuracy.Valid {
		value := accuracy.Float64
		checkin.AccuracyMeters = &value
	}
	return checkin, nil
}

func scanIncidentToken(s scanner) (incidents.IncidentToken, error) {
	var token incidents.IncidentToken
	var label sql.NullString
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime
	if err := s.Scan(
		&token.ID,
		&token.IncidentID,
		&token.TokenHash,
		&label,
		&token.CreatedAt,
		&expiresAt,
		&revokedAt,
	); err != nil {
		return incidents.IncidentToken{}, err
	}
	token.CreatedAt = token.CreatedAt.UTC()
	if label.Valid {
		token.Label = label.String
	}
	token.ExpiresAt = nullableDBTime(expiresAt)
	token.RevokedAt = nullableDBTime(revokedAt)
	return token, nil
}

func nullableDBTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	utc := value.Time.UTC()
	return &utc
}
