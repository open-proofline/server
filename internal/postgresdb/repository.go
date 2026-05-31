package postgresdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/open-proofline/server/internal/incidents"
)

// Repository stores incident metadata and related rows in PostgreSQL.
type Repository struct {
	db *sql.DB
}

// NewRepository wraps db with incident-specific PostgreSQL query methods.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Check verifies that the PostgreSQL metadata handle is reachable.
func (r *Repository) Check(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// CreateIncident inserts a new open legacy incident without an owner account.
func (r *Repository) CreateIncident(ctx context.Context, clientLabel, notes string) (incidents.Incident, error) {
	return r.CreateIncidentForAccount(ctx, "", clientLabel, notes)
}

// CreateIncidentForAccount inserts a new open incident owned by accountID.
func (r *Repository) CreateIncidentForAccount(ctx context.Context, accountID, clientLabel, notes string) (incidents.Incident, error) {
	now := time.Now().UTC()
	id, err := newID("inc")
	if err != nil {
		return incidents.Incident{}, err
	}
	incident := incidents.Incident{
		ID:             id,
		OwnerAccountID: accountID,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         incidents.StatusOpen,
		ClientLabel:    clientLabel,
		Notes:          notes,
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO incidents (
			id, owner_account_id, created_at, updated_at, status, client_label, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		incident.ID,
		nullableString(incident.OwnerAccountID),
		incident.CreatedAt,
		incident.UpdatedAt,
		incident.Status,
		nullableString(incident.ClientLabel),
		nullableString(incident.Notes),
	)
	if err != nil {
		if isIntegrityConstraint(err) {
			return incidents.Incident{}, incidents.ErrNotFound
		}
		return incidents.Incident{}, fmt.Errorf("insert postgres incident: %w", err)
	}

	return incident, nil
}

// GetIncident returns one incident by ID.
func (r *Repository) GetIncident(ctx context.Context, id string) (incidents.Incident, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, owner_account_id, created_at, updated_at, status, client_label, notes
		FROM incidents
		WHERE id = $1`, id)

	incident, err := scanIncident(row)
	if errors.Is(err, sql.ErrNoRows) {
		return incidents.Incident{}, incidents.ErrNotFound
	}
	if err != nil {
		return incidents.Incident{}, fmt.Errorf("get postgres incident: %w", err)
	}
	return incident, nil
}

// GetIncidentDetail returns an incident with its chunk, stream, and checkin metadata.
func (r *Repository) GetIncidentDetail(ctx context.Context, id string) (incidents.IncidentDetail, error) {
	incident, err := r.GetIncident(ctx, id)
	if err != nil {
		return incidents.IncidentDetail{}, err
	}
	streams, err := r.ListMediaStreams(ctx, id)
	if err != nil {
		return incidents.IncidentDetail{}, err
	}
	chunks, err := r.ListChunks(ctx, id)
	if err != nil {
		return incidents.IncidentDetail{}, err
	}
	checkins, err := r.ListCheckins(ctx, id)
	if err != nil {
		return incidents.IncidentDetail{}, err
	}

	return incidents.IncidentDetail{
		Incident: incident,
		Streams:  streams,
		Chunks:   chunks,
		Checkins: checkins,
	}, nil
}

// CloseIncident marks an incident closed so later chunk metadata inserts fail.
func (r *Repository) CloseIncident(ctx context.Context, id string) (incidents.Incident, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return incidents.Incident{}, fmt.Errorf("begin close postgres incident: %w", err)
	}
	if err := lockIncident(ctx, tx, id); err != nil {
		_ = tx.Rollback()
		return incidents.Incident{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE incidents
		SET status = $1, updated_at = $2
		WHERE id = $3`,
		incidents.StatusClosed,
		time.Now().UTC(),
		id,
	); err != nil {
		_ = tx.Rollback()
		return incidents.Incident{}, fmt.Errorf("close postgres incident: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return incidents.Incident{}, fmt.Errorf("commit close postgres incident: %w", err)
	}
	return r.GetIncident(ctx, id)
}

func lockIncident(ctx context.Context, tx *sql.Tx, incidentID string) error {
	var status string
	err := tx.QueryRowContext(ctx, `
		SELECT status
		FROM incidents
		WHERE id = $1
		FOR UPDATE`,
		incidentID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return incidents.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock postgres incident: %w", err)
	}
	return nil
}
