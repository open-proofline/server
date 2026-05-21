# Code Map

Safety Recorder currently contains the Go backend for a private personal-safety recording system. This backend receives already-encrypted recording chunks, records metadata in SQLite, and serves a scoped read-only emergency viewer.

## Package Layout

- `server/cmd/api`: starts the HTTP server, loads config, opens SQLite, creates storage, wires handlers, and handles graceful shutdown.
- `server/internal/config`: reads environment variables such as bind address, data directory, database path, and max upload size.
- `server/internal/db`: opens SQLite, enables foreign keys and WAL mode, and applies embedded migrations.
- `server/internal/httpapi`: owns routes, JSON responses, request logging, recovery, request validation, upload handling, and the emergency viewer.
- `server/internal/incidents`: defines incident/chunk/checkin models and writes metadata to SQLite.
- `server/internal/storage`: manages local disk blob storage, including temp uploads, hashing while streaming, and immutable final paths.
- `server/migrations`: embeds the SQLite schema.

## Main Request Flow

Incidents are created in `server/internal/httpapi.createIncident`, which calls `server/internal/incidents.Repository.CreateIncident`.

Chunks are uploaded through `POST /v1/incidents/{incident_id}/chunks`, handled by `server/internal/httpapi.uploadChunk`.

Upload handling first checks that the incident exists and is open. The file is then streamed by `server/internal/httpapi.readChunkUpload` into `server/internal/storage.Store.SaveTemp`, which writes to `data/tmp` while computing SHA-256 and enforcing the upload byte limit.

Hash verification happens in `server/internal/httpapi.uploadChunk` by comparing the computed temp-file hash with the client-provided `sha256_hex`.

After verification, `server/internal/storage.Store.CommitTemp` stores the file under:

```text
data/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

It uses no-overwrite behavior, so an existing chunk file is treated as a conflict.

SQLite metadata is written after the file is safely committed, through `server/internal/incidents.Repository.CreateChunk`. The schema also has a unique constraint on `incident_id + media_type + chunk_index`.

## Emergency Viewer Flow

Emergency tokens are created by `POST /v1/incidents/{incident_id}/emergency-tokens`. The raw token is returned once, while `server/internal/incidents.Repository.CreateEmergencyToken` stores only a SHA-256 hash in SQLite.

`GET /e/{token}` renders `server/internal/httpapi/web/templates/emergency.html` with `html/template`. CSS and JavaScript are embedded from `server/internal/httpapi/web/static`. `GET /e/{token}/data` returns the same read-only summary as JSON for polling.

Token lookup checks the hash, expiry, and revocation state before incident metadata is loaded. Invalid, expired, and revoked tokens all return the same public error. Emergency responses use `Referrer-Policy: no-referrer` and `Cache-Control: no-store`.

## Before Public Exposure

Do not expose v0.1 beyond localhost or a private network as-is. Review and add:

- real access control for `/v1` or a strict WireGuard/firewall-only deployment
- rate limits and abuse controls
- TLS and reverse-proxy settings, if reachable over a network
- retention, backup, and secure deletion policy
- operational monitoring for failed uploads and storage/DB errors
- a production review of emergency token sharing, expiry defaults, and revocation operations

## Out Of Scope Today

The repository does not currently include the iOS app, local recording, local encryption implementation, push notifications, SMS, Messenger integration, user accounts, or a public admin dashboard.
