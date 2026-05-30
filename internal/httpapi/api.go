package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/open-proofline/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultMaxUploadBytes   = int64(250 * 1024 * 1024)
	defaultIncidentTokenTTL = 24 * time.Hour
	defaultSessionTTL       = 12 * time.Hour
	jsonBodyLimit           = int64(64 * 1024)
	fieldLimit              = int64(64 * 1024)
	multipartOverhead       = int64(1024 * 1024)
	maxSafeUploadBytes      = int64(1<<63 - 1 - multipartOverhead)
)

// Options configures API construction.
type Options struct {
	MaxUploadBytes          int64
	DefaultIncidentTokenTTL *time.Duration
	SessionTTL              time.Duration
	BootstrapSecret         string
	PasswordCost            int
	Logger                  *slog.Logger
}

// API holds the dependencies and limits used by the HTTP handlers.
type API struct {
	repo                    MetadataRepository
	store                   storage.BlobStore
	maxUploadBytes          int64
	defaultIncidentTokenTTL time.Duration
	sessionTTL              time.Duration
	bootstrapSecret         string
	passwordCost            int
	logger                  *slog.Logger
}

// New builds the private HTTP handler. Prefer NewPrivate or NewPublic at call
// sites that need to make the routing boundary explicit.
func New(repo MetadataRepository, store storage.BlobStore, opts Options) http.Handler {
	return NewPrivate(repo, store, opts)
}

// NewPrivate builds the HTTP handler tree for the private write/admin API.
func NewPrivate(repo MetadataRepository, store storage.BlobStore, opts Options) http.Handler {
	return newAPI(repo, store, opts).privateRoutes()
}

// NewPublic builds the HTTP handler tree for the public read-only incident
// viewer.
func NewPublic(repo MetadataRepository, store storage.BlobStore, opts Options) http.Handler {
	return newAPI(repo, store, opts).publicRoutes()
}

func newAPI(repo MetadataRepository, store storage.BlobStore, opts Options) *API {
	maxUploadBytes := opts.MaxUploadBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = defaultMaxUploadBytes
	}
	if maxUploadBytes > maxSafeUploadBytes {
		maxUploadBytes = maxSafeUploadBytes
	}
	incidentTokenTTL := defaultIncidentTokenTTL
	if opts.DefaultIncidentTokenTTL != nil {
		incidentTokenTTL = *opts.DefaultIncidentTokenTTL
	}
	if incidentTokenTTL < 0 {
		incidentTokenTTL = 0
	}
	sessionTTL := opts.SessionTTL
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}
	passwordCost := opts.PasswordCost
	if passwordCost == 0 {
		passwordCost = bcrypt.DefaultCost
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &API{
		repo:                    repo,
		store:                   store,
		maxUploadBytes:          maxUploadBytes,
		defaultIncidentTokenTTL: incidentTokenTTL,
		sessionTTL:              sessionTTL,
		bootstrapSecret:         opts.BootstrapSecret,
		passwordCost:            passwordCost,
		logger:                  logger,
	}
}
