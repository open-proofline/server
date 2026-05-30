package config

import (
	"time"
)

const (
	defaultPrivateBindAddr  = "127.0.0.1:8080"
	defaultPublicBindAddr   = "127.0.0.1:8081"
	defaultDataDir          = "./data"
	defaultDBPath           = "./data/safety.db"
	defaultMaxUploadBytes   = int64(250 * 1024 * 1024)
	defaultIncidentTokenTTL = 24 * time.Hour
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
	PrivateBindAddrs        []string
	PublicBindAddrs         []string
	Backends                BackendSelection
	Postgres                PostgresConfig
	S3Blob                  S3BlobConfig
	Valkey                  ValkeyConfig
	DataDir                 string
	DBPath                  string
	MaxUploadBytes          int64
	DefaultIncidentTokenTTL time.Duration
	PrivateTimeouts         HTTPTimeouts
	PublicTimeouts          HTTPTimeouts
}

// BackendSelection records the configured storage and coordination backends.
type BackendSelection struct {
	Metadata     string
	Blob         string
	Coordination string
}

// S3BlobConfig contains the optional S3-compatible blob backend settings.
type S3BlobConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	Prefix          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	ForcePathStyle  bool
}

// ValkeyConfig contains optional Valkey/Redis-compatible coordination settings.
type ValkeyConfig struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	UseTLS       bool
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// PostgresConfig contains optional PostgreSQL metadata backend settings.
type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
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

	backends, err := backendSelectionFromEnv()
	if err != nil {
		return Config{}, err
	}
	postgres, err := postgresConfigFromEnv(backends.Metadata)
	if err != nil {
		return Config{}, err
	}
	s3Blob, err := s3BlobConfigFromEnv(backends.Blob)
	if err != nil {
		return Config{}, err
	}
	valkey, err := valkeyConfigFromEnv(backends.Coordination)
	if err != nil {
		return Config{}, err
	}

	maxUploadBytes, err := maxUploadBytesFromEnv()
	if err != nil {
		return Config{}, err
	}
	incidentTokenTTL, err := durationFromEnv("SAFE_DEFAULT_INCIDENT_TOKEN_TTL", defaultIncidentTokenTTL)
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
		PrivateBindAddrs:        privateBindAddrs,
		PublicBindAddrs:         publicBindAddrs,
		Backends:                backends,
		Postgres:                postgres,
		S3Blob:                  s3Blob,
		Valkey:                  valkey,
		DataDir:                 envOrDefault("SAFE_DATA_DIR", defaultDataDir),
		DBPath:                  envOrDefault("SAFE_DB_PATH", defaultDBPath),
		MaxUploadBytes:          maxUploadBytes,
		DefaultIncidentTokenTTL: incidentTokenTTL,
		PrivateTimeouts:         privateTimeouts,
		PublicTimeouts:          publicTimeouts,
	}, nil
}
