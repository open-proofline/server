# Safety Recorder Backend v0.1

Minimal Go backend for receiving already-encrypted safety recording chunks from a private client.

## Security Warning

This v0.1 server has no public authentication, no user accounts, no OAuth, and no JWT protection. Do not expose it to the public internet. Run it only on localhost or a private network until it is protected by WireGuard/firewall rules or another private access boundary.

## Requirements

- Go 1.26.3
- SQLite, via the bundled Go SQLite driver dependency
- Local disk storage for encrypted uploaded blobs

## Run Tests

```bash
go test ./...
```

## Start The Server

From the `server` directory:

```bash
go run ./cmd/api
```

Configuration is read from environment variables:

| Variable | Default |
|---|---|
| `SAFE_BIND_ADDR` | `:8080` |
| `SAFE_DATA_DIR` | `./data` |
| `SAFE_DB_PATH` | `./data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

## Create An Incident

```bash
curl -sS -X POST http://localhost:8080/v1/incidents \
  -H 'Content-Type: application/json' \
  -d '{"client_label":"iphone","notes":"test incident"}'
```

## Upload A Chunk

```bash
printf 'encrypted bytes go here' > /tmp/chunk.enc
SHA256_HEX="$(sha256sum /tmp/chunk.enc | awk '{print $1}')"
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://localhost:8080/v1/incidents/${INCIDENT_ID}/chunks" \
  -F "file=@/tmp/chunk.enc" \
  -F "chunk_index=1" \
  -F "media_type=audio" \
  -F "started_at=2026-05-21T10:00:00Z" \
  -F "ended_at=2026-05-21T10:00:10Z" \
  -F "sha256_hex=${SHA256_HEX}" \
  -F "original_filename=chunk.enc"
```

## API Summary

- `POST /v1/incidents`
- `GET /v1/incidents/{incident_id}`
- `POST /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`
- `POST /v1/incidents/{incident_id}/checkins`
- `POST /v1/incidents/{incident_id}/close`

## Next Steps

- WireGuard-only bind/firewall
- iOS client
- client-side encryption
- dead-man switch
- emergency read-only token viewer
