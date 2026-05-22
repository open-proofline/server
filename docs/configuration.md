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

## Data Directory Layout

By default:

```text
data/
  safety.db
  tmp/
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Uploaded chunks are staged in `tmp/`, hashed while streaming, and hard-linked into the final incident path only after SHA-256 verification. Stored chunk paths are relative server-controlled paths, not client-provided paths.
