# Code Map

Safety Recorder currently contains the Go backend for a private personal-safety recording system. This backend receives already-encrypted recording chunks, groups them into media streams, records metadata in SQLite, and serves a scoped read-only emergency viewer with encrypted evidence bundle downloads.

## Package Layout

- `.github/workflows/ci.yml`: runs Go tests on pull requests and pushes, builds a Linux amd64 binary artifact, builds the Docker image, and publishes it to GitHub Container Registry on `main` and `v*` tag pushes.
- `server/cmd/api`: starts one private API HTTP server per private bind address and one public emergency viewer HTTP server per public bind address, loads config, opens SQLite, creates storage, wires shared handlers, and handles graceful shutdown.
- `server/cmd/simclient`: simulates the future iOS client by creating an incident, creating an emergency viewer token, creating a media stream, encrypting and uploading fake chunks, completing the stream, sending periodic checkins, and optionally testing hash-failure retry, bundle download, and local decrypt verification behavior.
- `server/internal/config`: reads environment variables such as private/public bind address lists, legacy singular bind addresses, data directory, database path, max upload size, and HTTP server timeouts.
- `server/internal/db`: opens SQLite, enables foreign keys and WAL mode, applies embedded migrations, records `schema_migrations`, and runs named compatibility migrations.
- `server/internal/envelope`: implements the simulator/test AES-256-GCM client-side chunk envelope, associated data builder, and local simulator key file helpers.
- `server/internal/httpapi`: owns separate private/public muxes, JSON responses, request logging, recovery, request validation, upload handling, stream state handlers, ZIP bundle streaming, and the emergency viewer.
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

## Emergency Viewer Flow

Emergency tokens are created on the private API server by `POST /v1/incidents/{incident_id}/emergency-tokens`. The raw token is returned once, while `server/internal/incidents.Repository.CreateEmergencyToken` stores only a SHA-256 hash in SQLite.

`GET /e/{token}` is mounted only on the public emergency viewer server. It renders `server/internal/httpapi/web/templates/emergency.html` with `html/template`. CSS and JavaScript are embedded from `server/internal/httpapi/web/static`. `GET /e/{token}/data` returns the same read-only summary as JSON for polling.

Token lookup checks the hash, expiry, and revocation state before incident metadata is loaded. Invalid, expired, and revoked tokens all return the same public error. Emergency responses use `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, a strict `Content-Security-Policy`, restrictive `Permissions-Policy`, and `Cache-Control: no-store` for token-protected responses.

Completed stream bundle downloads are served by `server/internal/httpapi/bundles.go`. Bundles are generated on demand as ZIP responses and are not cached on disk. ZIP entry names are server-controlled, manifests are generated from database metadata, and chunk bytes are streamed from storage one file at a time. The first bundle format contains encrypted chunks and JSON manifests only; it does not decrypt, merge, or export playable media.

## Before Public Exposure

The separate ports are a deployment boundary, not a complete security model. Do not expose the private API server beyond localhost or a private network as-is. Review and add:

- real access control for `/v1` or a strict WireGuard/firewall-only deployment
- rate limits and abuse controls
- TLS and reverse-proxy settings for the public emergency viewer, if reachable over a network
- retention, backup, and secure deletion policy
- operational monitoring for failed uploads and storage/DB errors
- a production review of emergency token sharing, expiry defaults, and revocation operations

## Out Of Scope Today

The repository does not currently include the iOS app, local recording, production client key storage, key sharing, browser/client-side decryption, server-assisted break-glass key access, playable media export, push notifications, SMS, Messenger integration, user accounts, or a public admin dashboard.
