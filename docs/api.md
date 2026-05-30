# API

This is the current backend-only HTTP surface for Proofline. The API binary starts private API listeners and public incident viewer listeners on one or more configured bind addresses. The `/v1` routes are private and require local account authentication. The incident viewer routes are token-gated and read-only. Planned web, iOS, and Android clients are not part of this repository yet.

Media bundle downloads are encrypted chunk bundles. The backend does not decrypt, merge, or produce playable media. The simulator's current encrypted uploads use the envelope documented in [encryption.md](encryption.md), but the API treats uploaded bytes as opaque ciphertext.

The current API stores generic incidents owned by local accounts. Planned incident modes such as emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes are documented in [incident-modes.md](incident-modes.md), along with future capture-profile, escalation-policy, sharing-state, and migration boundaries. First-class incident-mode, escalation-policy, trusted-contact, and public product APIs do not exist yet.

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

## Authentication And Accounts

Private `/v1` routes require:

```http
Authorization: Bearer <session_token>
```

Session tokens are opaque server-side credentials. The raw token is returned only by login, while the metadata backend stores only its SHA-256 hash. Sessions expire after `SAFE_SESSION_TTL`, defaulting to `12h`, and can be revoked by logout, password reset, or the admin session-revocation route.

On startup, the server fails closed unless an admin account already exists or `SAFE_AUTH_BOOTSTRAP_SECRET` is set. With that secret set, create the first admin through the one-time bootstrap route, then remove the environment variable and restart or redeploy without it. The bootstrap route is disabled after an admin account exists.

### `POST /v1/bootstrap/admin`

Creates the first local admin account when no admin exists. This route does not require a session, but it requires the bootstrap secret header:

```http
X-Proofline-Bootstrap-Secret: ...
```

Request:

```json
{
  "username": "admin",
  "password": "long local password"
}
```

Response `201`:

```json
{
  "account": {
    "id": "acct_...",
    "username": "admin",
    "role": "admin",
    "created_at": "2026-05-31T10:00:00Z",
    "updated_at": "2026-05-31T10:00:00Z",
    "password_changed_at": "2026-05-31T10:00:00Z"
  }
}
```

### `POST /v1/auth/login`

Authenticates a local account and returns a raw session token once.

Request:

```json
{
  "username": "admin",
  "password": "long local password"
}
```

Response `201`:

```json
{
  "session_id": "ses_...",
  "account": {
    "id": "acct_...",
    "username": "admin",
    "role": "admin",
    "created_at": "2026-05-31T10:00:00Z",
    "updated_at": "2026-05-31T10:00:00Z",
    "password_changed_at": "2026-05-31T10:00:00Z"
  },
  "token": "...",
  "created_at": "2026-05-31T10:00:00Z",
  "expires_at": "2026-05-31T22:00:00Z"
}
```

### `POST /v1/auth/logout`

Revokes the current session.

### `GET /v1/account`

Returns the authenticated account.

### `POST /v1/account/password`

Changes the authenticated account password after verifying `current_password`; other sessions for the account are revoked.

### Private Admin Web Routes

The private API listener also serves a small admin web surface outside the
`/v1` API namespace:

- `GET /admin`
- `POST /admin/login`
- `POST /admin/bootstrap`
- `POST /admin/logout`
- `POST /admin/password`
- `POST /admin/accounts/{account_id}/password`
- `GET /admin/static/styles.css`

`GET /admin` renders either the first-admin bootstrap form, the admin login
form, or the authenticated admin dashboard. The form handlers reuse the same
local account records and opaque server-side session store as the JSON API, but
the browser flow stores the raw session token in an HttpOnly, SameSite=Strict
cookie scoped to `/admin`.

The bootstrap screen is available only when no admin account exists and
`SAFE_AUTH_BOOTSTRAP_SECRET` is configured. It requires the bootstrap secret,
admin username, and admin password. After an admin exists, `/admin` shows the
login screen and requires an admin account. Non-admin sessions are rejected.

