# Code Map

Proofline Server currently contains the Go backend for a private encrypted
incident-capture system. This backend receives already-encrypted recording
chunks, groups them into media streams, records metadata in SQLite by default
or optional PostgreSQL, supports optional Valkey/Redis-compatible short-lived
coordination, serves private incident deletion and optional closed-incident
retention workflows, and serves a scoped read-only incident viewer with
encrypted evidence bundle downloads.

This repository is the server/backend component only. In the planned `open-proofline` organisation layout it corresponds to `open-proofline/server`. Future web-client, iOS-client, Android-client, and protocol implementation should live in separate repositories.

The current backend stores generic incidents by default and can store optional
incident-mode, capture-profile, escalation-policy, and sharing-state metadata on
private incident create/read routes. Those fields do not drive access,
notification, retention, sharing, viewer, or key-custody behavior. Mode-driven
behavior boundaries are documented in [incident-modes.md](incident-modes.md),
with role and grant boundaries in [v1-access-control.md](v1-access-control.md).

## Package Layout

- `go.mod`: defines the root Go module `github.com/open-proofline/server`.
- `.github/workflows/ci.yml`: runs Go tests with a coverage signal on pull requests and pushes, runs `govulncheck`, builds the `proofline-server-linux-amd64` binary artifact, gates release binary attestation and trusted GHCR publishing on the vulnerability scan, uploads the binary as a GitHub Release asset on `v*` tag pushes, builds the Docker image, and publishes attested images to GitHub Container Registry from a trusted job limited to `main`, `develop`, and `v*` tag pushes.
- `.dockerignore`: excludes local runtime, review, and build artifacts from the root Docker build context used by `Dockerfile`.
- `cmd/api`: starts one private API HTTP server per private bind address and one public incident viewer HTTP server per public bind address, loads config, enforces the local account bootstrap gate, checks the selected coordination backend, opens the selected metadata backend, creates storage, wires shared handlers including private health/readiness checks and public viewer rate limiting, starts the deletion worker, and handles graceful shutdown.
- `cmd/simclient`: simulates future client flows by logging in, creating an incident, creating a media stream, encrypting and uploading complete chunks, completing or failing streams, sending periodic checkins, and optionally testing hash-failure retry, bundle download, local decrypt verification, durable desktop-recorder staging, local file input, ffmpeg segment capture, restart/resume behavior, and poor-network retry controls. Token-bearing viewer URLs are omitted from simulator output.
- `internal/config`: reads environment variables such as backend selectors, backend-specific settings, private/public bind address lists, legacy singular bind addresses, data directory, database path, max upload size, public viewer rate limits, HTTP server timeouts, local account bootstrap secret, session TTL, deletion worker interval, closed-incident retention window, token metadata retention window, and tombstone retention window.
- `internal/coordination`: defines the small optional coordination boundary, the default no-coordination backend, and the Valkey/Redis-compatible startup check and public viewer rate-limit counter backend.
- `internal/db`: opens SQLite, enables foreign keys and WAL mode, applies embedded SQLite migrations, records `schema_migrations`, and runs named compatibility migrations.
- `internal/envelope`: implements the simulator/test AES-256-GCM client-side chunk envelope, associated data builder, and local simulator key file helpers.
- `internal/auth`: normalizes local account usernames, validates passwords, hashes passwords with bcrypt, and hashes opaque session tokens before storage.
- `internal/httpapi`: owns separate private/public muxes, JSON responses, request logging, recovery, private account/session authentication, request validation, upload handling, stream state handlers, incident deletion handlers, ZIP bundle streaming, app-level public viewer rate limiting, the private admin web surface, the incident viewer, and the narrow metadata repository boundary consumed by handlers.
- `internal/incidents`: defines incident/stream/chunk/checkin/account/session/deletion models and provides the SQLite metadata repository implementation, including deletion decisions, tombstones, retry item state, and write guards for deleting incidents.
- `internal/postgresdb`: opens optional PostgreSQL metadata connections, applies PostgreSQL migrations, and implements the metadata repository behavior with PostgreSQL transaction, row-locking, deletion, and constraint semantics.
- `internal/retention`: runs the background deletion and optional closed-incident retention worker. It claims retryable deletion decisions, removes encrypted blobs through the storage boundary using stored paths snapshotted from metadata, records safe retry state, prunes sensitive child metadata after blob deletion, and logs only non-sensitive counts or error categories.
- `internal/storage`: defines the blob-store boundary used by HTTP handlers and provides local filesystem and optional S3-compatible implementations, including temp uploads, hashing while streaming, server-controlled stored paths, and immutable final commits.
- `migrations`: embeds the SQLite schema.
- `migrations/postgres`: embeds the PostgreSQL schema.
- `compose`: contains local Docker Compose smoke-test stacks and a runner script
  for disposable SQLite/local, PostgreSQL/local, SQLite/S3-compatible MinIO, and
  full PostgreSQL/MinIO/Valkey validation. These files are local release-smoke
  helpers, not production deployment manifests.

