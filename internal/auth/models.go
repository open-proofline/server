package auth

import (
	"errors"
	"time"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

var (
	ErrDuplicate = errors.New("duplicate auth row")
	ErrNotFound  = errors.New("auth row not found")
)

type Account struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"-"`
	Role              string    `json:"role"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
}

type Session struct {
	ID        string     `json:"id"`
	AccountID string     `json:"account_id"`
	TokenHash string     `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type CreateAccountParams struct {
	Username     string
	PasswordHash string
	Role         string
}

func ValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}
