package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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

func (a *API) privateRoutes() http.Handler {
	mux := http.NewServeMux()
	// Request flow:
	// 1. create an incident;
	// 2. upload encrypted chunks for that incident;
	// 3. stream each upload to temp storage while hashing;
	// 4. verify the client-provided SHA-256;
	// 5. commit the blob to an immutable final path;
	// 6. insert chunk metadata only after the file is safely written.
	mux.HandleFunc("POST /v1/incidents", a.createIncident)
	mux.HandleFunc("GET /v1/incidents/{incident_id}", a.getIncident)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/chunks", a.uploadChunk)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/chunks", a.listChunks)
	mux.HandleFunc("GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}", a.getChunkBytes)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/checkins", a.createCheckin)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/close", a.closeIncident)
	mux.HandleFunc("POST /v1/incidents/{incident_id}/emergency-tokens", a.createEmergencyToken)
	mux.HandleFunc("POST /v1/emergency-tokens/{token_id}/revoke", a.revokeEmergencyToken)
	mux.HandleFunc("/", a.notFound)

	// v0.1 has no public authentication by design. Deployment must provide the
	// private boundary, for example localhost, WireGuard, or firewall rules.
	return a.loggingMiddleware(a.recoveryMiddleware(mux))
}

func (a *API) publicRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /e/{token}", a.emergencyPage)
	mux.HandleFunc("GET /e/{token}/data", a.emergencyData)
	// Static emergency assets are embedded and token-neutral; the token stays
	// in the request path handled above.
	mux.Handle("GET /static/", emergencyStaticHandler())
	mux.HandleFunc("/", a.notFound)

	return a.loggingMiddleware(a.recoveryMiddleware(mux))
}

func (a *API) createIncident(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ClientLabel string `json:"client_label"`
		Notes       string `json:"notes"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	// Creating an incident is the first step in the client upload flow. New
	// incidents start open so encrypted chunks can be attached immediately.
	incident, err := a.repo.CreateIncident(r.Context(), request.ClientLabel, request.Notes)
	if err != nil {
		a.internalError(w, "create incident", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"incident_id": incident.ID,
		"status":      incident.Status,
	})
}

func (a *API) getIncident(w http.ResponseWriter, r *http.Request) {
	detail, err := a.repo.GetIncidentDetail(r.Context(), r.PathValue("incident_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (a *API) uploadChunk(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	incident, err := a.repo.GetIncident(r.Context(), incidentID)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get incident", err)
		return
	}
	if incident.Status == incidents.StatusClosed {
		writeError(w, http.StatusConflict, "incident_closed", "incident is closed")
		return
	}

	// readChunkUpload enforces request size limits and stages the file in temp
	// storage while computing its SHA-256 hash.
	upload, ok := a.readChunkUpload(w, r)
	if !ok {
		return
	}
	defer upload.temp.Cleanup()

	// The backend assumes encryption already happened on the client; this hash
	// check is only an integrity check for the encrypted bytes in transit.
	if upload.temp.SHA256Hex != upload.sha256Hex {
		writeError(w, http.StatusBadRequest, "hash_mismatch", "computed SHA-256 did not match provided hash")
		return
	}

	// Fast duplicate rejection avoids unnecessary final-file work in the common
	// case. SQLite still enforces uniqueness during metadata insert below.
	exists, err := a.repo.ChunkExists(r.Context(), incidentID, upload.mediaType, upload.chunkIndex)
	if err != nil {
		a.internalError(w, "check duplicate chunk", err)
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "duplicate_chunk", "chunk_index already exists for this incident and media type")
		return
	}

	// CommitTemp places the verified file at its final immutable path and fails
	// rather than overwriting an existing chunk file.
	storedPath, err := a.store.CommitTemp(upload.temp, incidentID, upload.mediaType, upload.chunkIndex)
	if errors.Is(err, storage.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "duplicate_chunk", "stored chunk already exists for this incident and media type")
		return
	}
	if err != nil {
		a.internalError(w, "commit upload", err)
		return
	}

	// Metadata is inserted only after the verified blob has been committed. If
	// SQLite rejects the insert, the just-written blob is removed to avoid an
	// orphaned chunk.
	chunk, err := a.repo.CreateChunk(r.Context(), incidents.CreateChunkParams{
		IncidentID:       incidentID,
		ChunkIndex:       upload.chunkIndex,
		MediaType:        upload.mediaType,
		StartedAt:        upload.startedAt,
		EndedAt:          upload.endedAt,
		OriginalFilename: upload.originalFilename,
		StoredPath:       storedPath,
		ByteSize:         upload.temp.ByteSize,
		SHA256Hex:        upload.sha256Hex,
	})
	if errors.Is(err, incidents.ErrDuplicate) {
		_ = a.store.Remove(storedPath)
		writeError(w, http.StatusConflict, "duplicate_chunk", "chunk_index already exists for this incident and media type")
		return
	}
	if err != nil {
		_ = a.store.Remove(storedPath)
		a.internalError(w, "insert chunk metadata", err)
		return
	}

	writeJSON(w, http.StatusCreated, chunk)
}

func (a *API) listChunks(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if _, err := a.repo.GetIncident(r.Context(), incidentID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	} else if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	chunks, err := a.repo.ListChunks(r.Context(), incidentID)
	if err != nil {
		a.internalError(w, "list chunks", err)
		return
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (a *API) getChunkBytes(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	mediaType := r.PathValue("media_type")
	if !incidents.ValidMediaType(mediaType) {
		writeError(w, http.StatusBadRequest, "invalid_media_type", "media_type must be audio, video, location, or metadata")
		return
	}
	chunkIndex, err := parseChunkIndex(r.PathValue("chunk_index"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_chunk_index", "chunk_index must be a non-negative integer")
		return
	}

	chunk, err := a.repo.GetChunkByKey(r.Context(), incidentID, mediaType, chunkIndex)
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "chunk_not_found", "chunk was not found")
		return
	}
	if err != nil {
		a.internalError(w, "get chunk", err)
		return
	}

	// This returns encrypted bytes only and is intended for private/dev testing,
	// not a public emergency viewer.
	file, err := a.store.Open(chunk.StoredPath)
	if errors.Is(err, os.ErrNotExist) {
		a.internalError(w, "open chunk bytes", fmt.Errorf("metadata exists but file is missing: %w", err))
		return
	}
	if err != nil {
		a.internalError(w, "open chunk bytes", err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(chunk.ByteSize, 10))
	http.ServeContent(w, r, path.Base(chunk.StoredPath), chunk.CreatedAt, file)
}

func (a *API) createCheckin(w http.ResponseWriter, r *http.Request) {
	incidentID := r.PathValue("incident_id")
	if _, err := a.repo.GetIncident(r.Context(), incidentID); errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	} else if err != nil {
		a.internalError(w, "get incident", err)
		return
	}

	var request struct {
		DeviceBatteryPercent *int     `json:"device_battery_percent"`
		DeviceNetwork        *string  `json:"device_network"`
		Latitude             *float64 `json:"latitude"`
		Longitude            *float64 `json:"longitude"`
		AccuracyMeters       *float64 `json:"accuracy_meters"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	checkin, err := a.repo.CreateCheckin(r.Context(), incidentID, incidents.CreateCheckinParams{
		DeviceBatteryPercent: request.DeviceBatteryPercent,
		DeviceNetwork:        request.DeviceNetwork,
		Latitude:             request.Latitude,
		Longitude:            request.Longitude,
		AccuracyMeters:       request.AccuracyMeters,
	})
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "create checkin", err)
		return
	}

	writeJSON(w, http.StatusCreated, checkin)
}

