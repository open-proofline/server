package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 12
	MaxPasswordLength = 72
)

func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 64 {
		return errors.New("username must be between 3 and 64 characters")
	}
	for _, ch := range username {
		if ch > unicode.MaxASCII {
			return errors.New("username must contain only ASCII characters")
		}
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			continue
		}
		switch ch {
		case '.', '-', '_', '@':
			continue
		default:
			return errors.New("username may contain only letters, digits, '.', '-', '_', and '@'")
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d bytes", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must be at most %d bytes", MaxPasswordLength)
	}
	return nil
}

func HashPassword(password string, cost int) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func VerifyPassword(passwordHash, password string) bool {
	if passwordHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func SpendPasswordHashCost(password string, cost int) {
	_, _ = bcrypt.GenerateFromPassword([]byte(password), cost)
}

func SessionTokenHash(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func EncodeToken(bytes []byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes)
}
