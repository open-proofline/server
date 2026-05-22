package incidents

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotFound indicates that a requested incident, chunk, or related row
	// does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicate indicates that SQLite rejected a duplicate chunk identity.
	ErrDuplicate = errors.New("duplicate")
)

// Repository stores incident, chunk, and checkin metadata in SQLite.
type Repository struct {
	db *sql.DB
}

// NewRepository wraps db with incident-specific query methods.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateIncident inserts a new open incident.
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

// GetIncident returns one incident by ID.
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

// GetIncidentDetail returns an incident with its chunk and checkin metadata.
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

// CloseIncident marks an incident closed so the HTTP layer rejects later chunk
// uploads.
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

// ChunkExists reports whether an incident already has a chunk with the same
// media type and index.
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

// CreateChunk inserts metadata for a chunk after the blob has been committed to
// disk.
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
		// The schema's unique constraint is the final duplicate guard. This
		// matters if two uploads race past the HTTP preflight check.
		if isConstraint(err) {
			return Chunk{}, ErrDuplicate
		}
		return Chunk{}, fmt.Errorf("insert chunk: %w", err)
	}

	return chunk, nil
}

// ListChunks returns chunk metadata for an incident without loading file bytes.
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

	chunks := []Chunk{}
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

// GetChunkByKey returns one chunk by incident, media type, and chunk index.
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

// CreateCheckin inserts a checkin for an incident.
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

// ListCheckins returns checkin metadata for an incident.
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

	checkins := []Checkin{}
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

// CreateEmergencyToken creates a read-only token scoped to one incident and
// returns the raw token once for the caller to share.
func (r *Repository) CreateEmergencyToken(ctx context.Context, incidentID, label string, expiresAt *time.Time) (EmergencyToken, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return EmergencyToken{}, "", fmt.Errorf("generate emergency token: %w", err)
	}
	// Generate a URL-safe 256-bit bearer token and persist only its SHA-256
	// hash so database disclosure does not reveal usable emergency links.
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashEmergencyToken(rawToken)

	id, err := newID("etk")
	if err != nil {
		return EmergencyToken{}, "", err
	}
	now := time.Now().UTC()
	token := EmergencyToken{
		ID:         id,
		IncidentID: incidentID,
		TokenHash:  tokenHash,
		Label:      label,
		CreatedAt:  now,
		ExpiresAt:  utcTimePtr(expiresAt),
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO emergency_tokens (
			id, incident_id, token_hash, label, created_at, expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?)`,
		token.ID,
		token.IncidentID,
		token.TokenHash,
		nullableString(token.Label),
		formatDBTime(token.CreatedAt),
		nullableTime(token.ExpiresAt),
	)
	if err != nil {
		// Constraint failures include missing incident foreign keys and the
		// vanishingly unlikely token-hash collision; callers treat both as a
		// failed token creation.
		if isConstraint(err) {
			return EmergencyToken{}, "", ErrNotFound
		}
		return EmergencyToken{}, "", fmt.Errorf("insert emergency token: %w", err)
	}

	return token, rawToken, nil
}

// LookupEmergencyToken returns token metadata when rawToken is valid, unexpired,
// and not revoked.
func (r *Repository) LookupEmergencyToken(ctx context.Context, rawToken string) (EmergencyToken, error) {
	tokenHash := hashEmergencyToken(rawToken)
	row := r.db.QueryRowContext(ctx, `
		SELECT id, incident_id, token_hash, label, created_at, expires_at, revoked_at, last_used_at
		FROM emergency_tokens
		WHERE token_hash = ?`,
		tokenHash,
	)

	token, err := scanEmergencyToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return EmergencyToken{}, ErrNotFound
	}
	if err != nil {
		return EmergencyToken{}, fmt.Errorf("lookup emergency token: %w", err)
	}
	// The indexed lookup should already match the hash; keep a constant-time
	// comparison as a final equality check before considering token state.
	if subtle.ConstantTimeCompare([]byte(token.TokenHash), []byte(tokenHash)) != 1 {
		return EmergencyToken{}, ErrNotFound
	}
	if token.RevokedAt != nil {
		return EmergencyToken{}, ErrNotFound
	}
	if token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now().UTC()) {
		return EmergencyToken{}, ErrNotFound
	}

	return token, nil
}

// RevokeEmergencyToken revokes a token so it can no longer read emergency data.
func (r *Repository) RevokeEmergencyToken(ctx context.Context, tokenID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE emergency_tokens
		SET revoked_at = ?
		WHERE id = ? AND revoked_at IS NULL`,
		formatDBTime(time.Now().UTC()),
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("revoke emergency token: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke emergency token rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateEmergencyTokenLastUsed records successful emergency token use.
func (r *Repository) UpdateEmergencyTokenLastUsed(ctx context.Context, tokenID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE emergency_tokens
		SET last_used_at = ?
		WHERE id = ?`,
		formatDBTime(time.Now().UTC()),
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("update emergency token last used: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update emergency token last used rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