func (a *API) closeIncident(w http.ResponseWriter, r *http.Request) {
	incident, err := a.repo.CloseIncident(r.Context(), r.PathValue("incident_id"))
	if errors.Is(err, incidents.ErrNotFound) {
		writeError(w, http.StatusNotFound, "incident_not_found", "incident was not found")
		return
	}
	if err != nil {
		a.internalError(w, "close incident", err)
		return
	}
	writeJSON(w, http.StatusOK, incident)
}

func (a *API) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "endpoint was not found")
}

type chunkUpload struct {
	temp             *storage.TempUpload
	chunkIndex       int
	mediaType        string
	startedAt        time.Time
	endedAt          time.Time
	sha256Hex        string
	originalFilename string
}

func (a *API) readChunkUpload(w http.ResponseWriter, r *http.Request) (chunkUpload, bool) {
	// The multipart reader adds overhead around the file bytes, so the request
	// cap allows a small fixed envelope while SaveTemp enforces the exact file
	// byte limit.
	r.Body = http.MaxBytesReader(w, r.Body, a.maxUploadBytes+multipartOverhead)

	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_multipart", "request must be multipart/form-data")
		return chunkUpload{}, false
	}

	fields := make(map[string]string)
	var temp *storage.TempUpload
	var partFilename string

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if isMaxBytesError(err) {
			if temp != nil {
				temp.Cleanup()
			}
			writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "upload exceeded SAFE_MAX_UPLOAD_BYTES")
			return chunkUpload{}, false
		}
		if err != nil {
			if temp != nil {
				temp.Cleanup()
			}
			writeError(w, http.StatusBadRequest, "invalid_multipart", "could not read multipart request")
			return chunkUpload{}, false
		}

		formName := part.FormName()
		if formName == "file" {
			if temp != nil {
				temp.Cleanup()
				writeError(w, http.StatusBadRequest, "duplicate_file", "only one file field is allowed")
				return chunkUpload{}, false
			}
			partFilename = cleanFilename(part.FileName())
			// SaveTemp streams the file part straight to disk and hashes it as it
			// reads, so large uploads are never buffered in memory.
			temp, err = a.store.SaveTemp(part, a.maxUploadBytes)
			if errors.Is(err, storage.ErrTooLarge) || isMaxBytesError(err) {
				writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "upload exceeded SAFE_MAX_UPLOAD_BYTES")
				return chunkUpload{}, false
			}
			if err != nil {
				a.internalError(w, "save temp upload", err)
				return chunkUpload{}, false
			}
			continue
		}

		if formName == "" {
			continue
		}
		value, err := readField(part)
		if isMaxBytesError(err) {
			if temp != nil {
				temp.Cleanup()
			}
			writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "upload exceeded SAFE_MAX_UPLOAD_BYTES")
			return chunkUpload{}, false
		}
		if err != nil {
			if temp != nil {
				temp.Cleanup()
			}
			writeError(w, http.StatusBadRequest, "invalid_field", "multipart field was too large or unreadable")
			return chunkUpload{}, false
		}
		fields[formName] = value
	}

	if temp == nil {
		writeError(w, http.StatusBadRequest, "missing_file", "file field is required")
		return chunkUpload{}, false
	}

	parsed, ok := parseChunkFields(w, fields, partFilename)
	if !ok {
		temp.Cleanup()
		return chunkUpload{}, false
	}
	parsed.temp = temp
	return parsed, true
}

