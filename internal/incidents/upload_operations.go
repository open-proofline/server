package incidents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const uploadOperationColumns = `
	id, operation, idempotency_key_hash, incident_id, stream_id, chunk_index,
	media_type, started_at, ended_at, original_filename, byte_size, sha256_hex,
	fingerprint_hash, state, chunk_id, stored_path, created_at, updated_at`

// ReserveUploadOperation binds an idempotency-key hash to immutable upload
// inputs. Reusing the same key with different inputs returns ErrIdempotencyConflict.
func (r *Repository) ReserveUploadOperation(ctx context.Context, params UploadOperationParams) (UploadOperation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return UploadOperation{}, fmt.Errorf("begin reserve upload operation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := requireActiveIncidentTx(ctx, tx, params.IncidentID); err != nil {
		return UploadOperation{}, err
	}

	now := time.Now().UTC()
	id, err := newID("uop")
	if err != nil {
		return UploadOperation{}, err
	}
	operation := UploadOperation{
		ID:                 id,
		Operation:          params.Operation,
		IdempotencyKeyHash: params.IdempotencyKeyHash,
		IncidentID:         params.IncidentID,
		StreamID:           params.StreamID,
		ChunkIndex:         params.ChunkIndex,
		MediaType:          params.MediaType,
		StartedAt:          params.StartedAt.UTC(),
		EndedAt:            params.EndedAt.UTC(),
		OriginalFilename:   params.OriginalFilename,
		ByteSize:           params.ByteSize,
		SHA256Hex:          params.SHA256Hex,
		FingerprintHash:    params.FingerprintHash,
		State:              UploadOperationStateReserved,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO upload_operations (
			id, operation, idempotency_key_hash, incident_id, stream_id, chunk_index,
			media_type, started_at, ended_at, original_filename, byte_size, sha256_hex,
			fingerprint_hash, state, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		operation.ID,
		operation.Operation,
		operation.IdempotencyKeyHash,
		operation.IncidentID,
		nullableString(operation.StreamID),
		operation.ChunkIndex,
		operation.MediaType,
		formatDBTime(operation.StartedAt),
		formatDBTime(operation.EndedAt),
		nullableString(operation.OriginalFilename),
		operation.ByteSize,
		operation.SHA256Hex,
		operation.FingerprintHash,
		operation.State,
		formatDBTime(operation.CreatedAt),
		formatDBTime(operation.UpdatedAt),
	)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return UploadOperation{}, fmt.Errorf("commit reserve upload operation: %w", err)
		}
		return operation, nil
	}
	if !isConstraint(err) {
		return UploadOperation{}, fmt.Errorf("insert upload operation: %w", err)
	}
	_ = tx.Rollback()

	existing, getErr := r.getUploadOperationByKey(ctx, params.Operation, params.IdempotencyKeyHash)
	if errors.Is(getErr, ErrNotFound) {
		return UploadOperation{}, ErrNotFound
	}
	if getErr != nil {
		return UploadOperation{}, getErr
	}
	if !uploadOperationMatchesParams(existing, params) {
		return UploadOperation{}, ErrIdempotencyConflict
	}
	return existing, nil
}

// CompleteUploadOperation records the final chunk row confirmed for a reserved
// idempotent upload operation.
func (r *Repository) CompleteUploadOperation(ctx context.Context, params UploadOperationParams, chunk Chunk) (UploadOperation, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE upload_operations
		SET state = ?, chunk_id = ?, stored_path = ?, updated_at = ?
		WHERE operation = ? AND idempotency_key_hash = ? AND fingerprint_hash = ?`,
		UploadOperationStateMetadataCommitted,
		chunk.ID,
		chunk.StoredPath,
		formatDBTime(now),
		params.Operation,
		params.IdempotencyKeyHash,
		params.FingerprintHash,
	)
	if err != nil {
		return UploadOperation{}, fmt.Errorf("complete upload operation: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return UploadOperation{}, fmt.Errorf("complete upload operation rows affected: %w", err)
	}
	if rowsAffected == 0 {
		existing, getErr := r.getUploadOperationByKey(ctx, params.Operation, params.IdempotencyKeyHash)
		if errors.Is(getErr, ErrNotFound) {
			return UploadOperation{}, ErrNotFound
		}
		if getErr != nil {
			return UploadOperation{}, getErr
		}
		if !uploadOperationMatchesParams(existing, params) {
			return UploadOperation{}, ErrIdempotencyConflict
		}
		return UploadOperation{}, ErrNotFound
	}
	return r.getUploadOperationByKey(ctx, params.Operation, params.IdempotencyKeyHash)
}

func (r *Repository) getUploadOperationByKey(ctx context.Context, operation, keyHash string) (UploadOperation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+uploadOperationColumns+`
		FROM upload_operations
		WHERE operation = ? AND idempotency_key_hash = ?`,
		operation,
		keyHash,
	)
	uploadOperation, err := scanUploadOperation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return UploadOperation{}, ErrNotFound
	}
	if err != nil {
		return UploadOperation{}, fmt.Errorf("get upload operation: %w", err)
	}
	return uploadOperation, nil
}

func uploadOperationMatchesParams(operation UploadOperation, params UploadOperationParams) bool {
	return operation.Operation == params.Operation &&
		operation.IdempotencyKeyHash == params.IdempotencyKeyHash &&
		operation.IncidentID == params.IncidentID &&
		operation.StreamID == params.StreamID &&
		operation.ChunkIndex == params.ChunkIndex &&
		operation.MediaType == params.MediaType &&
		operation.StartedAt.Equal(params.StartedAt.UTC()) &&
		operation.EndedAt.Equal(params.EndedAt.UTC()) &&
		operation.OriginalFilename == params.OriginalFilename &&
		operation.ByteSize == params.ByteSize &&
		operation.SHA256Hex == params.SHA256Hex &&
		operation.FingerprintHash == params.FingerprintHash
}
