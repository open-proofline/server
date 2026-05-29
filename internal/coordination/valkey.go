package coordination

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ValkeyOptions contains connection settings for a Valkey/Redis-compatible
// coordination service.
type ValkeyOptions struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	UseTLS       bool
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewValkeyClient creates a Valkey coordinator backed by go-redis.
func NewValkeyClient(opts ValkeyOptions) (*Valkey, error) {
	if opts.Addr == "" {
		return nil, fmt.Errorf("missing valkey coordination address")
	}
	redisOptions := &redis.Options{
		Addr:         opts.Addr,
		Username:     opts.Username,
		Password:     opts.Password,
		DB:           opts.DB,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
	}
	if opts.UseTLS {
		redisOptions.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return NewValkey(redisPinger{client: redis.NewClient(redisOptions)})
}

type redisPinger struct {
	client *redis.Client
}

func (r redisPinger) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r redisPinger) Close() error {
	return r.client.Close()
}
