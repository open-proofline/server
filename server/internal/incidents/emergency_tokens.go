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
		SELECT id, incident_id, token_hash, label, created_at, expires_at, revoked_at
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
