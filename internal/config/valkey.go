package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultValkeyDB           = 0
	defaultValkeyUseTLS       = false
	defaultValkeyDialTimeout  = 5 * time.Second
	defaultValkeyReadTimeout  = 5 * time.Second
	defaultValkeyWriteTimeout = 5 * time.Second
)

func valkeyConfigFromEnv(coordinationBackend string) (ValkeyConfig, error) {
	cfg := ValkeyConfig{
		Addr:         strings.TrimSpace(os.Getenv("SAFE_VALKEY_ADDR")),
		Username:     strings.TrimSpace(os.Getenv("SAFE_VALKEY_USERNAME")),
		Password:     strings.TrimSpace(os.Getenv("SAFE_VALKEY_PASSWORD")),
		DB:           defaultValkeyDB,
		UseTLS:       defaultValkeyUseTLS,
		DialTimeout:  defaultValkeyDialTimeout,
		ReadTimeout:  defaultValkeyReadTimeout,
		WriteTimeout: defaultValkeyWriteTimeout,
	}
	if coordinationBackend == CoordinationBackendNone {
		return cfg, nil
	}

	if cfg.Addr == "" {
		return ValkeyConfig{}, fmt.Errorf("parse SAFE_VALKEY_ADDR: required when SAFE_COORDINATION_BACKEND=%s", coordinationBackend)
	}
	if err := validateValkeyAddr(cfg.Addr); err != nil {
		return ValkeyConfig{}, err
	}

	db, err := nonNegativeIntFromEnv("SAFE_VALKEY_DB", defaultValkeyDB)
	if err != nil {
		return ValkeyConfig{}, err
	}
	useTLS, err := boolFromEnv("SAFE_VALKEY_TLS", defaultValkeyUseTLS)
	if err != nil {
		return ValkeyConfig{}, err
	}
	dialTimeout, err := durationFromEnv("SAFE_VALKEY_DIAL_TIMEOUT", defaultValkeyDialTimeout)
	if err != nil {
		return ValkeyConfig{}, err
	}
	readTimeout, err := durationFromEnv("SAFE_VALKEY_READ_TIMEOUT", defaultValkeyReadTimeout)
	if err != nil {
		return ValkeyConfig{}, err
	}
	writeTimeout, err := durationFromEnv("SAFE_VALKEY_WRITE_TIMEOUT", defaultValkeyWriteTimeout)
	if err != nil {
		return ValkeyConfig{}, err
	}

	cfg.DB = db
	cfg.UseTLS = useTLS
	cfg.DialTimeout = dialTimeout
	cfg.ReadTimeout = readTimeout
	cfg.WriteTimeout = writeTimeout
	return cfg, nil
}

func validateValkeyAddr(addr string) error {
	if strings.Contains(addr, "://") {
		return fmt.Errorf("parse SAFE_VALKEY_ADDR: expected host:port, not a URL")
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("parse SAFE_VALKEY_ADDR: expected host:port")
	}
	return nil
}
