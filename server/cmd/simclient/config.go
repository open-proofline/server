package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type config struct {
	apiBase              string
	viewerBase           string
	chunks               int
	interval             time.Duration
	mediaType            string
	chunkSize            int64
	closeIncident        bool
	completeStream       bool
	downloadBundle       bool
	encrypt              bool
	keyFile              string
	verifyBundleDecrypt  bool
	simulateFailureEvery int
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
	fs.StringVar(&chunkSizeRaw, "chunk-size", defaultChunkSize, "Size of each fake plaintext chunk before optional encryption")
	fs.BoolVar(&cfg.closeIncident, "close", false, "Close the incident when complete")
	fs.BoolVar(&cfg.completeStream, "complete-stream", true, "Mark the uploaded media stream complete")
	fs.BoolVar(&cfg.downloadBundle, "download-bundle", false, "Download the completed stream bundle through the emergency viewer")
	fs.BoolVar(&cfg.encrypt, "encrypt", true, "Encrypt simulated chunk bytes before upload")
	fs.StringVar(&cfg.keyFile, "key-file", "", "Optional simulator encryption key file")
	fs.BoolVar(&cfg.verifyBundleDecrypt, "verify-bundle-decryption", true, "Decrypt downloaded stream bundles locally when encryption is enabled")
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
	if cfg.downloadBundle && !cfg.completeStream {
		return config{}, fmt.Errorf("--download-bundle requires --complete-stream")
	}
	if cfg.downloadBundle && cfg.chunks == 0 {
		return config{}, fmt.Errorf("--download-bundle requires at least one chunk")
	}

	cfg.chunkSize = chunkSize
	cfg.apiBase = cleanBaseURL(cfg.apiBase)
	cfg.viewerBase = cleanBaseURL(cfg.viewerBase)
	return cfg, nil
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
