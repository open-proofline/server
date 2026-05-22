# Simulator

The simulator CLI lives at `server/cmd/simclient`. It exercises the current ingest flow that a future recording client is expected to use.

## Basic Flow

Start the backend first:

```bash
cd server
go run ./cmd/api
```

Then run:

```bash
go run ./cmd/simclient --chunks 12 --interval 5s
```

The simulator prints an emergency viewer URL. Open it to watch incident metadata update.

## Bundle Download Flow

To test encrypted bundle download through the emergency viewer:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

This creates a media stream, uploads chunks with `stream_id`, completes the stream, and downloads the completed encrypted ZIP bundle through the emergency viewer.

## Failure And Retry Flow

To test hash failure and retry behavior:

```bash
go run ./cmd/simclient --chunks 12 --interval 2s --simulate-failure-every 4
```

Every fourth chunk intentionally fails SHA-256 verification before being retried.

## Useful Flags

| Flag | Purpose |
|---|---|
| `--api` | Private API base URL. |
| `--viewer` | Emergency viewer base URL. |
| `--chunks` | Number of chunks to upload. |
| `--interval` | Delay between chunk uploads. |
| `--chunk-size` | Size of each fake encrypted chunk. |
| `--media-type` | Media type to upload. |
| `--complete-stream` | Mark the uploaded media stream complete. |
| `--download-bundle` | Download the completed stream bundle through the emergency viewer. |
| `--simulate-failure-every` | Intentionally fail every Nth chunk hash before retrying. |
| `--close` | Close the incident when complete. |
