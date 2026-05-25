# Configuration

Configuration is read from environment variables when the API starts.

## Environment Variables

| Variable | Default | Notes |
|---|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `127.0.0.1:8080` | Comma-separated private `/v1` listener addresses. |
| `SAFE_PUBLIC_BIND_ADDRS` | `127.0.0.1:8081` | Comma-separated public emergency viewer listener addresses. |
| `SAFE_DATA_DIR` | `./data` | Local directory for SQLite, temp uploads, and encrypted blobs unless `SAFE_DB_PATH` points elsewhere. |
| `SAFE_DB_PATH` | `./data/safety.db` | SQLite database path. |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` | Maximum encrypted file bytes per upload. |
| `SAFE_DEFAULT_EMERGENCY_TOKEN_TTL` | `24h` | Default lifetime for emergency tokens created without `expires_at`. Set to `0` to disable the default for omitted `expires_at` values. |
| `SAFE_PRIVATE_READ_HEADER_TIMEOUT` | `10s` | Private API HTTP read-header timeout. |
| `SAFE_PRIVATE_READ_TIMEOUT` | `0s` | Private API HTTP read timeout. `0` disables it for large or slow uploads. |
| `SAFE_PRIVATE_WRITE_TIMEOUT` | `0s` | Private API HTTP write timeout. `0` disables it for large or slow downloads. |
| `SAFE_PRIVATE_IDLE_TIMEOUT` | `120s` | Private API HTTP idle connection timeout. |
| `SAFE_PUBLIC_READ_HEADER_TIMEOUT` | `10s` | Public emergency viewer HTTP read-header timeout. |
| `SAFE_PUBLIC_READ_TIMEOUT` | `30s` | Public emergency viewer HTTP read timeout. |
| `SAFE_PUBLIC_WRITE_TIMEOUT` | `300s` | Public emergency viewer HTTP write timeout for pages and ZIP downloads. |
| `SAFE_PUBLIC_IDLE_TIMEOUT` | `120s` | Public emergency viewer HTTP idle connection timeout. |

The older singular variables `SAFE_PRIVATE_BIND_ADDR` and `SAFE_PUBLIC_BIND_ADDR` are still supported when the matching plural variable is unset. Plural variables take precedence.

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

## Emergency Token Expiry

Emergency tokens created without an explicit `expires_at` default to expiring after `SAFE_DEFAULT_EMERGENCY_TOKEN_TTL`, which is `24h` unless configured otherwise. The value uses Go duration strings such as `12h` or `168h`.

Set `SAFE_DEFAULT_EMERGENCY_TOKEN_TTL=0` only when you deliberately want omitted `expires_at` values to create tokens that remain valid until revoked.

## HTTP Timeouts

Timeout values use Go duration strings such as `10s`, `30s`, or `5m`. `0` and `0s` disable a timeout.

Private read and write timeouts default to disabled so slow chunk uploads and private downloads are not accidentally cut off. Public viewer requests use more defensive defaults because public routes are read-only and do not accept upload bodies. Large public ZIP downloads may require increasing `SAFE_PUBLIC_WRITE_TIMEOUT`.

## Data Directory Layout

By default:

```text
data/
  safety.db
  tmp/
  incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Uploaded chunks are staged in `tmp/`, hashed while streaming, and hard-linked into the final incident path only after SHA-256 verification. New streamed uploads use the stream-scoped path. Legacy unstreamed chunks keep the older incident-level path. Stored chunk paths are relative server-controlled paths, not client-provided paths.

SQLite schema changes are tracked in a `schema_migrations` table in the configured database.
