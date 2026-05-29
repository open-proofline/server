package postgresdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/open-proofline/server/internal/incidents"
)

// CreateMediaStream inserts a new open stream for one incident.
func (r *Repository) CreateMediaStream(ctx context.Context, incidentID, mediaType, label string) (incidents.MediaStream, error) {
	now := time.Now().UTC()
	id, err := newID("str")
	if err != nil {
		return incidents.MediaStream{}, err
	}
	stream := incidents.MediaStream{
		ID:         id,
		IncidentID: incidentID,
		MediaType:  mediaType,
		Label:      label,
		Status:     incidents.StreamStatusOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO media_streams (
			id, incident_id, media_type, label, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		stream.ID,
		stream.IncidentID,
		stream.MediaType,
		nullableString(stream.Label),
		stream.Status,
		stream.CreatedAt,
		stream.UpdatedAt,
	)
	if err != nil {
		if isIntegrityConstraint(err) {
			return incidents.MediaStream{}, incidents.ErrNotFound
		}
		return incidents.MediaStream{}, fmt.Errorf("insert postgres media stream: %w", err)
	}
	return stream, nil
}

// ListMediaStreams returns streams for an incident ordered by creation time.
func (r *Repository) ListMediaStreams(ctx context.Context, incidentID string) ([]incidents.MediaStream, error) {
	rows, err := r.db.QueryContext(ctx, mediaStreamSelect(`
		WHERE incident_id = $1
		ORDER BY created_at ASC, id ASC`), incidentID)
	if err != nil {
		return nil, fmt.Errorf("list postgres media streams: %w", err)
	}
	defer rows.Close()

	streams := []incidents.MediaStream{}
	for rows.Next() {
		stream, err := scanMediaStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres media streams: %w", err)
	}
	return streams, nil
}

// ListCompletedMediaStreams returns completed streams for an incident.
func (r *Repository) ListCompletedMediaStreams(ctx context.Context, incidentID string) ([]incidents.MediaStream, error) {
	rows, err := r.db.QueryContext(ctx, mediaStreamSelect(`
		WHERE incident_id = $1 AND status = $2
		ORDER BY completed_at ASC, id ASC`), incidentID, incidents.StreamStatusComplete)
	if err != nil {
		return nil, fmt.Errorf("list completed postgres media streams: %w", err)
	}
	defer rows.Close()

	streams := []incidents.MediaStream{}
	for rows.Next() {
		stream, err := scanMediaStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completed postgres media streams: %w", err)
	}
	return streams, nil
}

// GetMediaStream returns one stream by incident and stream ID.
func (r *Repository) GetMediaStream(ctx context.Context, incidentID, streamID string) (incidents.MediaStream, error) {
	row := r.db.QueryRowContext(ctx, mediaStreamSelect(`
		WHERE incident_id = $1 AND id = $2`), incidentID, streamID)

	stream, err := scanMediaStream(row)
	if errors.Is(err, sql.ErrNoRows) {
		return incidents.MediaStream{}, incidents.ErrNotFound
	}
	if err != nil {
		return incidents.MediaStream{}, fmt.Errorf("get postgres media stream: %w", err)
	}
	return stream, nil
}

// ListStreamChunks returns chunks attached to one stream, ordered by index.
func (r *Repository) ListStreamChunks(ctx context.Context, incidentID, streamID string) ([]incidents.Chunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		FROM chunks
		WHERE incident_id = $1 AND stream_id = $2
		ORDER BY chunk_index ASC`,
		incidentID,
		streamID,
	)
	if err != nil {
		return nil, fmt.Errorf("list postgres stream chunks: %w", err)
	}
	defer rows.Close()

	chunks := []incidents.Chunk{}
	for rows.Next() {
		chunk, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres stream chunks: %w", err)
	}
	return chunks, nil
}

// CompleteMediaStream marks an open stream complete after callers have
// validated the stream's chunks and files.
func (r *Repository) CompleteMediaStream(ctx context.Context, incidentID, streamID string, expectedChunkCount int) (incidents.MediaStream, error) {
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return incidents.MediaStream{}, fmt.Errorf("begin complete postgres media stream: %w", err)
	}

	mediaType, err := lockOpenMediaStream(ctx, tx, incidentID, streamID)
	if err != nil {
		_ = tx.Rollback()
		return incidents.MediaStream{}, err
	}
	if err := validateCompleteStreamChunkRows(ctx, tx, incidentID, streamID, mediaType, expectedChunkCount); err != nil {
		_ = tx.Rollback()
		return incidents.MediaStream{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE media_streams
		SET status = $1, expected_chunk_count = $2, updated_at = $3, completed_at = $4,
			failed_at = NULL, failure_reason = NULL
		WHERE incident_id = $5 AND id = $6`,
		incidents.StreamStatusComplete,
		expectedChunkCount,
		now,
		now,
		incidentID,
		streamID,
	); err != nil {
		_ = tx.Rollback()
		return incidents.MediaStream{}, fmt.Errorf("complete postgres media stream: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return incidents.MediaStream{}, fmt.Errorf("commit complete postgres media stream: %w", err)
	}
	return r.GetMediaStream(ctx, incidentID, streamID)
}

func lockOpenMediaStream(ctx context.Context, tx *sql.Tx, incidentID, streamID string) (string, error) {
	var streamStatus string
	var mediaType string
	err := tx.QueryRowContext(ctx, `
		SELECT status, media_type
		FROM media_streams
		WHERE incident_id = $1 AND id = $2
		FOR UPDATE`,
		incidentID,
		streamID,
	).Scan(&streamStatus, &mediaType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", incidents.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock postgres media stream: %w", err)
	}
	if streamStatus != incidents.StreamStatusOpen {
		return "", incidents.ErrInvalidState
	}
	return mediaType, nil
}

