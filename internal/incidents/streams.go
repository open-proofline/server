package incidents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateMediaStream inserts a new open stream for one incident.
func (r *Repository) CreateMediaStream(ctx context.Context, incidentID, mediaType, label string) (MediaStream, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MediaStream{}, fmt.Errorf("begin create media stream: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireActiveIncidentTx(ctx, tx, incidentID); err != nil {
		return MediaStream{}, err
	}

	now := time.Now().UTC()
	id, err := newID("str")
	if err != nil {
		return MediaStream{}, err
	}
	stream := MediaStream{
		ID:         id,
		IncidentID: incidentID,
		MediaType:  mediaType,
		Label:      label,
		Status:     StreamStatusOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO media_streams (
			id, incident_id, media_type, label, status, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		stream.ID,
		stream.IncidentID,
		stream.MediaType,
		nullableString(stream.Label),
		stream.Status,
		formatDBTime(stream.CreatedAt),
		formatDBTime(stream.UpdatedAt),
	)
	if err != nil {
		if isConstraint(err) {
			return MediaStream{}, ErrNotFound
		}
		return MediaStream{}, fmt.Errorf("insert media stream: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return MediaStream{}, fmt.Errorf("commit create media stream: %w", err)
	}
	return stream, nil
}

// ListMediaStreams returns streams for an incident ordered by creation time.
func (r *Repository) ListMediaStreams(ctx context.Context, incidentID string) ([]MediaStream, error) {
	rows, err := r.db.QueryContext(ctx, mediaStreamSelect(`
		WHERE incident_id = ?
		ORDER BY created_at ASC, id ASC`), incidentID)
	if err != nil {
		return nil, fmt.Errorf("list media streams: %w", err)
	}
	defer rows.Close()

	streams := []MediaStream{}
	for rows.Next() {
		stream, err := scanMediaStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media streams: %w", err)
	}
	return streams, nil
}

// ListCompletedMediaStreams returns completed streams for an incident.
func (r *Repository) ListCompletedMediaStreams(ctx context.Context, incidentID string) ([]MediaStream, error) {
	rows, err := r.db.QueryContext(ctx, mediaStreamSelect(`
		WHERE incident_id = ? AND status = ?
		ORDER BY completed_at ASC, id ASC`), incidentID, StreamStatusComplete)
	if err != nil {
		return nil, fmt.Errorf("list completed media streams: %w", err)
	}
	defer rows.Close()

	streams := []MediaStream{}
	for rows.Next() {
		stream, err := scanMediaStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completed media streams: %w", err)
	}
	return streams, nil
}

// GetMediaStream returns one stream by incident and stream ID.
func (r *Repository) GetMediaStream(ctx context.Context, incidentID, streamID string) (MediaStream, error) {
	row := r.db.QueryRowContext(ctx, mediaStreamSelect(`
		WHERE incident_id = ? AND id = ?`), incidentID, streamID)

	stream, err := scanMediaStream(row)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaStream{}, ErrNotFound
	}
	if err != nil {
		return MediaStream{}, fmt.Errorf("get media stream: %w", err)
	}
	return stream, nil
}

// ListStreamChunks returns chunks attached to one stream, ordered by index.
func (r *Repository) ListStreamChunks(ctx context.Context, incidentID, streamID string) ([]Chunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			original_filename, stored_path, byte_size, sha256_hex, created_at
		FROM chunks
		WHERE incident_id = ? AND stream_id = ?
		ORDER BY chunk_index ASC`,
		incidentID,
		streamID,
	)
	if err != nil {
		return nil, fmt.Errorf("list stream chunks: %w", err)
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
		return nil, fmt.Errorf("iterate stream chunks: %w", err)
	}
	return chunks, nil
}

// CompleteMediaStream marks an open stream complete after callers have
// validated the stream's chunks and files.
func (r *Repository) CompleteMediaStream(ctx context.Context, incidentID, streamID string, expectedChunkCount int) (MediaStream, error) {
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MediaStream{}, fmt.Errorf("begin complete media stream: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireActiveIncidentTx(ctx, tx, incidentID); err != nil {
		return MediaStream{}, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE media_streams
		SET status = ?, expected_chunk_count = ?, updated_at = ?, completed_at = ?,
			failed_at = NULL, failure_reason = NULL
		WHERE incident_id = ? AND id = ? AND status = ?`,
		StreamStatusComplete,
		expectedChunkCount,
		formatDBTime(now),
		formatDBTime(now),
		incidentID,
		streamID,
		StreamStatusOpen,
	)
	if err != nil {
		return MediaStream{}, fmt.Errorf("complete media stream: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return MediaStream{}, fmt.Errorf("complete media stream rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if _, err := r.GetMediaStream(ctx, incidentID, streamID); errors.Is(err, ErrNotFound) {
			return MediaStream{}, ErrNotFound
		}
		return MediaStream{}, ErrInvalidState
	}
	var mediaType string
	err = tx.QueryRowContext(ctx, `
		SELECT media_type
		FROM media_streams
		WHERE incident_id = ? AND id = ?`,
		incidentID,
		streamID,
	).Scan(&mediaType)
	if err != nil {
		return MediaStream{}, fmt.Errorf("read completed stream media type: %w", err)
	}
	if err := validateCompleteStreamChunkRows(ctx, tx, incidentID, streamID, mediaType, expectedChunkCount); err != nil {
		return MediaStream{}, err
	}
	if err := tx.Commit(); err != nil {
		return MediaStream{}, fmt.Errorf("commit complete media stream: %w", err)
	}
	return r.GetMediaStream(ctx, incidentID, streamID)
}

func validateCompleteStreamChunkRows(ctx context.Context, tx *sql.Tx, incidentID, streamID, mediaType string, expectedChunkCount int) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT chunk_index, media_type
		FROM chunks
		WHERE incident_id = ? AND stream_id = ?
		ORDER BY chunk_index ASC`,
		incidentID,
		streamID,
	)
	if err != nil {
		return fmt.Errorf("list completion chunks: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var chunkIndex int
		var chunkMediaType string
		if err := rows.Scan(&chunkIndex, &chunkMediaType); err != nil {
			return fmt.Errorf("scan completion chunk: %w", err)
		}
		if chunkIndex != count || chunkMediaType != mediaType {
			return ErrInvalidState
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate completion chunks: %w", err)
	}
	if count != expectedChunkCount {
		return ErrInvalidState
	}
	return nil
}

// FailMediaStream marks an open stream failed while preserving uploaded chunks.
func (r *Repository) FailMediaStream(ctx context.Context, incidentID, streamID, reason string) (MediaStream, error) {
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MediaStream{}, fmt.Errorf("begin fail media stream: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireActiveIncidentTx(ctx, tx, incidentID); err != nil {
		return MediaStream{}, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE media_streams
		SET status = ?, updated_at = ?, failed_at = ?, failure_reason = ?,
			completed_at = NULL
		WHERE incident_id = ? AND id = ? AND status = ?`,
		StreamStatusFailed,
		formatDBTime(now),
		formatDBTime(now),
		nullableString(reason),
		incidentID,
		streamID,
		StreamStatusOpen,
	)
	if err != nil {
		return MediaStream{}, fmt.Errorf("fail media stream: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return MediaStream{}, fmt.Errorf("fail media stream rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if _, err := r.GetMediaStream(ctx, incidentID, streamID); errors.Is(err, ErrNotFound) {
			return MediaStream{}, ErrNotFound
		}
		return MediaStream{}, ErrInvalidState
	}
	if err := tx.Commit(); err != nil {
		return MediaStream{}, fmt.Errorf("commit fail media stream: %w", err)
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