The authenticated dashboard lists local accounts and offers password workflows.
`POST /admin/logout` revokes the current admin web session. `POST
/admin/password` changes the current admin account password after verifying the
current password, then revokes other sessions for that account. `POST
/admin/accounts/{account_id}/password` lets an admin reset another local
account password and revokes all sessions for that account. These authenticated
state-changing forms use a session-bound CSRF token.

`/admin/static/styles.css` is unauthenticated because it is token-neutral static
CSS from the AGPL-licensed source tree. It does not contain incident data,
tokens, deployment details, keys, or evidence metadata. The admin HTML pages
use `Cache-Control: no-store` and conservative browser security headers.

The admin web surface shows only route-boundary status, safe navigation stubs,
and local account-management data. It does not expose incident evidence, viewer
tokens, session tokens, password hashes, request bodies, uploaded bytes,
Authorization headers, plaintext, raw keys, stored paths, object keys, private
deployment details, or sensitive evidence metadata. It is not a public admin
dashboard and must stay on the private listener.

### Admin Account API Routes

The following routes require an admin account session:

- `GET /v1/admin/accounts`
- `POST /v1/admin/accounts`
- `POST /v1/admin/accounts/{account_id}/password`
- `POST /v1/admin/accounts/{account_id}/sessions/revoke`

`POST /v1/admin/accounts` accepts `username`, `password`, and `role`, where `role` is `user` or `admin`. Admin password reset and explicit session revocation revoke all sessions for the selected account.

Local account authentication does not make `/v1` a public product API. Keep private listeners behind localhost, LAN, WireGuard, firewall rules, or a strict private reverse proxy until public exposure, abuse controls, rate limiting, CSRF/browser credential rules, and production operations are explicitly designed and reviewed.

## Incidents

Incident routes are mounted only on the private API server and require a valid session. Incidents are owned by the account that creates them. Regular users can access only their own incidents; admins can access incidents across accounts. Legacy unowned incidents are admin-only until a future migration or reassignment workflow is implemented.

### `POST /v1/incidents`

Creates an open generic incident. Future clients may classify incidents as emergency incidents, interaction records, safety checks, or evidence notes only after the protocol, access-control, migration, and viewer-wording design is implemented. The current request does not accept an incident mode, capture profile, escalation policy, or sharing state.

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
- `original_filename`: optional client-supplied display metadata

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

`original_filename` is metadata, not a storage path. The server trims the value,
normalizes slash and backslash separators to a basename, falls back to the
multipart upload filename when the explicit field is empty, and stores the
resulting basename with chunk metadata. The value may be returned by private
chunk metadata routes, token-scoped public incident viewer summaries, and
completed stream or incident bundle manifests. Server stored paths, staging
paths, local filesystem paths, and object-storage keys are separate
server-controlled values and are not derived from `original_filename`.

Future clients should omit `original_filename` by default or send a generic,
non-identifying basename unless the user or a future protocol mode explicitly
chooses to preserve filename context as evidence metadata. Filenames can still
contain personal or contextual information even after path stripping. Do not use
`original_filename` for identity, authorization, storage lookup, decryption,
legal-record guarantees, or download path construction.

The current API does not implement resumable uploads, upload leases, or
client-side queue summary endpoints. Future clients should retry complete
encrypted chunks unless a later explicit resumable-upload protocol is
implemented. The planning decision is documented in
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md).

### Planned Duplicate Chunk Reconciliation

This section is a design contract for future implementation. The current server
does not yet expose the reconciliation route described here; duplicate uploads
still return `409 duplicate_chunk`.

The planned API shape is a separate private query workflow, not a public route
and not an enriched `409 duplicate_chunk` upload response. A separate route lets
clients compare expected metadata without re-uploading ciphertext, keeps
duplicate upload errors small, and can coexist with future idempotency-key
retry success.

Planned route:

```http
POST /v1/incidents/{incident_id}/chunks/reconcile
```

Request:

```json
{
  "stream_id": "str_...",
  "chunk_index": 1,
  "media_type": "audio",
  "started_at": "2026-05-21T10:00:00Z",
  "ended_at": "2026-05-21T10:00:10Z",
  "byte_size": 23,
  "sha256_hex": "...",
  "original_filename": "chunk.enc"
}
```

