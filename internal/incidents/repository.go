package incidents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotFound indicates that a requested incident, chunk, or related row
	// does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicate indicates that metadata storage rejected a duplicate chunk identity.
	ErrDuplicate = errors.New("duplicate")
	// ErrInvalidState indicates that a requested state transition is not
	// allowed for the current stream state.
	ErrInvalidState = errors.New("invalid state")
	// ErrIncidentClosed indicates that a chunk insert raced with an incident
	// close and the incident no longer accepts uploads.
	ErrIncidentClosed = errors.New("incident closed")
)

// Repository stores incident metadata and related rows in SQLite.
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
	streams, err := r.ListMediaStreams(ctx, id)
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
		Streams:  streams,
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
