# Safety Recorder Backend v0.1

Minimal Go backend for receiving already-encrypted safety recording chunks from a private client.

## Security Warning

This v0.1 server has no public user authentication, no user accounts, no OAuth, and no JWT protection. The `/e/{token}` emergency viewer is token-gated and read-only, but the private `/v1` write endpoints must still not be exposed casually. Public deployment needs TLS, rate limiting, logging review, and careful firewall/reverse proxy configuration.

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

## Emergency Viewer

Emergency viewer tokens are scoped to one incident, stored only as SHA-256 hashes, and can expire or be revoked. The raw token is returned only when created.

Create a token:

```bash
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://localhost:8080/v1/incidents/${INCIDENT_ID}/emergency-tokens" \
  -H 'Content-Type: application/json' \
  -d '{"label":"trusted contact","expires_at":"2030-01-01T00:00:00Z"}'
```

Open the emergency page:

```text
http://localhost:8080/e/{token_from_create_response}
```

Revoke a token:

```bash
TOKEN_ID="etk_replace_me"

curl -sS -X POST "http://localhost:8080/v1/emergency-tokens/${TOKEN_ID}/revoke"
```

## API Summary

- `POST /v1/incidents`
- `GET /v1/incidents/{incident_id}`
- `POST /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`
- `POST /v1/incidents/{incident_id}/checkins`
- `POST /v1/incidents/{incident_id}/close`
- `POST /v1/incidents/{incident_id}/emergency-tokens`
- `POST /v1/emergency-tokens/{token_id}/revoke`
- `GET /e/{token}`
- `GET /e/{token}/data`

## Next Steps

- WireGuard-only bind/firewall
- iOS client
- client-side encryption
- dead-man switch
- public deployment hardening for the emergency viewer
