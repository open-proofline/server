# Configuration

Configuration is read from environment variables when the Proofline API starts.

## Environment Variables

| Variable | Default | Notes |
|---|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `127.0.0.1:8080` | Comma-separated private listener addresses for `/v1` and `/admin`. |
| `SAFE_PUBLIC_BIND_ADDRS` | `127.0.0.1:8081` | Comma-separated public incident viewer listener addresses. |
| `SAFE_DATA_DIR` | `./data` | Local directory for SQLite, temp uploads, and encrypted blobs unless `SAFE_DB_PATH` points elsewhere. |
| `SAFE_DB_PATH` | `./data/safety.db` | SQLite database path. The default file name still uses `safety.db` until a separate data-layout migration is performed. |
| `SAFE_METADATA_BACKEND` | `sqlite` | Metadata backend selector. Supported values are `sqlite` and `postgresql`. |
| `SAFE_BLOB_BACKEND` | `local` | Encrypted blob backend selector. Supported values are `local` and `s3`. |
| `SAFE_COORDINATION_BACKEND` | `none` | Coordination backend selector. Supported values are `none`, `valkey`, and `redis`. |
| `SAFE_POSTGRES_DSN` | unset | PostgreSQL connection string. Required when `SAFE_METADATA_BACKEND=postgresql`; treat as secret-bearing. |
| `SAFE_POSTGRES_MAX_OPEN_CONNS` | `10` | Maximum open PostgreSQL connections when the PostgreSQL metadata backend is selected. |
| `SAFE_POSTGRES_MAX_IDLE_CONNS` | `5` | Maximum idle PostgreSQL connections when the PostgreSQL metadata backend is selected. |
| `SAFE_POSTGRES_CONN_MAX_LIFETIME` | `30m` | Maximum lifetime for PostgreSQL connections. |
| `SAFE_S3_ENDPOINT` | unset | S3-compatible endpoint URL. Required when `SAFE_BLOB_BACKEND=s3`. |
| `SAFE_S3_REGION` | `us-east-1` | S3 signing region used when `SAFE_BLOB_BACKEND=s3`. |
| `SAFE_S3_BUCKET` | unset | S3 bucket for committed encrypted chunks. Required when `SAFE_BLOB_BACKEND=s3`. |
| `SAFE_S3_PREFIX` | unset | Optional server-controlled object key prefix for committed chunks. |
| `SAFE_S3_ACCESS_KEY_ID` | unset | Static S3 access key. Required when `SAFE_BLOB_BACKEND=s3`. |
| `SAFE_S3_SECRET_ACCESS_KEY` | unset | Static S3 secret access key. Required when `SAFE_BLOB_BACKEND=s3`; treat as a secret. |
| `SAFE_S3_SESSION_TOKEN` | unset | Optional static S3 session token. Requires static S3 credentials. |
| `SAFE_S3_FORCE_PATH_STYLE` | `true` | Use path-style bucket addressing for S3-compatible services. Set to `false` for virtual-hosted-style services that require it. |
| `SAFE_VALKEY_ADDR` | unset | Valkey/Redis-compatible `host:port`. Required when `SAFE_COORDINATION_BACKEND=valkey` or `redis`. |
| `SAFE_VALKEY_USERNAME` | unset | Optional Valkey ACL username. |
| `SAFE_VALKEY_PASSWORD` | unset | Optional Valkey password; treat as a secret. |
| `SAFE_VALKEY_DB` | `0` | Non-negative Valkey database number. |
| `SAFE_VALKEY_TLS` | `false` | Use TLS for the Valkey connection. |
| `SAFE_VALKEY_DIAL_TIMEOUT` | `5s` | Valkey dial timeout. |
| `SAFE_VALKEY_READ_TIMEOUT` | `5s` | Valkey read timeout. |
| `SAFE_VALKEY_WRITE_TIMEOUT` | `5s` | Valkey write timeout. |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` | Maximum encrypted file bytes per upload. |
| `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` | `24h` | Default lifetime for viewer tokens created without `expires_at`. Set to `0` to disable the default for omitted `expires_at` values. |
| `SAFE_SESSION_TTL` | `12h` | Lifetime for local account sessions created by `/v1/auth/login`. |
| `SAFE_AUTH_BOOTSTRAP_SECRET` | unset | One-time bootstrap secret required to create the first admin account when no admin exists. Remove after bootstrap. |
| `SAFE_DELETION_WORKER_INTERVAL` | `1m` | Background deletion maintenance interval. Set to `0` to disable the automatic scheduler while keeping deletion decisions durable for a later run. |
| `SAFE_CLOSED_INCIDENT_RETENTION` | `0` | Retention window for closed incidents. `0` disables automatic retention deletion; positive Go durations delete closed incidents older than the window. |
| `SAFE_TOKEN_METADATA_RETENTION` | `0` | Audit window for pruning expired or revoked viewer-token metadata. `0` disables token metadata pruning. |
| `SAFE_DELETION_TOMBSTONE_RETENTION` | `0` | Retention window for minimal deleted-incident tombstones after deletion completion. `0` disables tombstone pruning. |
| `SAFE_TEMP_UPLOAD_CLEANUP_AGE` | `0` | Minimum age for startup cleanup of orphaned local temp upload files. `0` disables cleanup. |
| `SAFE_TEMP_UPLOAD_CLEANUP_DRY_RUN` | `false` | When temp cleanup is enabled, log safe counts without deleting eligible temp files. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_ENABLED` | `true` | Enables app-level rate limiting for public incident viewer route classes. Set to `false` to disable the app-level limiter. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_WINDOW` | `1m` | Fixed-window duration for app-level public viewer limits. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_PAGE` | `60` | Public viewer page lookup requests allowed per window per hashed socket peer. Set to `0` to disable this route-class limit. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_DATA` | `300` | Public viewer JSON polling requests allowed per window per hashed socket peer. Set to `0` to disable this route-class limit. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_DOWNLOAD` | `12` | Public viewer encrypted ZIP download starts allowed per window per hashed socket peer. Set to `0` to disable this route-class limit. |
| `SAFE_PUBLIC_VIEWER_RATE_LIMIT_STATIC` | `600` | Public viewer static asset requests allowed per window per hashed socket peer. Set to `0` to disable this route-class limit. |
| `SAFE_PRIVATE_READ_HEADER_TIMEOUT` | `10s` | Private API HTTP read-header timeout. |
| `SAFE_PRIVATE_READ_TIMEOUT` | `0s` | Private API HTTP read timeout. `0` disables it for large or slow uploads. |
| `SAFE_PRIVATE_WRITE_TIMEOUT` | `0s` | Private API HTTP write timeout. `0` disables it for large or slow downloads. |
| `SAFE_PRIVATE_IDLE_TIMEOUT` | `120s` | Private API HTTP idle connection timeout. |
| `SAFE_PUBLIC_READ_HEADER_TIMEOUT` | `10s` | Public incident viewer HTTP read-header timeout. |
| `SAFE_PUBLIC_READ_TIMEOUT` | `30s` | Public incident viewer HTTP read timeout. |
| `SAFE_PUBLIC_WRITE_TIMEOUT` | `300s` | Public incident viewer HTTP write timeout for pages and ZIP downloads. |
| `SAFE_PUBLIC_IDLE_TIMEOUT` | `120s` | Public incident viewer HTTP idle connection timeout. |

