# API

This is the current backend-only HTTP surface for Proofline. The API binary starts private API listeners and public incident viewer listeners on one or more configured bind addresses. The `/v1` routes are private and unauthenticated. The incident viewer routes are token-gated and read-only. Planned web, iOS, and Android clients are not part of this repository yet.

Media bundle downloads are encrypted chunk bundles. The backend does not decrypt, merge, or produce playable media. The simulator's current encrypted uploads use the envelope documented in [encryption.md](encryption.md), but the API treats uploaded bytes as opaque ciphertext.

The current API stores generic incidents only. Planned incident modes such as emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes are documented in [incident-modes.md](incident-modes.md), but first-class `incident_type`, escalation-policy, account, and trusted-contact APIs do not exist yet.

Default bind addresses:

- private API server: `127.0.0.1:8080`
- public incident viewer server: `127.0.0.1:8081`

Use `SAFE_PRIVATE_BIND_ADDRS` and `SAFE_PUBLIC_BIND_ADDRS` for comma-separated bind-address lists. The older singular variables remain supported when the matching plural variable is unset.

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

Non-upload JSON bodies are limited to 64 KiB. Upload file bytes are limited by `SAFE_MAX_UPLOAD_BYTES`; multipart metadata has a small fixed overhead allowance. `SAFE_MAX_UPLOAD_BYTES` accepts a positive byte count or binary unit suffixes `B`, `K`/`KB`, `M`/`MB`, and `G`/`GB`. Fractional unit values are allowed when they resolve to at least one byte. Non-positive, sub-byte, invalid, and oversized values are rejected during startup.

## Incidents

Incident routes are mounted only on the private API server.

### `POST /v1/incidents`

Creates an open generic incident. Future clients may classify incidents as emergency incidents, interaction records, safety checks, or evidence notes in client/protocol metadata after that design exists. The current request does not accept an incident type or escalation policy.

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
  "streams": [],
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
- `stream_id`: optional media stream ID for new clients
- `chunk_index`: non-negative integer for legacy unstreamed uploads; positive integer when `stream_id` is provided
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
  "stream_id": "str_...",
  "chunk_index": 1,
  "media_type": "audio",
  "started_at": "2026-05-21T10:00:00Z",
  "ended_at": "2026-05-21T10:00:10Z",
  "original_filename": "chunk.enc",
  "stored_path": "incidents/inc_.../streams/str_.../audio_000001.enc",
  "byte_size": 23,
  "sha256_hex": "...",
  "created_at": "2026-05-21T10:00:11Z"
}
```

When `stream_id` is provided, the stream must exist, belong to the same incident, be open, and have the same `media_type` as the uploaded chunk. Streamed chunks must use indexes starting at `1`; `chunk_index <= 0` returns `400 invalid_chunk_index`. Uploads to completed or failed streams return `409 stream_not_open`.

New clients should create a media stream and upload chunks with `stream_id`. `stream_id` remains optional for backwards compatibility with existing chunks and clients. Legacy unstreamed chunks may use `chunk_index = 0`; they are still stored and listed, but they are not included in completed-stream bundle downloads.

Streamed chunk identity is `(incident_id, stream_id, chunk_index)`, so each stream can use normal stream-local chunk numbering. Legacy unstreamed chunk identity remains `(incident_id, media_type, chunk_index)` for chunks without `stream_id`.

Duplicate streamed `(incident_id, stream_id, chunk_index)` uploads and duplicate legacy `(incident_id, media_type, chunk_index)` uploads return `409 duplicate_chunk`. Hash mismatches return `400 hash_mismatch` and do not commit a final file.

The repository rechecks incident and stream state when chunk metadata is inserted. If an upload races with incident close or stream completion, the final metadata insert is rejected and the committed blob path is removed.

For clients using the v1 encryption envelope, `sha256_hex` is the SHA-256 of the complete uploaded envelope bytes, not the plaintext.

### `GET /v1/incidents/{incident_id}/chunks`

Lists chunk metadata for one incident.

### `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`

Returns encrypted bytes for a legacy unstreamed chunk as `application/octet-stream`. This route is private/dev-only and is not used by the incident viewer. Streamed chunks are read through completed stream bundle downloads rather than this legacy media/index route.

## Media Streams

Media stream routes are mounted only on the private API server.

### `POST /v1/incidents/{incident_id}/streams`

Creates an open media stream for an incident.

Request:

```json
{
  "media_type": "audio",
  "label": "main audio recording"
}
```

Response `201`:

```json
{
  "stream": {
    "id": "str_...",
    "incident_id": "inc_...",
    "media_type": "audio",
    "label": "main audio recording",
    "status": "open",
    "created_at": "2026-05-21T10:00:00Z",
    "updated_at": "2026-05-21T10:00:00Z"
  }
}
```

Invalid media types return `400 invalid_media_type`.

### `GET /v1/incidents/{incident_id}/streams`

Lists media streams for an incident.

Response `200`:

```json
{
  "streams": []
}
```

### `GET /v1/incidents/{incident_id}/streams/{stream_id}`

Returns one stream as:

```json
{
  "stream": {
    "id": "str_...",
    "incident_id": "inc_...",
    "media_type": "audio",
    "status": "open",
    "created_at": "2026-05-21T10:00:00Z",
    "updated_at": "2026-05-21T10:00:00Z"
  }
}
```

### `POST /v1/incidents/{incident_id}/streams/{stream_id}/complete`

Marks an open stream complete after verifying chunks `1..expected_chunk_count` exist contiguously and each stored file is readable. Completion revalidates chunk rows in the repository before committing the state change.

Request:

```json
{
  "expected_chunk_count": 12
}
```

Response `200`:

```json
{
  "stream": {
    "id": "str_...",
    "incident_id": "inc_...",
    "media_type": "audio",
    "status": "complete",
    "expected_chunk_count": 12,
    "completed_at": "2026-05-21T10:02:00Z",
    "created_at": "2026-05-21T10:00:00Z",
    "updated_at": "2026-05-21T10:02:00Z"
  }
}
```

Missing or non-contiguous chunks return `409 stream_chunks_incomplete` or `409 stream_chunks_not_contiguous`. Completing an already complete or failed stream returns `409`.

### `POST /v1/incidents/{incident_id}/streams/{stream_id}/fail`

Marks an open stream failed while preserving uploaded chunks.

Request:

```json
{
  "failure_reason": "client stopped recording unexpectedly"
}
```

Response `200` is the updated stream object with `status: "failed"` and `failed_at` set.

### `GET /v1/incidents/{incident_id}/streams/{stream_id}/download`

Downloads a completed stream as a ZIP bundle. Open or failed streams return `409 stream_not_complete`.

Successful responses include:

```http
Content-Type: application/zip
Content-Disposition: attachment; filename="incident_inc_..._audio_str_....zip"
Content-Security-Policy: default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'
X-Content-Type-Options: nosniff
Cache-Control: no-store
Referrer-Policy: no-referrer
Permissions-Policy: geolocation=(), microphone=(), camera=()
X-Frame-Options: DENY
```

ZIP contents:

```text
manifest.json
chunks/audio_000001.enc
chunks/audio_000002.enc
```

The manifest is generated from trusted database metadata and includes incident ID, stream ID, media type, status, chunk count, total bytes, and chunk SHA-256 metadata. Server filesystem paths are not included.
It also includes a non-secret `encryption` hint indicating expected client-side encryption and `server_decrypts: false`.

### `GET /v1/incidents/{incident_id}/download`

Downloads a ZIP bundle containing all completed streams for an incident:

```text
manifest.json
streams/{stream_id}/manifest.json
streams/{stream_id}/chunks/audio_000001.enc
```

Open, failed, and legacy unstreamed chunks are omitted from this initial bundle format.

If any completed stream cannot be reconstructed, the incident bundle request fails with `409 incident_bundle_inconsistent` rather than returning a partial bundle. The error response does not include server filesystem paths, stored chunk paths, or ZIP entry names.

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

## Viewer Tokens

Incident-token creation and revocation routes are mounted only on the private API server.

### `POST /v1/incidents/{incident_id}/incident-tokens`

Creates a read-only viewer token for one incident. The raw token is returned only in this response; SQLite stores only a SHA-256 hash.

`expires_at` is optional. When omitted, the API applies the configured default token lifetime, which is 24 hours unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is changed. Explicit `expires_at` values are preserved; send `null` to explicitly create a token that remains valid until revoked. Setting `SAFE_DEFAULT_INCIDENT_TOKEN_TTL=0` disables the default and lets omitted expiries remain valid until revoked.

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
  "token_id": "itk_...",
  "incident_id": "inc_...",
  "token": "...",
  "label": "trusted contact",
  "created_at": "2026-05-21T10:00:00Z",
  "expires_at": "2030-01-01T00:00:00Z"
}
```

