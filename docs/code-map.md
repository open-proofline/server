# Code Map

Proofline Server currently contains the Go backend for a private encrypted incident-capture system. This backend receives already-encrypted recording chunks, groups them into media streams, records metadata in SQLite, and serves a scoped read-only incident viewer with encrypted evidence bundle downloads.

This repository is the server/backend component only. In the planned `open-proofline` organisation layout it corresponds to `open-proofline/server`. Future web-client, iOS-client, Android-client, and protocol implementation should live in separate repositories.

The current backend stores generic incidents only. Planned future clients may classify incidents as emergency incidents, non-emergency interaction records, timed safety checks, or evidence notes after the protocol, schema, access-control, and client designs exist. See [incident-modes.md](incident-modes.md).

## Package Layout

- `.github/workflows/ci.yml`: runs Go tests with a coverage signal on pull requests and pushes, runs `govulncheck`, builds a Linux amd64 binary artifact, generates release binary attestations on `v*` tag pushes, uploads the binary as a GitHub Release asset, builds the Docker image, and publishes attested images to GitHub Container Registry from a trusted job limited to `main` and `v*` tag pushes.
- `server/cmd/api`: starts one private API HTTP server per private bind address and one public incident viewer HTTP server per public bind address, loads config, opens SQLite, creates storage, wires shared handlers, and handles graceful shutdown.
- `server/cmd/simclient`: simulates a future client by creating an incident, creating a viewer token, creating a media stream, encrypting and uploading fake chunks, completing the stream, sending periodic checkins, and optionally testing hash-failure retry, bundle download, and local decrypt verification behavior.
- `server/internal/config`: reads environment variables such as private/public bind address lists, legacy singular bind addresses, data directory, database path, max upload size, and HTTP server timeouts.
- `server/internal/db`: opens SQLite, enables foreign keys and WAL mode, applies embedded migrations, records `schema_migrations`, and runs named compatibility migrations.
- `server/internal/envelope`: implements the simulator/test AES-256-GCM client-side chunk envelope, associated data builder, and local simulator key file helpers.
- `server/internal/httpapi`: owns separate private/public muxes, JSON responses, request logging, recovery, request validation, upload handling, stream state handlers, ZIP bundle streaming, and the incident viewer.
- `server/internal/incidents`: defines incident/stream/chunk/checkin models and writes metadata to SQLite.
- `server/internal/storage`: manages local disk blob storage, including temp uploads, hashing while streaming, and immutable final paths.
- `server/migrations`: embeds the SQLite schema.

## Main Request Flow

Incidents are created in `server/internal/httpapi.createIncident`, which calls `server/internal/incidents.Repository.CreateIncident`.

Chunks are uploaded through `POST /v1/incidents/{incident_id}/chunks`, handled by `server/internal/httpapi.uploadChunk`.

Upload handling first checks that the incident exists and is open. The file is then streamed by `server/internal/httpapi.readChunkUpload` into `server/internal/storage.Store.SaveTemp`, which writes to `data/tmp` while computing SHA-256 and enforcing the upload byte limit.

Hash verification happens in `server/internal/httpapi.uploadChunk` by comparing the computed temp-file hash with the client-provided `sha256_hex`.

After verification, `server/internal/storage.Store.CommitTemp` stores the file under:

```text
data/incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Legacy unstreamed chunks keep the older path:

```text
data/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Storage uses no-overwrite behavior, so an existing chunk file is treated as a conflict.

SQLite metadata is written after the file is safely committed, through `server/internal/incidents.Repository.CreateChunk`. The repository rechecks the incident and stream state before inserting chunk metadata so uploads that race with incident close or stream completion are rejected. The schema enforces separate unique identities for streamed and legacy unstreamed chunks.

New clients can create a media stream with `POST /v1/incidents/{incident_id}/streams` and include the returned `stream_id` during chunk upload. Streamed chunk indexes start at `1`, and streamed chunk identity is `incident_id + stream_id + chunk_index`. Existing chunks without `stream_id` remain valid and readable as legacy chunk metadata, including older index `0` chunks; legacy unstreamed identity remains `incident_id + media_type + chunk_index`. Legacy unstreamed chunks are not included in completed-stream evidence bundles.

Stream completion is handled by `server/internal/httpapi.completeMediaStream`. Before a stream moves from `open` to `complete`, the handler verifies that chunks `1..expected_chunk_count` exist contiguously for that stream and that each stored blob can be opened from local storage. `server/internal/incidents.Repository.CompleteMediaStream` then revalidates the chunk rows in the completion transaction before committing the state change. Failed streams preserve uploaded chunks but are not offered as normal downloads.

## Incident Viewer Flow

Viewer tokens are created on the private API server by `POST /v1/incidents/{incident_id}/incident-tokens`. The raw token is returned once, while `server/internal/incidents.Repository.CreateIncidentToken` stores only a SHA-256 hash in SQLite.

`GET /i/{token}` is mounted only on the public incident viewer server. It renders `server/internal/httpapi/web/templates/incident_viewer.html` with `html/template`. CSS and JavaScript are embedded from `server/internal/httpapi/web/static`. `GET /i/{token}/data` returns the same read-only summary as JSON for polling.

Token lookup checks the hash, expiry, and revocation state before incident metadata is loaded. Invalid, expired, and revoked tokens all return the same public error. Viewer responses use `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, a strict `Content-Security-Policy`, restrictive `Permissions-Policy`, and `Cache-Control: no-store` for token-protected responses.

Completed stream bundle downloads are served by `server/internal/httpapi/bundles.go`. Bundles are generated on demand as ZIP responses and are not cached on disk. ZIP entry names are server-controlled, manifests are generated from database metadata, and chunk bytes are streamed from storage one file at a time. The first bundle format contains encrypted chunks and JSON manifests only; it does not decrypt, merge, or export playable media.

## Server Repository Boundary

The separate ports are a deployment boundary, not a complete security model. Do not expose the private API server beyond localhost or a private network as-is.

This repository should stay focused on server/backend work:

- API handlers and routing
- SQLite migrations and repository code
- encrypted blob storage
- token-scoped incident viewer
- backend deployment docs
- backend security, retention, and threat-model docs
- simulator/reference backend flow

Before public exposure, review and add:

- real access control for `/v1` or a strict WireGuard/firewall-only deployment
- rate limits and abuse controls
- TLS and reverse-proxy settings for the public incident viewer, if reachable over a network
- deployment-specific enforcement of the documented [retention, backup, and deletion policy](retention-backup-deletion.md)
- operational monitoring for failed uploads and storage/DB errors
- a production review of viewer-token sharing, expiry defaults, and revocation operations
- first-class incident type, escalation-policy, account, and trusted-contact authorization design before implementing public account workflows

## Out Of Scope Today

The repository does not currently include the web client, iOS app, Android app, protocol repository, local recording, first-class incident types, escalation policies, trusted-contact accounts, dead-man switch notifications, production client key storage, key sharing, browser/client-side decryption, server-assisted break-glass key access, playable media export, push notifications, SMS, Messenger integration, user accounts, or a public admin dashboard.