The older singular variables `SAFE_PRIVATE_BIND_ADDR` and `SAFE_PUBLIC_BIND_ADDR` are still supported when the matching plural variable is unset. Plural variables take precedence.

## Backend Selection Scaffold

The backend selector variables are a startup validation scaffold for cluster support. Local-first values remain the defaults:

```bash
SAFE_METADATA_BACKEND=sqlite \
SAFE_BLOB_BACKEND=local \
SAFE_COORDINATION_BACKEND=none \
go run ./cmd/api
```

Values are matched case-insensitively after trimming surrounding whitespace. Unsupported names fail startup with a clear configuration error.

PostgreSQL metadata is implemented as an optional backend for new deployments:

```bash
SAFE_METADATA_BACKEND=postgresql \
SAFE_POSTGRES_DSN='postgres://proofline:example-password@db.example.invalid:5432/proofline?sslmode=require' \
SAFE_BLOB_BACKEND=local \
SAFE_COORDINATION_BACKEND=none \
go run ./cmd/api
```

`SAFE_POSTGRES_DSN` may contain credentials and private hostnames. Do not log it
or include it in public issues, support tickets, screenshots, shell history, or
deployment notes. `SAFE_DB_PATH` remains the SQLite database path and is ignored
by the PostgreSQL metadata backend.

