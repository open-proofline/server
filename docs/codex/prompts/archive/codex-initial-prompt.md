# Codex Initial Prompt: Safety Recorder Go Backend v0.1

Create a minimal Go backend for a personal safety recording app.

This is **v0.1**. Keep it boring, small, and testable.

## Project goal

The iPhone app will record encrypted audio/video chunks and upload them continuously.

The backend must:

- retain every successfully uploaded chunk
- verify uploaded file hashes
- reject duplicate chunks
- keep incident metadata
- assume chunks are already encrypted by the client

Do **not** implement encryption in the backend.

## Important security boundary

Do **not** implement:

- public login
- OAuth
- JWT auth
- user accounts
- admin UI
- React
- token sharing
- public emergency viewer

For v0.1 this API is private/dev only and will later be placed behind WireGuard/firewall rules.

Add a README warning that this must not be exposed publicly in its current form.

## Use

- Go
- Standard library `net/http` where practical
- SQLite for metadata
- Local disk storage for uploaded encrypted blobs
- Simple structured JSON responses
- Tests

Prefer minimal dependencies.

SQLite dependency is acceptable.

Do **not** add Docker, Kubernetes, React, OpenAPI generators, or a framework unless genuinely necessary.

## Repo structure

```text
server/
  cmd/api/main.go
  internal/config/
  internal/httpapi/
  internal/incidents/
  internal/storage/
  internal/db/
  migrations/
  README.md
```

## Configuration

Use environment variables:

| Variable | Default |
|---|---|
| `SAFE_BIND_ADDR` | `:8080` |
| `SAFE_DATA_DIR` | `./data` |
| `SAFE_DB_PATH` | `./data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

## Data model

### Incident

- `id`
- `created_at`
- `updated_at`
- `status`: `open` or `closed`
- `client_label`: optional string
- `notes`: optional string

### Chunk

- `id`
- `incident_id`
- `chunk_index`
- `media_type`: `audio`, `video`, `location`, or `metadata`
- `started_at`
- `ended_at`
- `original_filename`
- `stored_path`
- `byte_size`
- `sha256_hex`
- `created_at`

### Checkin

- `id`
- `incident_id`
- `created_at`
- `device_battery_percent`: optional integer
- `device_network`: optional string
- `latitude`: optional float
- `longitude`: optional float
- `accuracy_meters`: optional float

## API endpoints

### `POST /v1/incidents`

Create a new incident.

Request:

```json
{
  "client_label": "optional string",
  "notes": "optional string"
}
```

Response `201`:

```json
{
  "incident_id": "...",
  "status": "open"
}
```

---

### `GET /v1/incidents/{incident_id}`

Return incident metadata, chunks, and checkins.

Response:

```json
{
  "incident": {},
  "chunks": [],
  "checkins": []
}
```

---

### `POST /v1/incidents/{incident_id}/chunks`

Upload an encrypted chunk.

Accept `multipart/form-data`:

| Field | Type | Required |
|---|---|---|
| `file` | file | yes |
| `chunk_index` | integer | yes |
| `media_type` | `audio`, `video`, `location`, `metadata` | yes |
| `started_at` | RFC3339 timestamp | yes |
| `ended_at` | RFC3339 timestamp | yes |
| `sha256_hex` | lowercase SHA-256 hex | yes |
| `original_filename` | string | no |

Requirements:

- reject if incident does not exist
- reject if incident is closed
- reject if `chunk_index` already exists for that incident and media type
- enforce `SAFE_MAX_UPLOAD_BYTES`
- stream upload to a temporary file
- compute SHA-256 while reading
- compare computed hash to `sha256_hex`
- if hash mismatch, delete temp file and return `400`
- store final file under:

```text
data/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Example:

```text
data/incidents/abc123/audio_000001.enc
```

Rules:

- never overwrite an existing stored chunk
- insert chunk metadata into SQLite only after file is safely written
- return `201` with chunk metadata

---

### `GET /v1/incidents/{incident_id}/chunks`

Return chunk metadata only.

Do not return file contents.

---

### `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`

Return stored encrypted chunk bytes.

This is for dev/private testing only.

Set:

```http
Content-Type: application/octet-stream
```

---

### `POST /v1/incidents/{incident_id}/checkins`

Create a checkin.

Request:

```json
{
  "device_battery_percent": 82,
  "device_network": "wifi",
  "latitude": -37.0,
  "longitude": 145.0,
  "accuracy_meters": 20
}
```

All fields except `incident_id` are optional.

Return `201`.

---

### `POST /v1/incidents/{incident_id}/close`

Mark incident as closed.

Return updated incident.

## Implementation requirements

- use context-aware DB operations
- add graceful shutdown on `SIGINT` and `SIGTERM`
- add request logging middleware
- add panic recovery middleware
- do not log request bodies
- do not log uploaded file bytes
- do not log `Authorization` headers or future token-like values
- return consistent JSON errors

Error shape:

```json
{
  "error": {
    "code": "hash_mismatch",
    "message": "computed SHA-256 did not match provided hash"
  }
}
```

Use appropriate HTTP status codes:

| Status | Meaning |
|---|---|
| `400` | bad request |
| `404` | not found |
| `409` | duplicate chunk / conflict |
| `413` | upload too large |
| `500` | internal error |

## SQLite requirements

- create migrations or simple schema init code
- enable WAL mode if appropriate
- enable foreign keys
- add unique constraint on:

```text
incident_id + media_type + chunk_index
```

## Tests

Add Go tests covering:

- creating an incident
- uploading a valid chunk
- rejecting duplicate chunk index
- rejecting hash mismatch
- ensuring bad temp file is removed after hash mismatch
- rejecting upload to missing incident
- closing incident
- rejecting upload after close
- listing incident with chunks/checkins

## README requirements

Include:

- how to run tests
- how to start the server
- curl example for creating an incident
- curl example for uploading a chunk
- warning that v0.1 has no public auth and must only be run locally/private network
- next steps section:
  - WireGuard-only bind/firewall
  - iOS client
  - client-side encryption
  - dead-man switch
  - emergency read-only token viewer

## After implementing

Run:

```bash
go test ./...
```

Fix any failures.
