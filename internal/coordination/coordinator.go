package coordination

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrUnavailable reports that an explicitly configured coordination backend
// cannot currently be used. It intentionally carries no backend address,
// credential, key, token, or request detail.
var ErrUnavailable = errors.New("coordination backend unavailable")

// Coordinator is the narrow server boundary for optional short-lived
// coordination. Current upload semantics do not depend on it.
type Coordinator interface {
	Check(context.Context) error
	Close() error
}

// Noop is the default coordination backend.
type Noop struct{}

// NewNone returns the disabled/default coordination backend.
func NewNone() Noop {
	return Noop{}
}

// Check is always successful for the disabled/default backend.
func (Noop) Check(context.Context) error {
	return nil
}

// Close is a no-op for the disabled/default backend.
func (Noop) Close() error {
	return nil
}

// Client is the small client surface needed for configured coordination.
type Client interface {
	Ping(context.Context) error
	IncrementWithExpiry(context.Context, string, time.Duration) (int64, error)
	Close() error
}

// Valkey uses a Valkey/Redis-compatible service for optional coordination.
type Valkey struct {
	client Client
}

// NewValkey builds a Valkey coordinator around a client. Tests can provide a
// fake client without requiring a running Valkey service.
func NewValkey(client Client) (*Valkey, error) {
	if client == nil {
		return nil, fmt.Errorf("missing valkey coordination client")
	}
	return &Valkey{client: client}, nil
}

// Check verifies that the configured service is reachable. The returned error
// is deliberately generic so startup logs do not expose private endpoints.
func (v *Valkey) Check(ctx context.Context) error {
	if err := v.client.Ping(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return ErrUnavailable
	}
	return nil
}

// Allow records a fixed-window rate-limit hit against a server-controlled key.
func (v *Valkey) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	count, err := v.client.IncrementWithExpiry(ctx, key, window)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, err
		}
		return false, ErrUnavailable
	}
	return count <= int64(limit), nil
}

// Close releases the underlying Valkey client.
func (v *Valkey) Close() error {
	return v.client.Close()
}
