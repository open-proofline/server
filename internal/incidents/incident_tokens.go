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

// CreateIncidentToken creates a read-only token scoped to one incident and
// returns the raw token once for the caller to share.
func (r *Repository) CreateIncidentToken(ctx context.Context, incidentID, label string, expiresAt *time.Time) (IncidentToken, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return IncidentToken{}, "", fmt.Errorf("generate incident token: %w", err)
	}
	// Generate a URL-safe 256-bit bearer token and persist only its SHA-256
	// hash so database disclosure does not reveal usable incident viewer links.
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashIncidentToken(rawToken)

	id, err := newID("itk")
	if err != nil {
		return IncidentToken{}, "", err
	}
	now := time.Now().UTC()
	token := IncidentToken{
		ID:         id,
		IncidentID: incidentID,
		TokenHash:  tokenHash,
		Label:      label,
		CreatedAt:  now,
		ExpiresAt:  utcTimePtr(expiresAt),
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO incident_tokens (
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
			return IncidentToken{}, "", ErrNotFound
		}
		return IncidentToken{}, "", fmt.Errorf("insert incident token: %w", err)
	}

	return token, rawToken, nil
}

// LookupIncidentToken returns token metadata when rawToken is valid, unexpired,
// and not revoked.
func (r *Repository) LookupIncidentToken(ctx context.Context, rawToken string) (IncidentToken, error) {
	tokenHash := hashIncidentToken(rawToken)
	row := r.db.QueryRowContext(ctx, `
		SELECT id, incident_id, token_hash, label, created_at, expires_at, revoked_at
		FROM incident_tokens
		WHERE token_hash = ?`,
		tokenHash,
	)

	token, err := scanIncidentToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return IncidentToken{}, ErrNotFound
	}
	if err != nil {
		return IncidentToken{}, fmt.Errorf("lookup incident token: %w", err)
	}
	// The indexed lookup should already match the hash; keep a constant-time
	// comparison as a final equality check before considering token state.
	if subtle.ConstantTimeCompare([]byte(token.TokenHash), []byte(tokenHash)) != 1 {
		return IncidentToken{}, ErrNotFound
	}
	if token.RevokedAt != nil {
		return IncidentToken{}, ErrNotFound
	}
	if token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now().UTC()) {
		return IncidentToken{}, ErrNotFound
	}

	return token, nil
}

// RevokeIncidentToken revokes a token so it can no longer read incident viewer data.
func (r *Repository) RevokeIncidentToken(ctx context.Context, tokenID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE incident_tokens
		SET revoked_at = ?
		WHERE id = ? AND revoked_at IS NULL`,
		formatDBTime(time.Now().UTC()),
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("revoke incident token: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke incident token rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
