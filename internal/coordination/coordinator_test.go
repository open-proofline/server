package coordination

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNoopCoordinator(t *testing.T) {
	coord := NewNone()

	if err := coord.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if err := coord.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestValkeyCoordinatorCheck(t *testing.T) {
	client := &fakePinger{}
	coord, err := NewValkey(client)
	if err != nil {
		t.Fatalf("NewValkey: %v", err)
	}

	if err := coord.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if client.pings != 1 {
		t.Fatalf("pings = %d, want 1", client.pings)
	}
}

func TestValkeyCoordinatorCheckFailureIsSafe(t *testing.T) {
	client := &fakePinger{
		pingErr: errors.New("dial 10.0.0.5:6379 with password secret failed"),
	}
	coord, err := NewValkey(client)
	if err != nil {
		t.Fatalf("NewValkey: %v", err)
	}

	err = coord.Check(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Check error = %v, want ErrUnavailable", err)
	}
	if strings.Contains(err.Error(), "10.0.0.5") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("coordination error exposed backend detail: %v", err)
	}
}

func TestValkeyCoordinatorClose(t *testing.T) {
	client := &fakePinger{closeErr: errors.New("close failed")}
	coord, err := NewValkey(client)
	if err != nil {
		t.Fatalf("NewValkey: %v", err)
	}

	err = coord.Close()
	if !errors.Is(err, client.closeErr) {
		t.Fatalf("Close error = %v, want %v", err, client.closeErr)
	}
	if client.closes != 1 {
		t.Fatalf("closes = %d, want 1", client.closes)
	}
}

func TestNewValkeyRequiresClient(t *testing.T) {
	_, err := NewValkey(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakePinger struct {
	pingErr  error
	closeErr error
	pings    int
	closes   int
}

func (f *fakePinger) Ping(context.Context) error {
	f.pings++
	return f.pingErr
}

func (f *fakePinger) Close() error {
	f.closes++
	return f.closeErr
}
