package config

import (
	"time"
)

const (
	defaultPrivateBindAddr   = "127.0.0.1:8080"
	defaultPublicBindAddr    = "127.0.0.1:8081"
	defaultDataDir           = "./data"
	defaultDBPath            = "./data/safety.db"
	defaultMaxUploadBytes    = int64(250 * 1024 * 1024)
	defaultEmergencyTokenTTL = 24 * time.Hour
	// Leave room for the multipart envelope added by the HTTP upload handler
	// so configured upload limits cannot overflow request-size arithmetic.
	maxConfiguredUploadBytes = int64(1<<63 - 1 - 1024*1024)

	defaultPrivateReadHeaderTimeout = 10 * time.Second
	defaultPrivateReadTimeout       = 0
	defaultPrivateWriteTimeout      = 0
	defaultPrivateIdleTimeout       = 120 * time.Second

	defaultPublicReadHeaderTimeout = 10 * time.Second
	defaultPublicReadTimeout       = 30 * time.Second
	defaultPublicWriteTimeout      = 300 * time.Second
	defaultPublicIdleTimeout       = 120 * time.Second
)

// Config contains the runtime settings needed by the API server.
type Config struct {
	PrivateBindAddrs         []string
	PublicBindAddrs          []string
	DataDir                  string
	DBPath                   string
	MaxUploadBytes           int64
	DefaultEmergencyTokenTTL time.Duration
	PrivateTimeouts          HTTPTimeouts
	PublicTimeouts           HTTPTimeouts
}

// HTTPTimeouts groups net/http server timeout settings.
type HTTPTimeouts struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// Load reads configuration from environment variables and applies defaults for
// unset values.
func Load() (Config, error) {
	privateBindAddrs, err := bindAddrsFromEnv("SAFE_PRIVATE_BIND_ADDRS", "SAFE_PRIVATE_BIND_ADDR", defaultPrivateBindAddr)
	if err != nil {
		return Config{}, err
	}
	publicBindAddrs, err := bindAddrsFromEnv("SAFE_PUBLIC_BIND_ADDRS", "SAFE_PUBLIC_BIND_ADDR", defaultPublicBindAddr)
	if err != nil {
		return Config{}, err
	}

	maxUploadBytes, err := maxUploadBytesFromEnv()
	if err != nil {
		return Config{}, err
	}
	emergencyTokenTTL, err := durationFromEnv("SAFE_DEFAULT_EMERGENCY_TOKEN_TTL", defaultEmergencyTokenTTL)
	if err != nil {
		return Config{}, err
	}

	privateTimeouts, err := privateTimeoutsFromEnv()
	if err != nil {
		return Config{}, err
	}
	publicTimeouts, err := publicTimeoutsFromEnv()
	if err != nil {
		return Config{}, err
	}

	return Config{
		PrivateBindAddrs:         privateBindAddrs,
		PublicBindAddrs:          publicBindAddrs,
		DataDir:                  envOrDefault("SAFE_DATA_DIR", defaultDataDir),
		DBPath:                   envOrDefault("SAFE_DB_PATH", defaultDBPath),
		MaxUploadBytes:           maxUploadBytes,
		DefaultEmergencyTokenTTL: emergencyTokenTTL,
		PrivateTimeouts:          privateTimeouts,
		PublicTimeouts:           publicTimeouts,
	}, nil
}
