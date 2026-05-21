package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultBindAddr       = ":8080"
	defaultDataDir        = "./data"
	defaultDBPath         = "./data/safety.db"
	defaultMaxUploadBytes = int64(250 * 1024 * 1024)
)

type Config struct {
	BindAddr       string
	DataDir        string
	DBPath         string
	MaxUploadBytes int64
}

func Load() (Config, error) {
	maxUploadBytes := defaultMaxUploadBytes
	if raw := os.Getenv("SAFE_MAX_UPLOAD_BYTES"); raw != "" {
		parsed, err := parseBytes(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse SAFE_MAX_UPLOAD_BYTES: %w", err)
		}
		maxUploadBytes = parsed
	}

	return Config{
		BindAddr:       envOrDefault("SAFE_BIND_ADDR", defaultBindAddr),
		DataDir:        envOrDefault("SAFE_DATA_DIR", defaultDataDir),
		DBPath:         envOrDefault("SAFE_DB_PATH", defaultDBPath),
		MaxUploadBytes: maxUploadBytes,
	}, nil
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
