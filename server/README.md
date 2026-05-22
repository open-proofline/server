# Safety Recorder Server

This directory contains the Go backend for Safety Recorder.

From this directory:

```bash
go run ./cmd/api
go test ./...
```

The API starts private `/v1` listeners and public emergency viewer listeners from the configured bind address lists. Keep `/v1` behind localhost, WireGuard, a firewall, or a strict reverse proxy.

The simulator exercises the current ingest flow:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

That flow creates an incident, creates an emergency token, creates a media stream, uploads encrypted chunks with `stream_id`, completes the stream, and optionally downloads the encrypted ZIP bundle through the emergency viewer.

Evidence bundles contain encrypted chunk files and JSON manifests only. They are not decrypted, merged, or playable media exports.

See the repository root `README.md`, `docs/api.md`, and `docs/code-map.md` for the full route and architecture notes.
