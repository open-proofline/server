package incidents

import (
	"context"
	"fmt"
	"time"
)

// CreateCheckin inserts a checkin for an incident.
func (r *Repository) CreateCheckin(ctx context.Context, incidentID string, params CreateCheckinParams) (Checkin, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Checkin{}, fmt.Errorf("begin create checkin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireActiveIncidentTx(ctx, tx, incidentID); err != nil {
		return Checkin{}, err
	}

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

	_, err = tx.ExecContext(ctx, `
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
	if err := tx.Commit(); err != nil {
		return Checkin{}, fmt.Errorf("commit create checkin: %w", err)
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