## Main Request Flow

Private `/v1` routes require `Authorization: Bearer <session_token>` except for
bootstrap, login, and the private health/readiness checks. Bootstrap creates the
first admin account when no admin exists and `SAFE_AUTH_BOOTSTRAP_SECRET` is
configured. Session tokens are opaque, returned only to the client, and stored
as hashes by the metadata repository.

`GET /v1/health/live` and `GET /v1/health/ready` are mounted only on the
private API server. Liveness reports process availability. Readiness checks the
selected metadata, blob, and coordination backends and returns only coarse
backend type plus `ok` or `unavailable` status values. It does not expose DSNs,
credentials, bucket names, object keys, stored paths, local filesystem paths,
private hostnames, tokens, uploaded bytes, plaintext, raw keys, or underlying
error strings.

Incidents are created in `internal/httpapi.createIncident`, which calls
`CreateIncidentForAccount` on the configured metadata repository and records the
authenticated account as the owner. Admin accounts can operate across incidents;
regular user accounts are limited to their own incidents. Legacy unowned
incidents are admin-only until a future private reassignment or quarantine
workflow is implemented; see
[legacy unowned incident reassignment](legacy-unowned-incident-reassignment.md).

Chunks are uploaded through `POST /v1/incidents/{incident_id}/chunks`, handled by `internal/httpapi.uploadChunk`.

Upload handling first checks that the incident exists and is open. The file is then streamed by `internal/httpapi.readChunkUpload` into `internal/storage.BlobStore.SaveTemp`, which the current local implementation writes to `data/tmp` while computing SHA-256 and enforcing the upload byte limit.

Hash verification happens in `internal/httpapi.uploadChunk` by comparing the computed temp-file hash with the client-provided `sha256_hex`.

When `Idempotency-Key` is supplied, `internal/httpapi` hashes the raw key,
builds a canonical complete-upload fingerprint from normalized chunk identity,
timestamps, normalized `original_filename`, ciphertext byte size, and
`sha256_hex`, then reserves or replays durable upload-operation state through
the metadata repository. Equivalent retries return `200 OK` with
`Idempotency-Replayed: true`; uploads without the header keep the existing
duplicate behavior.

`POST /v1/incidents/{incident_id}/chunks/reconcile` is a private read-only
metadata route handled by `internal/httpapi.reconcileChunk`. It compares a
client's expected normalized chunk identity, timestamps, normalized
`original_filename`, ciphertext byte size, and ciphertext SHA-256 against an
accepted chunk row. Matched responses return only safe identity and fingerprint
metadata; conflict responses return mismatched field names without stored
paths, object keys, uploaded bytes, plaintext, raw keys, raw tokens, request
bodies, or conflicting stored values.

After verification, `internal/storage.BlobStore.CommitTemp` commits the encrypted bytes under the server-controlled stored path:

```text
data/incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Legacy unstreamed chunks keep the older path:

```text
data/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Local storage maps that stored path under `SAFE_DATA_DIR`. Optional S3-compatible storage maps the same stored path under `SAFE_S3_PREFIX` in the configured bucket. Storage uses no-overwrite behavior, so an existing local file or final object is treated as a conflict.