S3-compatible object storage is implemented as an optional encrypted blob backend for committed chunks:

```bash
SAFE_METADATA_BACKEND=sqlite \
SAFE_BLOB_BACKEND=s3 \
SAFE_COORDINATION_BACKEND=none \
SAFE_S3_ENDPOINT=https://s3.example.invalid \
SAFE_S3_REGION=us-east-1 \
SAFE_S3_BUCKET=proofline-evidence \
SAFE_S3_PREFIX=prod/server \
SAFE_S3_ACCESS_KEY_ID=example-access-key \
SAFE_S3_SECRET_ACCESS_KEY=example-secret-key \
go run ./cmd/api
```

Valkey/Redis-compatible coordination is implemented as an optional, explicit
backend. The current server validates the configured service at startup.
Public viewer app-level rate-limit counters use the configured Valkey service
when `SAFE_COORDINATION_BACKEND=valkey` or `redis`; otherwise they use local
in-memory process counters. Current upload routes still use complete encrypted
chunk uploads and do not yet implement upload leases, resumable uploads, or
Valkey-backed cluster coordination. Complete-upload idempotency keys are stored
in the selected metadata backend, not Valkey.

`SAFE_DB_PATH` and `SAFE_DATA_DIR` keep their current behavior for the supported `sqlite` and `local` backends. When `SAFE_METADATA_BACKEND=postgresql`, `SAFE_DB_PATH` is not used for metadata. When `SAFE_BLOB_BACKEND=s3`, `SAFE_DATA_DIR/tmp` is still used for local temporary upload staging before final object writes.

PostgreSQL schema, migration, test, and restore expectations are documented in
[PostgreSQL metadata migration path](postgresql-metadata-migration.md). Initial
PostgreSQL support is for new metadata deployments only. The server does not
automatically migrate an existing SQLite database to PostgreSQL at startup.

Cluster backup, restore, and failure-mode guidance for PostgreSQL metadata,
S3-compatible encrypted blobs, and Valkey/Redis-compatible coordination is
documented in
[Cluster backup, restore, and failure runbook](cluster-backup-restore-runbook.md).

## S3-Compatible Blob Storage

The S3-compatible backend stores only opaque encrypted chunk bytes. It does not add backend decryption, raw media keys, key escrow, browser decryption, public `/v1` exposure, public account workflows, or production-readiness guarantees.

Uploads are first staged as local temp files under `SAFE_DATA_DIR/tmp` while the server enforces `SAFE_MAX_UPLOAD_BYTES` and computes SHA-256 over the uploaded ciphertext. After the client-provided hash is verified, the server writes the final object key with conditional no-overwrite behavior. The final object key is derived from server-controlled incident, stream, media type, and chunk index metadata:

```text
{SAFE_S3_PREFIX}/incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
{SAFE_S3_PREFIX}/incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

The optional prefix must be relative and must not contain empty, `.`, `..`, or backslash path segments. Client requests never provide final object keys or stored paths.

Use HTTPS for S3-compatible endpoints unless the endpoint is limited to a local
or private test network. Plain HTTP object-storage traffic can expose
credentials, session tokens, object keys, and encrypted evidence bytes to the
network path. Before enabling a provider for evidence storage, run a small
no-overwrite smoke test that confirms conditional writes reject an existing
object instead of replacing it.

This implementation does not create S3 staging objects. Failed uploads and hash mismatches clean up local temp files through the normal upload path. If the process crashes, abandoned local temp files under `SAFE_DATA_DIR/tmp` may remain and should be cleaned only by a conservative operator policy that never deletes committed objects. `SAFE_TEMP_UPLOAD_CLEANUP_AGE` applies to this local staging directory for both local and S3-compatible blob backends. Object-store lifecycle cleanup for staging prefixes is not needed unless a future resumable or multipart S3 staging design adds such prefixes.

`SAFE_S3_ACCESS_KEY_ID` and `SAFE_S3_SECRET_ACCESS_KEY` are required when the S3 backend is selected. `SAFE_S3_SESSION_TOKEN` is optional. Credentials, endpoints, bucket names, object keys, and private deployment details should not be written to public issue drafts, logs, or support tickets.

Bundle downloads continue to generate server-controlled ZIP entry names such as `chunks/audio_000001.enc`; they do not expose object-store URLs, bucket names, configured prefixes, or filesystem paths.

## Optional Valkey / Redis-Compatible Coordination

No coordination backend is used by default. To enable Valkey or another
Redis-compatible service for short-lived coordination, explicitly set the
coordination selector and connection settings:

```bash
SAFE_COORDINATION_BACKEND=valkey \
SAFE_VALKEY_ADDR=valkey.example.invalid:6379 \
SAFE_VALKEY_USERNAME=proofline \
SAFE_VALKEY_PASSWORD=example-password \
SAFE_VALKEY_TLS=true \
go run ./cmd/api
```

`SAFE_COORDINATION_BACKEND=redis` is accepted as an alias for Redis-compatible
deployments. `SAFE_VALKEY_ADDR` must be a `host:port`, not a URL, so passwords
and database numbers stay in their dedicated settings.

Treat Valkey passwords, private hostnames, private network details,
rate-limit counter keys, and any future coordination keys as private
deployment details. Do not put them in public issues, logs, dashboards,
screenshots, support tickets, or metrics labels.

Coordination is not durable evidence storage. Incident metadata and
viewer-token metadata remain in the selected metadata backend, and committed
encrypted bytes remain in the selected blob backend. If a configured Valkey
backend cannot be checked at startup, the server fails closed instead of
silently running with a misleading cluster configuration.

The current implementation stores only short-lived public viewer rate-limit
counters in Valkey when coordination is configured. Those keys are
server-controlled route-class keys using a hash of the socket peer identity;
they do not include raw `/i/{token}` paths, legacy `/e/{token}` paths, raw
viewer tokens, request bodies, Authorization headers, uploaded bytes,
plaintext, raw keys, or private deployment details. The current implementation
does not store upload leases or idempotency results in Valkey. Future
upload-operation work must keep Valkey keys server-controlled and must not
include raw viewer tokens, incident tokens, request bodies, uploaded bytes,
plaintext, raw keys, private deployment details, raw idempotency keys, or user
safety data.

## Bind Address Lists

`SAFE_PRIVATE_BIND_ADDRS` and `SAFE_PUBLIC_BIND_ADDRS` are comma-separated `host:port` lists.

Empty entries are rejected. These values fail startup:

```text
,
127.0.0.1:8080,,10.66.0.1:8080
```

Example:

```bash
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080 \
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081 \
go run ./cmd/api
```

## Upload Size Limits

`SAFE_MAX_UPLOAD_BYTES` accepts a positive byte count or binary unit suffix:

- `B`
- `K` / `KB`
- `M` / `MB`
- `G` / `GB`

Fractional unit values are allowed when they resolve to at least one byte, for example `0.5KB`. Non-positive, sub-byte, invalid, and oversized values are rejected during startup.

## Viewer Token Expiry

Viewer tokens created without an explicit `expires_at` default to expiring after `SAFE_DEFAULT_INCIDENT_TOKEN_TTL`, which is `24h` unless configured otherwise.

The value uses Go duration strings such as `12h` or `168h`.

Set `SAFE_DEFAULT_INCIDENT_TOKEN_TTL=0` only when you deliberately want omitted `expires_at` values to create tokens that remain valid until revoked.

## Local Account Sessions

The private `/v1` API requires local account sessions. Sessions created by
`POST /v1/auth/login` expire after `SAFE_SESSION_TTL`, which defaults to `12h`.
The private `/admin` browser flow uses the same session store and TTL, with the
raw session token held in an HttpOnly SameSite cookie scoped to `/admin`. The
value uses Go duration strings such as `6h` or `30m`.

For a new metadata database, startup fails until an admin account exists unless
`SAFE_AUTH_BOOTSTRAP_SECRET` is set. Use that secret only long enough to call
`POST /v1/bootstrap/admin` or create the first admin through `/admin`, then
remove it from the environment and restart.
Treat the bootstrap secret, account passwords, session tokens, raw
idempotency keys, and Authorization headers as secrets. They must not appear in
public issues, logs, dashboards, screenshots, support tickets, or shell
history.

## Deletion And Retention

The server starts a background deletion worker by default. The worker processes
durable incident deletion decisions created through private owner-scoped or
admin routes, deletes encrypted blobs by server-controlled stored paths from
metadata, prunes sensitive child metadata after blob deletion, and leaves a
minimal tombstone.

```bash
SAFE_DELETION_WORKER_INTERVAL=30s \
go run ./cmd/api
```

Set `SAFE_DELETION_WORKER_INTERVAL=0` to disable the automatic scheduler. This
does not delete or discard pending deletion decisions; a later process run with
the worker enabled can resume them.

Closed-incident retention is disabled by default. To queue deletion decisions
for closed incidents older than a configured window, set a positive duration:

```bash
SAFE_CLOSED_INCIDENT_RETENTION=720h \
go run ./cmd/api
```

Open incidents are not selected by automatic retention. Deleting an open
incident requires an explicit private deletion request with `allow_open: true`.
Mode-specific retention windows and backup expiry are not configured by these
variables.

Expired or revoked viewer-token metadata pruning is disabled by default. Set a
positive audit window only after reviewing whether token labels and token-hash
metadata must remain available for operational review:

```bash
SAFE_TOKEN_METADATA_RETENTION=168h \
go run ./cmd/api
```

Token metadata pruning removes only incident-token rows whose `expires_at` or
`revoked_at` timestamp is older than the configured window. It does not delete
incidents, streams, chunks, checkins, blobs, backups, or raw tokens. Raw viewer
tokens are not stored.

Deleted-incident tombstone pruning is also disabled by default:

```bash
SAFE_DELETION_TOMBSTONE_RETENTION=2160h \
go run ./cmd/api
```

Tombstone pruning removes only completed minimal tombstones after deletion retry
state is no longer needed and no sensitive child metadata remains. Backup
expiry, restore reconciliation, object-store versions, filesystem snapshots,
and downloaded bundles remain deployment responsibilities.

## Orphan Temp Upload Cleanup

Temp upload cleanup is disabled by default. To clean up abandoned local upload
staging files after a crash, set a positive age threshold and restart the
server:

```bash
SAFE_TEMP_UPLOAD_CLEANUP_AGE=24h \
go run ./cmd/api
```

Only regular files whose names match the server's `upload-*` temp-upload
pattern under `SAFE_DATA_DIR/tmp` are eligible. Active files newer than the
configured age are skipped. Directories, symlinks, unrelated temp files,
committed chunk blobs, stored object keys, SQLite or PostgreSQL metadata, and
evidence bundle contents are never cleanup targets.

To preview safe counts without deleting files:

```bash
SAFE_TEMP_UPLOAD_CLEANUP_AGE=24h \
SAFE_TEMP_UPLOAD_CLEANUP_DRY_RUN=true \
go run ./cmd/api
```

Cleanup logs only counts such as scanned, eligible, removed, skipped, and error
totals. Logs must not include temp paths, committed stored paths, object keys,
request bodies, uploaded bytes, raw tokens, plaintext, raw keys, or private
deployment details.

## HTTP Timeouts

Timeout values use Go duration strings such as `10s`, `30s`, or `5m`. `0` and `0s` disable a timeout.

Private read and write timeouts default to disabled so slow chunk uploads and private downloads are not accidentally cut off. Public viewer requests use more defensive defaults because public routes are read-only and do not accept upload bodies. Large public ZIP downloads may require increasing `SAFE_PUBLIC_WRITE_TIMEOUT`.

## Data Directory Layout

By default:

```text
data/
  safety.db
  safety.db-wal
  safety.db-shm
  tmp/
  incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

The `safety.db-wal` and `safety.db-shm` sidecar files appear while SQLite is
running in WAL mode. Keep them on the same local filesystem as the main
database and include them when making a direct live copy. See
[SQLite WAL operations](deployment.md#sqlite-wal-operations) for deployment,
backup, restore, and checkpoint-pressure guidance.

Uploaded chunks are staged in `tmp/`, hashed while streaming, and hard-linked into the final incident path only after SHA-256 verification. New streamed uploads use the stream-scoped path. Legacy unstreamed chunks keep the older incident-level path. Stored chunk paths are relative server-controlled paths, not client-provided paths.

SQLite schema changes are tracked in a `schema_migrations` table in the configured SQLite database. PostgreSQL schema changes use a separate PostgreSQL migration path and `schema_migrations` table in the configured PostgreSQL database.

With `SAFE_BLOB_BACKEND=s3`, committed encrypted chunks use the same stored path values in SQLite, but those values are resolved to S3 object keys under `SAFE_S3_PREFIX` instead of local files under `SAFE_DATA_DIR/incidents`.