func validateCompleteStreamChunkRows(ctx context.Context, tx *sql.Tx, incidentID, streamID, mediaType string, expectedChunkCount int) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT chunk_index, media_type
		FROM chunks
		WHERE incident_id = $1 AND stream_id = $2
		ORDER BY chunk_index ASC`,
		incidentID,
		streamID,
	)
	if err != nil {
		return fmt.Errorf("list postgres completion chunks: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var chunkIndex int
		var chunkMediaType string
		if err := rows.Scan(&chunkIndex, &chunkMediaType); err != nil {
			return fmt.Errorf("scan postgres completion chunk: %w", err)
		}
		if chunkIndex != count || chunkMediaType != mediaType {
			return incidents.ErrInvalidState
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate postgres completion chunks: %w", err)
	}
	if count != expectedChunkCount {
		return incidents.ErrInvalidState
	}
	return nil
}

// FailMediaStream marks an open stream failed while preserving uploaded chunks.
func (r *Repository) FailMediaStream(ctx context.Context, incidentID, streamID, reason string) (incidents.MediaStream, error) {
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return incidents.MediaStream{}, fmt.Errorf("begin fail postgres media stream: %w", err)
	}
	if _, err := lockOpenMediaStream(ctx, tx, incidentID, streamID); err != nil {
		_ = tx.Rollback()
		return incidents.MediaStream{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE media_streams
		SET status = $1, updated_at = $2, failed_at = $3, failure_reason = $4,
			completed_at = NULL
		WHERE incident_id = $5 AND id = $6`,
		incidents.StreamStatusFailed,
		now,
		now,
		nullableString(reason),
		incidentID,
		streamID,
	); err != nil {
		_ = tx.Rollback()
		return incidents.MediaStream{}, fmt.Errorf("fail postgres media stream: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return incidents.MediaStream{}, fmt.Errorf("commit fail postgres media stream: %w", err)
	}
	return r.GetMediaStream(ctx, incidentID, streamID)
}

func mediaStreamSelect(where string) string {
	return `
		SELECT id, incident_id, media_type, label, status, expected_chunk_count,
			created_at, updated_at, completed_at, failed_at, failure_reason
		FROM media_streams
	` + where
}
