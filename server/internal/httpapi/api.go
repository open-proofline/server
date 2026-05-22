package httpapi

import (
	"log/slog"
	"net/http"

	"safety-recorder/server/internal/incidents"
	"safety-recorder/server/internal/storage"
)

const (
	defaultMaxUploadBytes = int64(250 * 1024 * 1024)
	jsonBodyLimit         = int64(64 * 1024)
	fieldLimit            = int64(64 * 1024)
	multipartOverhead     = int64(1024 * 1024)
)

// Options configures API construction.
type Options struct {
	MaxUploadBytes int64
	Logger         *slog.Logger
}

// API holds the dependencies and limits used by the HTTP handlers.
type API struct {
	repo           *incidents.Repository
	store          *storage.Store
	maxUploadBytes int64
	logger         *slog.Logger
}

// New builds the private HTTP handler. Prefer NewPrivate or NewPublic at call
// sites that need to make the routing boundary explicit.
func New(repo *incidents.Repository, store *storage.Store, opts Options) http.Handler {
	return NewPrivate(repo, store, opts)
}

// NewPrivate builds the HTTP handler tree for the private v0.1 write/admin API.
func NewPrivate(repo *incidents.Repository, store *storage.Store, opts Options) http.Handler {
	return newAPI(repo, store, opts).privateRoutes()
}

// NewPublic builds the HTTP handler tree for the public read-only emergency
// viewer.
func NewPublic(repo *incidents.Repository, store *storage.Store, opts Options) http.Handler {
	return newAPI(repo, store, opts).publicRoutes()
}

func newAPI(repo *incidents.Repository, store *storage.Store, opts Options) *API {
	maxUploadBytes := opts.MaxUploadBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = defaultMaxUploadBytes
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &API{
		repo:           repo,
		store:          store,
		maxUploadBytes: maxUploadBytes,
		logger:         logger,
	}
}