For streamed chunks, `stream_id` is required and identity is
`(incident_id, stream_id, chunk_index)`. `media_type` remains required and must
match the stream media type. For legacy unstreamed chunks, omit `stream_id`;
identity is `(incident_id, media_type, chunk_index)`, and `chunk_index = 0`
remains valid for compatibility.

The comparison fingerprint is:

- normalized chunk identity
- `media_type`
- `started_at`
- `ended_at`
- normalized `original_filename`, including empty value
- ciphertext `byte_size`
- ciphertext `sha256_hex`

The route should allow reconciliation after an incident is closed or a stream is
complete or failed, because it is read-only and only confirms already accepted
metadata. It must not overwrite, replace, delete, or rewrite stored chunks.

Matched response `200`:

```json
{
  "reconciliation": {
    "status": "matched",
    "identity": {
      "incident_id": "inc_...",
      "stream_id": "str_...",
      "chunk_index": 1,
      "media_type": "audio"
    },
    "chunk_id": "chk_...",
    "byte_size": 23,
    "sha256_hex": "...",
    "started_at": "2026-05-21T10:00:00Z",
    "ended_at": "2026-05-21T10:00:10Z",
    "created_at": "2026-05-21T10:00:11Z"
  }
}
```

Conflict response `409`:

```json
{
  "error": {
    "code": "duplicate_chunk_conflict",
    "message": "existing chunk does not match expected ciphertext or metadata"
  },
  "reconciliation": {
    "status": "conflict",
    "identity": {
      "incident_id": "inc_...",
      "stream_id": "str_...",
      "chunk_index": 1,
      "media_type": "audio"
    },
    "mismatched_fields": ["sha256_hex", "byte_size"]
  }
}
```

The conflict response should identify mismatched field names, not the existing
stored values. If no accepted chunk exists for the requested identity, return
`404 chunk_not_found`. Invalid identity or fingerprint fields should reuse the
existing upload validation error codes where practical, such as
`400 invalid_chunk_index`, `400 invalid_media_type`, or
`400 invalid_sha256_hex`.

Safe reconciliation responses may return server-generated chunk ID, normalized
identity fields, timestamps, byte size, ciphertext hash, creation time, and
field names that matched or mismatched. They must not return uploaded bytes,
plaintext, raw keys, raw tokens, request bodies, local filesystem paths,
`stored_path`, staging paths, object-storage keys, or object-storage
credentials.

Future implementation should extend `internal/httpapi/uploads_test.go` for:

- matched streamed duplicate reconciliation
- conflicting streamed duplicate reconciliation
- matched and conflicting legacy unstreamed reconciliation
- omission of `stored_path` and stored conflicting values from reconciliation
  responses
- read-only reconciliation after stream completion, stream failure, or incident
  close

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

The manifest is generated from trusted database metadata and includes incident
ID, stream ID, media type, status, chunk count, total bytes, chunk SHA-256
metadata, and any stored `original_filename` basename for each chunk. Server
filesystem paths are not included.
It also includes a non-secret `encryption` hint indicating expected client-side encryption and `server_decrypts: false`.

Future live or partial stream access is planning-only and should not be inferred
from this completed bundle route. See
[live-partial-stream-access-boundary.md](live-partial-stream-access-boundary.md).

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

Creates a read-only viewer token for one incident. The raw token is returned only in this response; the configured metadata backend stores only a SHA-256 hash.

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

Open and failed streams are visible only as metadata in the current viewer
summary. The current token-scoped viewer does not expose live chunk bytes or
partial stream manifests. See
[live-partial-stream-access-boundary.md](live-partial-stream-access-boundary.md).

### `GET /i/{token}/incident/download`

Downloads all completed streams for the token's incident as one encrypted evidence ZIP. Failed/open streams and legacy unstreamed chunks are omitted.

If any completed stream cannot be reconstructed, the incident bundle request fails with `409 incident_bundle_inconsistent` rather than returning a partial bundle. Invalid, expired, and revoked tokens still return `404 incident_token_invalid`.