Metadata is written after the file is safely committed, through the configured metadata repository. SQLite uses `internal/incidents.Repository`; PostgreSQL uses `internal/postgresdb.Repository`. Both implementations recheck incident and stream state before inserting chunk metadata so uploads that race with incident close or stream completion are rejected. The schemas enforce separate unique identities for streamed and legacy unstreamed chunks.

New clients can create a media stream with `POST /v1/incidents/{incident_id}/streams` and include the returned `stream_id` during chunk upload. Streamed chunk indexes start at `1`, and streamed chunk identity is `incident_id + stream_id + chunk_index`. Existing chunks without `stream_id` remain valid and readable as legacy chunk metadata, including older index `0` chunks; legacy unstreamed identity remains `incident_id + media_type + chunk_index`. Legacy unstreamed chunks are not included in completed-stream evidence bundles.

Stream completion is handled by `internal/httpapi.completeMediaStream`. Before a stream moves from `open` to `complete`, the handler verifies that chunks `1..expected_chunk_count` exist contiguously for that stream and that each stored blob can be opened from the configured blob store. `internal/incidents.Repository.CompleteMediaStream` then revalidates the chunk rows in the completion transaction before committing the state change. Failed streams preserve uploaded chunks but are not offered as normal downloads.

## Deletion And Retention Flow

Private owner-scoped deletion requests are handled by
`POST /v1/incidents/{incident_id}/deletion`. Admin-global deletion requests are
handled by `POST /v1/admin/incidents/{incident_id}/deletion`. Both route groups
are mounted only on the private API server. Public incident viewer routes do
not expose deletion controls or deletion status.

The configured metadata repository creates or returns one durable deletion
decision for the incident. In the same transaction, it snapshots
server-controlled chunk `stored_path` values into deletion item rows and marks
the incident deletion state as `deletion_pending`. Repeated requests return the
existing decision instead of creating competing work. Open incidents are
rejected unless the private request explicitly sets `allow_open: true`;
automatic closed-incident retention never selects open incidents.

While an incident is `deletion_pending`, `deleting`, `deletion_failed`, or
`deleted`, normal write paths fail closed in the repository. Public viewer token
lookups also fail closed with the same public error shape used for invalid,
expired, or revoked tokens, so public routes do not reveal deletion state.

`internal/retention.Worker` queues closed-incident retention decisions only
when `SAFE_CLOSED_INCIDENT_RETENTION` is positive. It can also prune
expired/revoked viewer-token metadata and completed minimal tombstones when the
corresponding retention settings are positive. It processes pending, failed, or
stale `deleting` decisions in batches, deletes encrypted blobs through
`storage.BlobStore.Remove` using only metadata-derived stored paths, treats a
missing blob as idempotent success for an existing deletion item, and records
safe retry error classes for failed blob deletions.

After every deletion item is complete or confirmed absent, the repository
prunes sensitive child metadata such as upload operations, incident tokens,
checkins, chunks, and streams, then leaves a minimal incident tombstone and
marks the deletion decision `deleted`.

## Admin Web Flow

`GET /admin` is mounted only on the private API server, outside the `/v1` API
namespace. When no admin account exists and `SAFE_AUTH_BOOTSTRAP_SECRET` is
configured, it renders a first-admin bootstrap screen. After an admin exists,
it renders an admin login screen until a valid admin web session cookie is
present. `POST /admin/login`, `POST /admin/bootstrap`, and
`POST /admin/logout` use the same account and server-side session repository
as the JSON API, with the raw session token stored in an HttpOnly
SameSite=Strict cookie scoped to `/admin`.

