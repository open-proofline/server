package main

import "time"

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
