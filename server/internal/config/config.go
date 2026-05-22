package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPrivateBindAddr = "127.0.0.1:8080"
	defaultPublicBindAddr  = "127.0.0.1:8081"
	defaultDataDir         = "./data"
	defaultDBPath          = "./data/safety.db"
	defaultMaxUploadBytes  = int64(250 * 1024 * 1024)
)

// Config contains the runtime settings needed by the API server.
type Config struct {
	PrivateBindAddrs []string
	PublicBindAddrs  []string
	DataDir          string
	DBPath           string
	MaxUploadBytes   int64
}

// Load reads configuration from environment variables and applies v0.2.0
// defaults for unset values.
func Load() (Config, error) {
	privateBindAddrs, err := bindAddrsFromEnv("SAFE_PRIVATE_BIND_ADDRS", "SAFE_PRIVATE_BIND_ADDR", defaultPrivateBindAddr)
	if err != nil {
		return Config{}, err
	}
	publicBindAddrs, err := bindAddrsFromEnv("SAFE_PUBLIC_BIND_ADDRS", "SAFE_PUBLIC_BIND_ADDR", defaultPublicBindAddr)
	if err != nil {
		return Config{}, err
	}

	maxUploadBytes := defaultMaxUploadBytes
	if raw := os.Getenv("SAFE_MAX_UPLOAD_BYTES"); raw != "" {
		parsed, err := parseBytes(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse SAFE_MAX_UPLOAD_BYTES: %w", err)
		}
		maxUploadBytes = parsed
	}

	return Config{
		PrivateBindAddrs: privateBindAddrs,
		PublicBindAddrs:  publicBindAddrs,
		DataDir:          envOrDefault("SAFE_DATA_DIR", defaultDataDir),
		DBPath:           envOrDefault("SAFE_DB_PATH", defaultDBPath),
		MaxUploadBytes:   maxUploadBytes,
	}, nil
}

func bindAddrsFromEnv(pluralName, singularName, fallback string) ([]string, error) {
	if raw, ok := os.LookupEnv(pluralName); ok {
		addrs, err := parseBindAddrs(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pluralName, err)
		}
		return addrs, nil
	}
	if raw, ok := os.LookupEnv(singularName); ok {
		addrs, err := parseBindAddrs(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", singularName, err)
		}
		return addrs, nil
	}
	return []string{fallback}, nil
}

func parseBindAddrs(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	addrs := make([]string, 0, len(parts))
	for index, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			return nil, fmt.Errorf("bind address list contains empty entry at position %d", index+1)
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("bind address list must contain at least one address")
	}
	return addrs, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func parseBytes(raw string) (int64, error) {
	value := strings.TrimSpace(strings.ToUpper(raw))
	if value == "" {
		return 0, fmt.Errorf("empty byte value")
	}

	units := []struct {
		suffix string
		size   int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"M", 1024 * 1024},
		{"KB", 1024},
		{"K", 1024},
		{"B", 1},
	}

	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			parsed, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0, err
			}
			if parsed <= 0 {
				return 0, fmt.Errorf("byte value must be positive")
			}
			return int64(parsed * float64(unit.size)), nil
		}
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("byte value must be positive")
	}
	return parsed, nil
}
