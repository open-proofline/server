package incidents

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateIncident(ctx context.Context, clientLabel, notes string) (Incident, error) {
	now := time.Now().UTC()
	id, err := newID("inc")
	if err != nil {
		return Incident{}, err
	}
	incident := Incident{
		ID:          id,
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      StatusOpen,
		ClientLabel: clientLabel,
		Notes:       notes,
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO incidents (id, created_at, updated_at, status, client_label, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		incident.ID,
		formatDBTime(incident.CreatedAt),
		formatDBTime(incident.UpdatedAt),
		incident.Status,
		nullableString(incident.ClientLabel),
		nullableString(incident.Notes),
	)
	if err != nil {
		return Incident{}, fmt.Errorf("insert incident: %w", err)
	}

	return incident, nil
}

func (r *Repository) GetIncident(ctx context.Context, id string) (Incident, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, created_at, updated_at, status, client_label, notes
		FROM incidents
		WHERE id = ?`, id)

	incident, err := scanIncident(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Incident{}, ErrNotFound
	}
	if err != nil {
		return Incident{}, fmt.Errorf("get incident: %w", err)
	}
	return incident, nil
}

func (r *Repository) GetIncidentDetail(ctx context.Context, id string) (IncidentDetail, error) {
	incident, err := r.GetIncident(ctx, id)
	if err != nil {
		return IncidentDetail{}, err
	}
	chunks, err := r.ListChunks(ctx, id)
	if err != nil {
		return IncidentDetail{}, err
	}
	checkins, err := r.ListCheckins(ctx, id)
	if err != nil {
		return IncidentDetail{}, err
	}

	return IncidentDetail{
		Incident: incident,
		Chunks:   chunks,
		Checkins: checkins,
	}, nil
}

func (r *Repository) CloseIncident(ctx context.Context, id string) (Incident, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = ?, updated_at = ?
		WHERE id = ?`,
		StatusClosed,
		formatDBTime(now),
		id,
	)
	if err != nil {
		return Incident{}, fmt.Errorf("close incident: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Incident{}, fmt.Errorf("close incident rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return Incident{}, ErrNotFound
	}
	return r.GetIncident(ctx, id)
}

func (r *Repository) ChunkExists(ctx context.Context, incidentID, mediaType string, chunkIndex int) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1
		FROM chunks
		WHERE incident_id = ? AND media_type = ? AND chunk_index = ?`,
		incidentID,
		mediaType,
		chunkIndex,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check chunk exists: %w", err)
	}
	return true, nil
}

func (r *Repository) CreateChunk(ctx context.Context, params CreateChunkParams) (Chunk, error) {
	id, err := newID("chk")
	if err != nil {
		return Chunk{}, err
	}
	chunk := Chunk{
		ID:               id,
		IncidentID:       params.IncidentID,
		ChunkIndex:       params.ChunkIndex,
		MediaType:        params.MediaType,
		StartedAt:        params.StartedAt,
		EndedAt:          params.EndedAt,
		OriginalFilename: params.OriginalFilename,
		StoredPath:       params.StoredPath,
		ByteSize:         params.ByteSize,
		SHA256Hex:        params.SHA256Hex,
		CreatedAt:        time.Now().UTC(),
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO chunks (
			id, incident_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chunk.ID,
		chunk.IncidentID,
		chunk.ChunkIndex,
		chunk.MediaType,
		formatDBTime(chunk.StartedAt),
		formatDBTime(chunk.EndedAt),
		nullableString(chunk.OriginalFilename),
		chunk.StoredPath,
		chunk.ByteSize,
		chunk.SHA256Hex,
		formatDBTime(chunk.CreatedAt),
	)
	if err != nil {
		if isConstraint(err) {
			return Chunk{}, ErrDuplicate
		}
		return Chunk{}, fmt.Errorf("insert chunk: %w", err)
	}

	return chunk, nil
}

func (r *Repository) ListChunks(ctx context.Context, incidentID string) ([]Chunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		FROM chunks
		WHERE incident_id = ?
		ORDER BY chunk_index ASC, media_type ASC`,
		incidentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		chunk, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}
	return chunks, nil
}

func (r *Repository) GetChunkByKey(ctx context.Context, incidentID, mediaType string, chunkIndex int) (Chunk, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, incident_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		FROM chunks
		WHERE incident_id = ? AND media_type = ? AND chunk_index = ?`,
		incidentID,
		mediaType,
		chunkIndex,
	)

	chunk, err := scanChunk(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Chunk{}, ErrNotFound
	}
	if err != nil {
		return Chunk{}, fmt.Errorf("get chunk: %w", err)
	}
	return chunk, nil
}

func (r *Repository) CreateCheckin(ctx context.Context, incidentID string, params CreateCheckinParams) (Checkin, error) {
	id, err := newID("cin")
	if err != nil {
		return Checkin{}, err
	}
	checkin := Checkin{
		ID:                   id,
		IncidentID:           incidentID,
		CreatedAt:            time.Now().UTC(),
		DeviceBatteryPercent: params.DeviceBatteryPercent,
		DeviceNetwork:        params.DeviceNetwork,
		Latitude:             params.Latitude,
		Longitude:            params.Longitude,
		AccuracyMeters:       params.AccuracyMeters,
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO checkins (
			id, incident_id, created_at, device_battery_percent, device_network,
			latitude, longitude, accuracy_meters
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		checkin.ID,
		checkin.IncidentID,
		formatDBTime(checkin.CreatedAt),
		nullableInt(checkin.DeviceBatteryPercent),
		nullableStringPtr(checkin.DeviceNetwork),
		nullableFloat(checkin.Latitude),
		nullableFloat(checkin.Longitude),
		nullableFloat(checkin.AccuracyMeters),
	)
	if err != nil {
		if isConstraint(err) {
			return Checkin{}, ErrNotFound
		}
		return Checkin{}, fmt.Errorf("insert checkin: %w", err)
	}

	return checkin, nil
}

func (r *Repository) ListCheckins(ctx context.Context, incidentID string) ([]Checkin, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, created_at, device_battery_percent, device_network,
			latitude, longitude, accuracy_meters
		FROM checkins
		WHERE incident_id = ?
		ORDER BY created_at ASC`,
		incidentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list checkins: %w", err)
	}
	defer rows.Close()

	var checkins []Checkin
	for rows.Next() {
		checkin, err := scanCheckin(rows)
		if err != nil {
			return nil, err
		}
		checkins = append(checkins, checkin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkins: %w", err)
	}
	return checkins, nil
}

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

func formatDBTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseDBTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse database time %q: %w", value, err)
	}
	return parsed, nil
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullableStringPtr(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func nullableFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func isConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint
}

func newID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:]), nil
}