The response includes `Cache-Control: no-store`.

### `POST /v1/incident-tokens/{token_id}/revoke`

Revokes a viewer token by ID.

Response `200`:

```json
{
  "token_id": "itk_...",
  "revoked": true
}
```

## Incident Viewer

Incident viewer routes are mounted only on the public incident viewer server.
`/i/{token}` is the canonical path for new links. The pre-rename `/e/{token}`
paths remain as compatibility aliases for already shared viewer URLs, including
the `/data`, stream download, and incident download variants.

### `GET /i/{token}`

Renders a read-only HTML summary for a valid, unexpired, unrevoked token. The page includes embedded static CSS/JS files served from `/static/`.

### `GET /i/{token}/data`

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
  "streams": [],
  "completed_streams": [],
  "warning": "If you are concerned about immediate safety, call emergency services now.",
  "generated_at": "2026-05-21T10:00:12Z"
}
```

Incident viewer responses include `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, `Permissions-Policy: geolocation=(), microphone=(), camera=()`, `X-Frame-Options: DENY`, and a strict `Content-Security-Policy` with `frame-ancestors 'none'`. Token-protected pages, JSON, errors, and downloads include `Cache-Control: no-store`. Invalid, expired, and revoked tokens all return `404 incident_token_invalid`.

The Go app does not set `Strict-Transport-Security` in local/dev HTTP mode. Set HSTS at the HTTPS reverse proxy or deployment edge for production hostnames.

### `GET /i/{token}/streams/{stream_id}/download`

Downloads a completed stream bundle for the token's incident. The route is read-only and never accepts a client-provided file path. Invalid, expired, and revoked tokens return `404 incident_token_invalid`.

### `GET /i/{token}/incident/download`

Downloads all completed streams for the token's incident as one encrypted evidence ZIP. Failed/open streams and legacy unstreamed chunks are omitted.

If any completed stream cannot be reconstructed, the incident bundle request fails with `409 incident_bundle_inconsistent` rather than returning a partial bundle. Invalid, expired, and revoked tokens still return `404 incident_token_invalid`.
