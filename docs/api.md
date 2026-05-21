# API

This is the current backend-only v0.1 HTTP surface. The binary starts a private API server and a public emergency viewer server. The `/v1` routes are private and unauthenticated. The emergency viewer routes are token-gated and read-only. The planned iOS recording client is not part of this repository yet.

Default bind addresses:

- private API server: `127.0.0.1:8080`
- public emergency viewer server: `127.0.0.1:8081`

## Common Responses

Errors use:

```json
{
  "error": {
    "code": "invalid_json",
    "message": "request body must be valid JSON"
  }
}
```

Non-upload JSON bodies are limited to 64 KiB. Upload file bytes are limited by `SAFE_MAX_UPLOAD_BYTES`; multipart metadata has a small fixed overhead allowance.

## Incidents

Incident routes are mounted only on the private API server.

### `POST /v1/incidents`

Creates an open incident.

Request:

```json
{
  "client_label": "iphone",
  "notes": "test incident"
}
```

Response `201`:

```json
{
  "incident_id": "inc_...",
  "status": "open"
}
```

### `GET /v1/incidents/{incident_id}`

Returns incident metadata, chunk metadata, and checkins. Chunk file bytes are not included.

Response `200`:

```json
{
  "incident": {
    "id": "inc_...",
    "created_at": "2026-05-21T10:00:00Z",
    "updated_at": "2026-05-21T10:00:00Z",
    "status": "open",
    "client_label": "iphone"
  },
  "chunks": [],
  "checkins": []
}
```

### `POST /v1/incidents/{incident_id}/close`

Marks an incident closed. Later chunk uploads return `409 incident_closed`.

Response `200` is the updated incident object.

## Chunks

Chunk routes are mounted only on the private API server.

### `POST /v1/incidents/{incident_id}/chunks`

Uploads one already-encrypted chunk as `multipart/form-data`.

Fields:

- `file`: encrypted chunk bytes
- `chunk_index`: non-negative integer
- `media_type`: `audio`, `video`, `location`, or `metadata`
- `started_at`: RFC3339 timestamp
- `ended_at`: RFC3339 timestamp, not before `started_at`
- `sha256_hex`: lowercase SHA-256 hex of the encrypted bytes
- `original_filename`: optional display metadata

Response `201`:

```json
{
  "id": "chk_...",
  "incident_id": "inc_...",
  "chunk_index": 1,
  "media_type": "audio",
  "started_at": "2026-05-21T10:00:00Z",
  "ended_at": "2026-05-21T10:00:10Z",
  "original_filename": "chunk.enc",
  "stored_path": "incidents/inc_.../audio_000001.enc",
  "byte_size": 23,
  "sha256_hex": "...",
  "created_at": "2026-05-21T10:00:11Z"
}
```

Duplicate `(incident_id, media_type, chunk_index)` uploads return `409 duplicate_chunk`. Hash mismatches return `400 hash_mismatch` and do not commit a final file.

### `GET /v1/incidents/{incident_id}/chunks`

Lists chunk metadata for one incident.

### `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`

Returns encrypted chunk bytes as `application/octet-stream`. This route is private/dev-only and is not used by the emergency viewer.

## Checkins

Checkin routes are mounted only on the private API server.

### `POST /v1/incidents/{incident_id}/checkins`

Adds optional device status and location metadata.

Request:

```json
{
  "device_battery_percent": 82,
  "device_network": "wifi",
  "latitude": -37,
  "longitude": 145,
  "accuracy_meters": 20
}
```

Response `201` is the created checkin.

## Emergency Tokens

Emergency token creation and revocation routes are mounted only on the private API server.

### `POST /v1/incidents/{incident_id}/emergency-tokens`

Creates a read-only emergency token for one incident. The raw token is returned only in this response; SQLite stores only a SHA-256 hash.

Request:

```json
{
  "label": "trusted contact",
  "expires_at": "2030-01-01T00:00:00Z"
}
```

Response `201`:

```json
{
  "token_id": "etk_...",
  "incident_id": "inc_...",
  "token": "...",
  "label": "trusted contact",
  "created_at": "2026-05-21T10:00:00Z",
  "expires_at": "2030-01-01T00:00:00Z"
}
```

The response includes `Cache-Control: no-store`.

### `POST /v1/emergency-tokens/{token_id}/revoke`

Revokes an emergency token by ID.

Response `200`:

```json
{
  "token_id": "etk_...",
  "revoked": true
}
```

## Emergency Viewer

Emergency viewer routes are mounted only on the public emergency viewer server.

### `GET /e/{token}`

Renders a read-only HTML summary for a valid, unexpired, unrevoked token. The page includes embedded static CSS/JS files served from `/static/`.

### `GET /e/{token}/data`

Returns the same read-only summary as JSON for polling.

Response `200`:

```json
{
  "incident": {
    "id": "inc_...",
    "status": "open",
    "client_label": "iphone",
    "created_at": "2026-05-21T10:00:00Z",
    "updated_at": "2026-05-21T10:00:00Z"
  },
  "latest_checkin": null,
  "chunk_count_by_media_type": {},
  "latest_chunk_by_media_type": {},
  "media": [
    {
      "media_type": "audio",
      "chunk_count": 0
    },
    {
      "media_type": "video",
      "chunk_count": 0
    },
    {
      "media_type": "location",
      "chunk_count": 0
    },
    {
      "media_type": "metadata",
      "chunk_count": 0
    }
  ],
  "warning": "If you are concerned about immediate safety, call emergency services now.",
  "generated_at": "2026-05-21T10:00:12Z"
}
```

Emergency Viewer responses include `Referrer-Policy: no-referrer` and `Cache-Control: no-store`. Invalid, expired, and revoked tokens all return `404 emergency_token_invalid`.