func parseChunkFields(w http.ResponseWriter, fields map[string]string, partFilename string) (chunkUpload, bool) {
	chunkIndex, err := parseChunkIndex(requiredField(fields, "chunk_index"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_chunk_index", "chunk_index must be a non-negative integer")
		return chunkUpload{}, false
	}

	mediaType := requiredField(fields, "media_type")
	if !incidents.ValidMediaType(mediaType) {
		writeError(w, http.StatusBadRequest, "invalid_media_type", "media_type must be audio, video, location, or metadata")
		return chunkUpload{}, false
	}

	startedAt, err := time.Parse(time.RFC3339Nano, requiredField(fields, "started_at"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_started_at", "started_at must be an RFC3339 timestamp")
		return chunkUpload{}, false
	}
	endedAt, err := time.Parse(time.RFC3339Nano, requiredField(fields, "ended_at"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_ended_at", "ended_at must be an RFC3339 timestamp")
		return chunkUpload{}, false
	}
	if endedAt.Before(startedAt) {
		writeError(w, http.StatusBadRequest, "invalid_time_range", "ended_at must be after started_at")
		return chunkUpload{}, false
	}

	sha256Hex := requiredField(fields, "sha256_hex")
	if !validSHA256Hex(sha256Hex) {
		writeError(w, http.StatusBadRequest, "invalid_sha256_hex", "sha256_hex must be lowercase SHA-256 hex")
		return chunkUpload{}, false
	}

	originalFilename := cleanFilename(fields["original_filename"])
	if originalFilename == "" {
		originalFilename = partFilename
	}

	return chunkUpload{
		chunkIndex:       chunkIndex,
		mediaType:        mediaType,
		startedAt:        startedAt.UTC(),
		endedAt:          endedAt.UTC(),
		sha256Hex:        sha256Hex,
		originalFilename: originalFilename,
	}, true
}

func requiredField(fields map[string]string, name string) string {
	return strings.TrimSpace(fields[name])
}

func readField(part *multipart.Part) (string, error) {
	var buffer bytes.Buffer
	n, err := io.Copy(&buffer, io.LimitReader(part, fieldLimit+1))
	if err != nil {
		return "", err
	}
	if n > fieldLimit {
		return "", fmt.Errorf("field too large")
	}
	return buffer.String(), nil
}

func parseChunkIndex(raw string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("missing chunk_index")
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, fmt.Errorf("chunk_index must be non-negative")
	}
	return int(value), nil
}

func validSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}

func cleanFilename(value string) string {
	cleaned := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "" {
		return ""
	}
	base := path.Base(cleaned)
	if base == "." || base == "/" {
		return ""
	}
	return base
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	// Non-upload JSON bodies are intentionally small metadata requests.
	r.Body = http.MaxBytesReader(w, r.Body, jsonBodyLimit)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		if isMaxBytesError(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "JSON request body is too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain a single JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]map[string]string{
		"error": {
			"code":    code,
			"message": message,
		},
	})
}

func (a *API) internalError(w http.ResponseWriter, operation string, err error) {
	a.logger.Error("internal error", "operation", operation, "err", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader records the response status before forwarding it to the client.
func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write records response size for access logs without inspecting response
// contents.
func (r *statusRecorder) Write(bytes []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(bytes)
	r.bytes += n
	return n, err
}

func (a *API) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		// Log routing metadata only. Bodies, upload bytes, Authorization headers,
		// and any future token-like values are deliberately omitted.
		a.logger.Info("request",
			"method", r.Method,
			"path", safeLogPath(r),
			"status", status,
			"bytes", recorder.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func safeLogPath(r *http.Request) string {
	if r.Pattern != "" && r.Pattern != "/" {
		return r.Pattern
	}
	if strings.HasPrefix(r.URL.Path, "/e/") {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return "/e/{token}/data"
		}
		return "/e/{token}"
	}
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.URL.Path
}

func (a *API) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "err", recovered)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
