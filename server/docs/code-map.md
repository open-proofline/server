# Code Map

This backend is a small private Go API for receiving already-encrypted safety recording chunks and recording metadata in SQLite.

## Package Layout

- `cmd/api`: starts the HTTP server, loads config, opens SQLite, creates storage, wires handlers, and handles graceful shutdown.
- `internal/config`: reads environment variables such as bind address, data directory, database path, and max upload size.
- `internal/db`: opens SQLite, enables foreign keys and WAL mode, and applies embedded migrations.
- `internal/httpapi`: owns routes, JSON responses, request logging, recovery, request validation, and upload handling.
- `internal/incidents`: defines incident/chunk/checkin models and writes metadata to SQLite.
- `internal/storage`: manages local disk blob storage, including temp uploads, hashing while streaming, and immutable final paths.
- `migrations`: embeds the SQLite schema.

## Main Request Flow

Incidents are created in `internal/httpapi.createIncident`, which calls `internal/incidents.Repository.CreateIncident`.

Chunks are uploaded through `POST /v1/incidents/{incident_id}/chunks`, handled by `internal/httpapi.uploadChunk`.

Upload handling first checks that the incident exists and is open. The file is then streamed by `internal/httpapi.readChunkUpload` into `internal/storage.Store.SaveTemp`, which writes to `data/tmp` while computing SHA-256 and enforcing the upload byte limit.

Hash verification happens in `internal/httpapi.uploadChunk` by comparing the computed temp-file hash with the client-provided `sha256_hex`.

After verification, `internal/storage.Store.CommitTemp` stores the file under:

```text
data/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

It uses no-overwrite behavior, so an existing chunk file is treated as a conflict.

SQLite metadata is written after the file is safely committed, through `internal/incidents.Repository.CreateChunk`. The schema also has a unique constraint on `incident_id + media_type + chunk_index`.

## Before Public Exposure

Do not expose v0.1 beyond localhost or a private network as-is. Review and add:

- real access control or a strict WireGuard/firewall-only deployment
- rate limits and abuse controls
- TLS and reverse-proxy settings, if reachable over a network
- retention, backup, and secure deletion policy
- operational monitoring for failed uploads and storage/DB errors
- a careful design for any emergency read-only viewer or token flow
