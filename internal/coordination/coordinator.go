package coordination

import (
	"context"
	"errors"
	"fmt"
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

// Pinger is the small client surface needed to validate a configured backend.
type Pinger interface {
	Ping(context.Context) error
	Close() error
}

// Valkey uses a Valkey/Redis-compatible service for optional coordination.
type Valkey struct {
	client Pinger
}

// NewValkey builds a Valkey coordinator around a client. Tests can provide a
// fake client without requiring a running Valkey service.
func NewValkey(client Pinger) (*Valkey, error) {
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

// Close releases the underlying Valkey client.
func (v *Valkey) Close() error {
	return v.client.Close()
}
