package config

import (
	"os"
	"strings"
)

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func secretFromEnv(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}
