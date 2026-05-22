package incidents

import "database/sql"

type scanner interface {
	Scan(dest ...any) error
}

func scanIncident(s scanner) (Incident, error) {
	var incident Incident
	var createdAt string
	var updatedAt string
	var clientLabel sql.NullString
	var notes sql.NullString
	if err := s.Scan(&incident.ID, &createdAt, &updatedAt, &incident.Status, &clientLabel, &notes); err != nil {
		return Incident{}, err
	}
	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return Incident{}, err
	}
	parsedUpdatedAt, err := parseDBTime(updatedAt)
	if err != nil {
		return Incident{}, err
	}
	incident.CreatedAt = parsedCreatedAt
	incident.UpdatedAt = parsedUpdatedAt
	if clientLabel.Valid {
		incident.ClientLabel = clientLabel.String
	}
	if notes.Valid {
		incident.Notes = notes.String
	}
	return incident, nil
}

func scanChunk(s scanner) (Chunk, error) {
	var chunk Chunk
	var startedAt string
	var endedAt string
	var createdAt string
	var originalFilename sql.NullString
	if err := s.Scan(
		&chunk.ID,
		&chunk.IncidentID,
		&chunk.ChunkIndex,
		&chunk.MediaType,
		&startedAt,
		&endedAt,
		&originalFilename,
		&chunk.StoredPath,
		&chunk.ByteSize,
		&chunk.SHA256Hex,
		&createdAt,
	); err != nil {
		return Chunk{}, err
	}
	parsedStartedAt, err := parseDBTime(startedAt)
	if err != nil {
		return Chunk{}, err
	}
	parsedEndedAt, err := parseDBTime(endedAt)
	if err != nil {
		return Chunk{}, err
	}
	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return Chunk{}, err
	}
	chunk.StartedAt = parsedStartedAt
	chunk.EndedAt = parsedEndedAt
	chunk.CreatedAt = parsedCreatedAt
	if originalFilename.Valid {
		chunk.OriginalFilename = originalFilename.String
	}
	return chunk, nil
}

func scanCheckin(s scanner) (Checkin, error) {
	var checkin Checkin
	var createdAt string
	var battery sql.NullInt64
	var network sql.NullString
	var latitude sql.NullFloat64
	var longitude sql.NullFloat64
	var accuracy sql.NullFloat64
	if err := s.Scan(
		&checkin.ID,
		&checkin.IncidentID,
		&createdAt,
		&battery,
		&network,
		&latitude,
		&longitude,
		&accuracy,
	); err != nil {
		return Checkin{}, err
	}
	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return Checkin{}, err
	}
	checkin.CreatedAt = parsedCreatedAt
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

func scanEmergencyToken(s scanner) (EmergencyToken, error) {
	var token EmergencyToken
	var label sql.NullString
	var createdAt string
	var expiresAt sql.NullString
	var revokedAt sql.NullString
	var lastUsedAt sql.NullString
	if err := s.Scan(
		&token.ID,
		&token.IncidentID,
		&token.TokenHash,
		&label,
		&createdAt,
		&expiresAt,
		&revokedAt,
		&lastUsedAt,
	); err != nil {
		return EmergencyToken{}, err
	}
	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return EmergencyToken{}, err
	}
	token.CreatedAt = parsedCreatedAt
	if label.Valid {
		token.Label = label.String
	}
	if token.ExpiresAt, err = nullableDBTime(expiresAt); err != nil {
		return EmergencyToken{}, err
	}
	if token.RevokedAt, err = nullableDBTime(revokedAt); err != nil {
		return EmergencyToken{}, err
	}
	if token.LastUsedAt, err = nullableDBTime(lastUsedAt); err != nil {
		return EmergencyToken{}, err
	}
	return token, nil
}