Authenticated admin pages list local accounts and support limited password
workflows. `POST /admin/password` changes the current admin account password
after verifying the current password, keeping the current session and revoking
other sessions. `POST /admin/accounts/{account_id}/password` resets another
local account password and revokes that account's sessions. `POST
/admin/logout` revokes the current admin web session. These authenticated
state-changing forms use a session-bound CSRF token.

The page renders `internal/httpapi/web/templates/admin.html` with Go
`html/template`. Token-neutral CSS is embedded from
`internal/httpapi/web/admin/static` and served without authentication under
`/admin/static/...`.

The admin web surface shows only safe route-boundary status, navigation stubs,
and local account-management data. It does not read incident data, expose
tokens or password hashes, expose stored paths or object keys, show uploaded
bytes, decrypt evidence, or add public dashboard behavior. Admin web responses
use no-store behavior and conservative browser security headers.

## Incident Viewer Flow

Viewer tokens are created on the authenticated private API server by
`POST /v1/incidents/{incident_id}/incident-tokens`. The raw token is returned
once, while the configured metadata repository stores only a SHA-256 hash.

`GET /i/{token}` is mounted only on the public incident viewer server. It renders `internal/httpapi/web/templates/incident_viewer.html` with `html/template`. CSS and JavaScript are embedded from `internal/httpapi/web/static`. `GET /i/{token}/data` returns the same read-only summary as JSON for polling. Pre-rename `/e/{token}` viewer, data, and download paths remain as read-only compatibility aliases for already shared links; new links should use `/i/{token}`.

Token lookup checks the hash, expiry, and revocation state before incident metadata is loaded. Invalid, expired, and revoked tokens all return the same public error. The public viewer limiter groups requests by safe route class and a hash of the socket peer identity before token lookup; limiter keys do not include raw viewer tokens or token-bearing paths. Viewer responses use `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, a strict `Content-Security-Policy`, restrictive `Permissions-Policy`, and `Cache-Control: no-store` for token-protected responses.

Completed stream bundle downloads are served by `internal/httpapi/bundles.go`. Bundles are generated on demand as ZIP responses and are not cached on disk. ZIP entry names are server-controlled, manifests are generated from database metadata, and chunk bytes are streamed from storage one file at a time. The first bundle format contains encrypted chunks and JSON manifests only; it does not decrypt, merge, or export playable media.

## Server Repository Boundary

The separate ports are a deployment boundary, not a complete security model.
Local account sessions reduce accidental unauthenticated access, but the private
API server should still stay behind localhost, LAN, WireGuard, firewall rules, or
a strict private reverse proxy.

This repository should stay focused on server/backend work:

- API handlers and routing
- SQLite migrations and repository code
- encrypted blob storage
- token-scoped incident viewer
- backend deployment docs
- backend security, retention, and threat-model docs
- simulator/reference backend flow

Before public exposure, review and add:

- the public product API and separately bound private admin API access-control
  design in [v1-access-control.md](v1-access-control.md), or a strict
  WireGuard/firewall-only deployment for the current private API
- edge rate limits, app-level public viewer limits, and broader abuse controls
- TLS and reverse-proxy settings for the public incident viewer, if reachable over a network
- deployment-specific enforcement of the documented [retention, backup, and deletion policy](retention-backup-deletion.md)
- cluster backup, restore, and failure drills for optional PostgreSQL metadata,
  S3-compatible encrypted blobs, and Valkey/Redis-compatible coordination as
  documented in
  [cluster-backup-restore-runbook.md](cluster-backup-restore-runbook.md)
- operational monitoring for failed uploads and storage/DB errors
- a production review of viewer-token sharing, expiry defaults, and revocation operations
- mode-driven access, escalation, retention, sharing, viewer, key-custody,
  trusted-contact, public product API, and broader admin/operator authorization
  design before implementing public account workflows or a separately bound
  private admin API

## Out Of Scope Today

The repository does not currently include the web client, iOS app, Android app,
protocol repository, production local recording client, mode-driven access,
escalation, retention, sharing, viewer behavior, trusted-contact accounts,
dead-man switch notifications, production client key storage, key sharing,
browser/client-side decryption, server-assisted break-glass key access,
playable media export, push notifications, SMS, Messenger integration, OAuth,
JWT, public account workflows, or a public admin dashboard. The local
desktop-recorder behavior in `cmd/simclient` is simulator/reference flow only.
