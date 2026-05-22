# Safety Recorder Backend v0.2.1

![CI](https://github.com/thesilkky/safety-recorder/actions/workflows/ci.yml/badge.svg)

Safety Recorder is a private personal-safety recording system.

This repository currently contains the Go backend only. The intended client is a future iOS app that records audio/video in short chunks, encrypts them locally, and uploads them continuously so already-uploaded evidence is retained if the phone is lost, damaged, powered off, or taken.

## Current scope

v0.2.1 implements the backend ingest and emergency read-only viewing layer:

- create incidents
- receive already-encrypted recording chunks
- verify uploaded chunks using SHA-256
- store chunks immutably on local disk
- store incident, chunk, and check-in metadata in SQLite
- create scoped emergency viewer tokens
- serve a simple read-only emergency incident page
- run a small CLI simulator for incident upload/check-in flows

The backend does **not** currently implement recording, client-side encryption, an iOS app, push notifications, SMS, Messenger integration, user accounts, or a public admin dashboard.

## Intended future design

The planned system is:

```text
iOS app
  → WireGuard/private network
      → private backend write API on `SAFE_PRIVATE_BIND_ADDRS`

trusted contact
  → HTTPS emergency viewer link
      → public emergency viewer on `SAFE_PUBLIC_BIND_ADDRS`
```

## Security Warning

The v0.2.1 API binary starts separate listener groups: private `/v1` write/admin listeners and public-shaped emergency viewer listeners. Separate bind addresses are a deployment boundary, not a complete security model. The private server has no public user authentication, no user accounts, no OAuth, and no JWT protection, so it must stay behind localhost, WireGuard, a firewall, or a strict reverse proxy.

Do not treat this as production-ready public infrastructure. Public deployment still needs TLS, rate limiting, access control for `/v1`, operational logging review, retention policy, and proxy hardening.

## Requirements

- Go 1.26.3
- SQLite, via the bundled Go SQLite driver dependency
- Local disk storage for encrypted uploaded blobs

## Run Tests

From the `server` directory:

```bash
go test ./...
```

## CI/CD

GitHub Actions runs CI on pull requests and pushes. The workflow:

- runs Go tests from `server/`
- builds a Linux amd64 binary artifact
- builds the Docker image from `server/Dockerfile`
- publishes `ghcr.io/thesilkky/safety-recorder` on pushes to `main` and `v*` tags

## Start The Server

From the `server` directory:

```bash
go run ./cmd/api
```

By default this starts:

- private API server: `127.0.0.1:8080`
- public emergency viewer server: `127.0.0.1:8081`

Configuration is read from environment variables:

| Variable | Default |
|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `127.0.0.1:8080` |
| `SAFE_PUBLIC_BIND_ADDRS` | `127.0.0.1:8081` |
| `SAFE_DATA_DIR` | `./data` |
| `SAFE_DB_PATH` | `./data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

`SAFE_PRIVATE_BIND_ADDRS` and `SAFE_PUBLIC_BIND_ADDRS` are comma-separated `host:port` lists. Empty entries are rejected, so values like `,` or `127.0.0.1:8080,,10.66.0.1:8080` fail configuration loading. The older singular variables `SAFE_PRIVATE_BIND_ADDR` and `SAFE_PUBLIC_BIND_ADDR` are still supported when the matching plural variable is unset; plural variables take precedence.

Example:

```bash
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080 \
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081 \
go run ./cmd/api
```

## Simulate An Incident

Run the backend, then in another terminal from the `server` directory:

```bash
go run ./cmd/simclient --chunks 12 --interval 5s
```

Open the printed emergency viewer URL to watch incident metadata update.

To test upload failure/retry behavior:

```bash
go run ./cmd/simclient --chunks 12 --interval 2s --simulate-failure-every 4
```

## Run With Docker

Build from the repository root:

```bash
docker build -t safety-recorder-backend ./server
```

Run with a named volume for SQLite and uploaded blobs:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

The container defaults are:

| Variable | Container default |
|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `0.0.0.0:8080` |
| `SAFE_PUBLIC_BIND_ADDRS` | `0.0.0.0:8081` |
| `SAFE_DATA_DIR` | `/data` |
| `SAFE_DB_PATH` | `/data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

Inside containers, binding directly to host IP addresses may not work unless using host networking. Bind inside the container, usually to `0.0.0.0`, then restrict exposure with Docker port publishing, firewall rules, WireGuard, or a reverse proxy. The example `docker run` command publishes both ports on localhost. Keep the private API port behind WireGuard, a firewall, or an equivalent private boundary.

## Data Directory Layout

By default the server writes under `./data`:

```text
data/
  safety.db
  tmp/
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Uploads are staged in `tmp/`, hashed while streaming, then hard-linked into the final incident path without overwriting existing files.

## Create An Incident

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/incidents \
  -H 'Content-Type: application/json' \
  -d '{"client_label":"iphone","notes":"test incident"}'
```

## Upload A Chunk

```bash
printf 'encrypted bytes go here' > /tmp/chunk.enc
SHA256_HEX="$(sha256sum /tmp/chunk.enc | awk '{print $1}')"
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/chunks" \
  -F "file=@/tmp/chunk.enc" \
  -F "chunk_index=1" \
  -F "media_type=audio" \
  -F "started_at=2026-05-21T10:00:00Z" \
  -F "ended_at=2026-05-21T10:00:10Z" \
  -F "sha256_hex=${SHA256_HEX}" \
  -F "original_filename=chunk.enc"
```

## Emergency Viewer

Emergency viewer tokens are scoped to one incident, stored only as SHA-256 hashes, and can expire or be revoked. The raw token is returned only when created. Emergency responses set `Referrer-Policy: no-referrer` and `Cache-Control: no-store`.

Create a token:

```bash
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/emergency-tokens" \
  -H 'Content-Type: application/json' \
  -d '{"label":"trusted contact","expires_at":"2030-01-01T00:00:00Z"}'
```

Open the emergency page:

```text
http://127.0.0.1:8081/e/{token_from_create_response}
```

Revoke a token:

```bash
TOKEN_ID="etk_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/emergency-tokens/${TOKEN_ID}/revoke"
```

## API Summary

Private API server:

- `POST /v1/incidents`
- `GET /v1/incidents/{incident_id}`
- `POST /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`
- `POST /v1/incidents/{incident_id}/checkins`
- `POST /v1/incidents/{incident_id}/close`
- `POST /v1/incidents/{incident_id}/emergency-tokens`
- `POST /v1/emergency-tokens/{token_id}/revoke`

Public emergency viewer server:

- `GET /e/{token}`
- `GET /e/{token}/data`

See [docs/api.md](docs/api.md) for request and response examples.
See [docs/threat-model.md](docs/threat-model.md) for current security assumptions and limitations.

## Next Steps

- WireGuard-only bind/firewall
- iOS client
- client-side encryption
- dead-man switch
- reverse proxy/TLS hardening for the emergency viewer
- public deployment hardening and `/v1` access control
