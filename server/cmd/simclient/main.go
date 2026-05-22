package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIBase       = "http://localhost:8080"
	defaultViewerBase    = "http://localhost:8081"
	defaultChunks        = 12
	defaultInterval      = 5 * time.Second
	defaultMediaType     = "audio"
	defaultChunkSize     = "64KiB"
	defaultCheckinEvery  = 3
	clientRequestTimeout = 30 * time.Second
	chunkDuration        = 10 * time.Second
)

type config struct {
	apiBase              string
	viewerBase           string
	chunks               int
	interval             time.Duration
	mediaType            string
	chunkSize            int64
	closeIncident        bool
	simulateFailureEvery int
}

type client struct {
	httpClient *http.Client
	apiBase    string
	viewerBase string
}

type createIncidentResponse struct {
	IncidentID string `json:"incident_id"`
	Status     string `json:"status"`
}

type createEmergencyTokenResponse struct {
	TokenID    string     `json:"token_id"`
	IncidentID string     `json:"incident_id"`
	Token      string     `json:"token"`
	Label      string     `json:"label,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type apiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type chunkUpload struct {
	incidentID string
	chunkIndex int
	mediaType  string
	startedAt  time.Time
	endedAt    time.Time
	filename   string
	body       []byte
	sha256Hex  string
}

func main() {
	if err := run(context.Background(), os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "simclient: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}

	sim := client{
		httpClient: &http.Client{Timeout: clientRequestTimeout},
		apiBase:    cfg.apiBase,
		viewerBase: cfg.viewerBase,
	}

	fmt.Fprintln(out, "Creating incident...")
	incidentID, err := sim.createIncident(ctx)
	if err != nil {
		return err
	}

	token, err := sim.createEmergencyToken(ctx, incidentID)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Incident: %s\n", incidentID)
	fmt.Fprintf(out, "Emergency viewer: %s\n\n", buildViewerURL(sim.viewerBase, token))

	startedAt := time.Now().UTC()
	for i := 1; i <= cfg.chunks; i++ {
		chunk, err := newChunkUpload(incidentID, i, cfg.mediaType, cfg.chunkSize, startedAt)
		if err != nil {
			return err
		}

		if shouldSimulateFailure(i, cfg.simulateFailureEvery) {
			fmt.Fprintf(out, "Uploading %s chunk %d/%d with intentionally bad hash...\n", cfg.mediaType, i, cfg.chunks)
			failed := chunk
			failed.sha256Hex = badHashFor(chunk.sha256Hex)
			if err := sim.expectHashMismatch(ctx, failed); err != nil {
				return err
			}
			fmt.Fprintln(out, "Server rejected chunk as expected.")

			fmt.Fprintf(out, "Retrying %s chunk %d/%d with correct hash...\n", cfg.mediaType, i, cfg.chunks)
			if err := sim.uploadChunk(ctx, chunk); err != nil {
				return err
			}
			fmt.Fprintln(out, "Retry succeeded.")
		} else {
			fmt.Fprintf(out, "Uploading %s chunk %d/%d...\n", cfg.mediaType, i, cfg.chunks)
			if err := sim.uploadChunk(ctx, chunk); err != nil {
				return err
			}
		}

		if shouldSendCheckin(i) {
			fmt.Fprintln(out, "Sending checkin...")
			if err := sim.createCheckin(ctx, incidentID, i); err != nil {
				return err
			}
		}

		if i < cfg.chunks && cfg.interval > 0 {
			time.Sleep(cfg.interval)
		}
	}

	if cfg.closeIncident {
		fmt.Fprintln(out, "Closing incident...")
		if err := sim.closeIncident(ctx, incidentID); err != nil {
			return err
		}
		fmt.Fprintln(out, "Incident closed.")
	}

	fmt.Fprintln(out, "Done.")
	return nil
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("simclient", flag.ContinueOnError)

	var chunkSizeRaw string
	cfg := config{}
	fs.StringVar(&cfg.apiBase, "api", defaultAPIBase, "Private API base URL")
	fs.StringVar(&cfg.viewerBase, "viewer", defaultViewerBase, "Emergency viewer base URL")
	fs.IntVar(&cfg.chunks, "chunks", defaultChunks, "Number of chunks to upload")
	fs.DurationVar(&cfg.interval, "interval", defaultInterval, "Delay between chunk uploads")
	fs.StringVar(&cfg.mediaType, "media-type", defaultMediaType, "Media type to upload")
	fs.StringVar(&chunkSizeRaw, "chunk-size", defaultChunkSize, "Size of each fake encrypted chunk")
	fs.BoolVar(&cfg.closeIncident, "close", false, "Close the incident when complete")
	fs.IntVar(&cfg.simulateFailureEvery, "simulate-failure-every", 0, "Every Nth chunk should intentionally fail hash verification before retrying")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if cfg.chunks < 0 {
		return config{}, fmt.Errorf("--chunks must be non-negative")
	}
	if cfg.interval < 0 {
		return config{}, fmt.Errorf("--interval must be non-negative")
	}
	if !validMediaType(cfg.mediaType) {
		return config{}, fmt.Errorf("--media-type must be audio, video, location, or metadata")
	}
	chunkSize, err := parseByteSize(chunkSizeRaw)
	if err != nil {
		return config{}, fmt.Errorf("--chunk-size: %w", err)
	}
	if chunkSize <= 0 {
		return config{}, fmt.Errorf("--chunk-size must be greater than zero")
	}
	if cfg.simulateFailureEvery < 0 {
		return config{}, fmt.Errorf("--simulate-failure-every must be non-negative")
	}

	cfg.chunkSize = chunkSize
	cfg.apiBase = cleanBaseURL(cfg.apiBase)
	cfg.viewerBase = cleanBaseURL(cfg.viewerBase)
	return cfg, nil
}

func (c client) createIncident(ctx context.Context) (string, error) {
	request := map[string]string{
		"client_label": "simclient",
		"notes":        "simulated incident",
	}
	var response createIncidentResponse
	if err := c.postJSON(ctx, "/v1/incidents", request, http.StatusCreated, &response); err != nil {
		return "", fmt.Errorf("create incident: %w", err)
	}
	if response.IncidentID == "" {
		return "", fmt.Errorf("create incident: empty incident_id in response")
	}
	return response.IncidentID, nil
}

func (c client) createEmergencyToken(ctx context.Context, incidentID string) (string, error) {
	request := map[string]string{"label": "simclient"}
	var response createEmergencyTokenResponse
	path := "/v1/incidents/" + url.PathEscape(incidentID) + "/emergency-tokens"
	if err := c.postJSON(ctx, path, request, http.StatusCreated, &response); err != nil {
		return "", fmt.Errorf("create emergency token: %w", err)
	}
	if response.Token == "" {
		return "", fmt.Errorf("create emergency token: empty token in response")
	}
	return response.Token, nil
}

func (c client) createCheckin(ctx context.Context, incidentID string, chunkIndex int) error {
	battery := 100 - chunkIndex
	if battery < 1 {
		battery = 1
	}
	request := map[string]any{
		"device_battery_percent": battery,
		"device_network":         "simulated",
		"latitude":               37.7749,
		"longitude":              -122.4194,
		"accuracy_meters":        15,
	}
	path := "/v1/incidents/" + url.PathEscape(incidentID) + "/checkins"
	if err := c.postJSON(ctx, path, request, http.StatusCreated, nil); err != nil {
		return fmt.Errorf("create checkin: %w", err)
	}
	return nil
}

func (c client) closeIncident(ctx context.Context, incidentID string) error {
	path := "/v1/incidents/" + url.PathEscape(incidentID) + "/close"
	if err := c.postJSON(ctx, path, map[string]any{}, http.StatusOK, nil); err != nil {
		return fmt.Errorf("close incident: %w", err)
	}
	return nil
}

func (c client) uploadChunk(ctx context.Context, upload chunkUpload) error {
	status, body, err := c.postChunk(ctx, upload)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("upload chunk: expected status %d, got %d: %s", http.StatusCreated, status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c client) expectHashMismatch(ctx context.Context, upload chunkUpload) error {
	status, body, err := c.postChunk(ctx, upload)
	if err != nil {
		return err
	}
	if status != http.StatusBadRequest {
		return fmt.Errorf("expected hash mismatch status %d, got %d: %s", http.StatusBadRequest, status, strings.TrimSpace(string(body)))
	}
	code := errorCode(body)
	if code != "hash_mismatch" {
		return fmt.Errorf("expected hash_mismatch error code, got %q: %s", code, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c client) postJSON(ctx context.Context, path string, payload any, wantStatus int, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(c.apiBase, path), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return err
	}
	if response.StatusCode != wantStatus {
		return fmt.Errorf("expected status %d, got %d: %s", wantStatus, response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c client) postChunk(ctx context.Context, upload chunkUpload) (int, []byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("file", upload.filename)
	if err != nil {
		return 0, nil, err
	}
	if _, err := filePart.Write(upload.body); err != nil {
		return 0, nil, err
	}
	fields := map[string]string{
		"chunk_index":       strconv.Itoa(upload.chunkIndex),
		"media_type":        upload.mediaType,
		"started_at":        upload.startedAt.Format(time.RFC3339Nano),
		"ended_at":          upload.endedAt.Format(time.RFC3339Nano),
		"sha256_hex":        upload.sha256Hex,
		"original_filename": upload.filename,
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return 0, nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return 0, nil, err
	}

	path := "/v1/incidents/" + url.PathEscape(upload.incidentID) + "/chunks"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(c.apiBase, path), &body)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return response.StatusCode, nil, err
	}
	return response.StatusCode, responseBody, nil
}

func newChunkUpload(incidentID string, chunkIndex int, mediaType string, size int64, startedAt time.Time) (chunkUpload, error) {
	if size > int64(int(^uint(0)>>1)) {
		return chunkUpload{}, fmt.Errorf("chunk size is too large for this platform")
	}
	body := make([]byte, int(size))
	if _, err := rand.Read(body); err != nil {
		return chunkUpload{}, fmt.Errorf("generate fake encrypted bytes: %w", err)
	}
	sum := sha256.Sum256(body)
	chunkStartedAt := startedAt.Add(time.Duration(chunkIndex-1) * chunkDuration)
	return chunkUpload{
		incidentID: incidentID,
		chunkIndex: chunkIndex,
		mediaType:  mediaType,
		startedAt:  chunkStartedAt,
		endedAt:    chunkStartedAt.Add(chunkDuration),
		filename:   fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex),
		body:       body,
		sha256Hex:  hex.EncodeToString(sum[:]),
	}, nil
}

func parseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("empty size")
	}

	digitsEnd := 0
	for digitsEnd < len(value) && value[digitsEnd] >= '0' && value[digitsEnd] <= '9' {
		digitsEnd++
	}
	if digitsEnd == 0 {
		return 0, fmt.Errorf("missing numeric value")
	}

	amount, err := strconv.ParseInt(value[:digitsEnd], 10, 64)
	if err != nil {
		return 0, err
	}
	unit := strings.ToLower(strings.TrimSpace(value[digitsEnd:]))
	multiplier, ok := byteSizeMultipliers()[unit]
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q", value[digitsEnd:])
	}
	if amount > 0 && multiplier > 0 && amount > (1<<63-1)/multiplier {
		return 0, fmt.Errorf("size overflows int64")
	}
	return amount * multiplier, nil
}

func byteSizeMultipliers() map[string]int64 {
	return map[string]int64{
		"":    1,
		"b":   1,
		"k":   1000,
		"kb":  1000,
		"kib": 1024,
		"m":   1000 * 1000,
		"mb":  1000 * 1000,
		"mib": 1024 * 1024,
		"g":   1000 * 1000 * 1000,
		"gb":  1000 * 1000 * 1000,
		"gib": 1024 * 1024 * 1024,
	}
}

func buildViewerURL(viewerBase, token string) string {
	return joinURL(cleanBaseURL(viewerBase), "/e/"+url.PathEscape(token))
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func cleanBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func shouldSimulateFailure(chunkIndex, every int) bool {
	return every > 0 && chunkIndex%every == 0
}

func shouldSendCheckin(chunkIndex int) bool {
	return chunkIndex == 1 || chunkIndex%defaultCheckinEvery == 0
}

func badHashFor(hash string) string {
	if len(hash) != 64 {
		return strings.Repeat("0", 64)
	}
	if hash[0] == '0' {
		return "1" + hash[1:]
	}
	return "0" + hash[1:]
}

func errorCode(body []byte) string {
	var apiError apiErrorResponse
	if err := json.Unmarshal(body, &apiError); err != nil {
		return ""
	}
	return apiError.Error.Code
}

func validMediaType(mediaType string) bool {
	switch mediaType {
	case "audio", "video", "location", "metadata":
		return true
	default:
		return false
	}
}
