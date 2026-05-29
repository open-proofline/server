package postgresdb

import (
	"context"
	"fmt"
	"time"

	"github.com/open-proofline/server/internal/incidents"
)

// CreateCheckin inserts a checkin for an incident.
func (r *Repository) CreateCheckin(ctx context.Context, incidentID string, params incidents.CreateCheckinParams) (incidents.Checkin, error) {
	id, err := newID("cin")
	if err != nil {
		return incidents.Checkin{}, err
	}
	checkin := incidents.Checkin{
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		checkin.ID,
		checkin.IncidentID,
		checkin.CreatedAt,
		nullableInt(checkin.DeviceBatteryPercent),
		nullableStringPtr(checkin.DeviceNetwork),
		nullableFloat(checkin.Latitude),
		nullableFloat(checkin.Longitude),
		nullableFloat(checkin.AccuracyMeters),
	)
	if err != nil {
		if isIntegrityConstraint(err) {
			return incidents.Checkin{}, incidents.ErrNotFound
		}
		return incidents.Checkin{}, fmt.Errorf("insert postgres checkin: %w", err)
	}

	return checkin, nil
}

// ListCheckins returns checkin metadata for an incident.
func (r *Repository) ListCheckins(ctx context.Context, incidentID string) ([]incidents.Checkin, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, created_at, device_battery_percent, device_network,
			latitude, longitude, accuracy_meters
		FROM checkins
		WHERE incident_id = $1
		ORDER BY created_at ASC`,
		incidentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list postgres checkins: %w", err)
	}
	defer rows.Close()

	checkins := []incidents.Checkin{}
	for rows.Next() {
		checkin, err := scanCheckin(rows)
		if err != nil {
			return nil, err
		}
		checkins = append(checkins, checkin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres checkins: %w", err)
	}
	return checkins, nil
}
