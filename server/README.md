# Proofline Server

This directory contains the Go backend for Proofline.

This repository is intended to become `open-proofline/server` in the planned multi-repo layout. It is server/backend only; web-client, iOS-client, Android-client, and protocol implementation should live in separate future repositories.

Repository, module, Docker image, and GHCR artifact names may still use `safety-recorder` until an explicit migration is performed.

From this directory:

```bash
go run ./cmd/api
go test ./...
```

The API starts private `/v1` listeners and public incident viewer listeners from the configured bind address lists. Keep `/v1` behind localhost, WireGuard, a firewall, or a strict reverse proxy.

The simulator exercises the current generic incident ingest flow:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

That flow creates an incident, creates a viewer token through the current emergency-token route, creates a media stream, encrypts and uploads chunks with `stream_id`, completes the stream, downloads the encrypted ZIP bundle through the incident viewer, and verifies local decryption when bundle download is enabled.

Evidence bundles contain encrypted chunk files and JSON manifests only. They are not decrypted, merged, or playable media exports.

The current backend does not implement first-class incident modes yet. Planned modes such as emergency incidents, interaction records, safety checks, and evidence notes are documented in `docs/incident-modes.md`.

See the repository root `README.md`, `docs/README.md`, `docs/architecture.md`, `docs/incident-modes.md`, `docs/encryption.md`, `docs/api.md`, and `docs/code-map.md` for the full route and architecture notes.
