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
- group chunks into media streams that can be marked complete or failed
- download completed encrypted stream and incident evidence bundles
- create scoped emergency viewer tokens
- serve a simple read-only emergency incident page with completed-stream download buttons
- run a small CLI simulator for incident upload/check-in flows

Evidence bundles are ZIP files containing encrypted chunks plus JSON manifests. They are not decrypted, playable, or merged media exports.

The backend does **not** currently implement recording, client-side encryption, decryption, playable media export, an iOS app, push notifications, SMS, Messenger integration, user accounts, or a public admin dashboard.

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

`SAFE_MAX_UPLOAD_BYTES` accepts a positive byte count or a binary unit suffix: `B`, `K`/`KB`, `M`/`MB`, or `G`/`GB`. Fractional unit values are allowed when they resolve to at least one byte, for example `0.5KB`. Non-positive, sub-byte, invalid, and oversized values are rejected during startup.

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

Open the printed emergency viewer URL to watch incident metadata update. The simulator now creates a media stream, uploads chunks with `stream_id`, and completes the stream by default.

To also test encrypted bundle download through the emergency viewer:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

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

## Create A Media Stream

New clients should create a media stream before uploading chunks and include the returned `stream.id` as `stream_id` on each chunk upload.

```bash
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/streams" \
  -H 'Content-Type: application/json' \
  -d '{"media_type":"audio","label":"main audio recording"}'
```

## Upload A Chunk

```bash
printf 'encrypted bytes go here' > /tmp/chunk.enc
SHA256_HEX="$(sha256sum /tmp/chunk.enc | awk '{print $1}')"
INCIDENT_ID="inc_replace_me"
STREAM_ID="str_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/chunks" \
  -F "file=@/tmp/chunk.enc" \
  -F "stream_id=${STREAM_ID}" \
  -F "chunk_index=1" \
  -F "media_type=audio" \
  -F "started_at=2026-05-21T10:00:00Z" \
  -F "ended_at=2026-05-21T10:00:10Z" \
  -F "sha256_hex=${SHA256_HEX}" \
  -F "original_filename=chunk.enc"
```

`stream_id` is optional for backwards compatibility. Chunks without a stream remain stored and listed as legacy chunk metadata, but they are not included in completed-stream bundle downloads.

The current chunk identity remains `(incident_id, media_type, chunk_index)`, so clients should keep chunk indexes unique per incident and media type even when using streams.

## Complete A Stream

When all chunks for a stream are uploaded, mark the stream complete. The backend verifies chunks `1..expected_chunk_count` exist contiguously and that their stored files are readable before enabling downloads.

```bash
curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/streams/${STREAM_ID}/complete" \
  -H 'Content-Type: application/json' \
  -d '{"expected_chunk_count":1}'
```

Download the encrypted ZIP bundle from the private API:

```bash
curl -OJ "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/streams/${STREAM_ID}/download"
```

## Emergency Viewer

Emergency viewer tokens are scoped to one incident, stored only as SHA-256 hashes, and can expire or be revoked. The raw token is returned only when created. Emergency responses use strict browser security headers and `Cache-Control: no-store` for token-protected content.

Create a token:

```bash
INCIDENT_ID="inc_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/incidents/${INCIDENT_ID}/emergency-tokens" \
  -H 'Content-Type: application/json' \
  -d '{"label":"trusted contact","expires_at":"2030-01-01T00:00:00Z"}'
```

Open the emergency page. Completed streams show encrypted bundle download buttons:

```text
http://127.0.0.1:8081/e/{token_from_create_response}
```

Revoke a token:

```bash
TOKEN_ID="etk_replace_me"

curl -sS -X POST "http://127.0.0.1:8080/v1/emergency-tokens/${TOKEN_ID}/revoke"
```

## Web Security Headers

The Go app sets browser-facing headers for the emergency viewer:

- `Content-Security-Policy: default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `X-Frame-Options: DENY`
- `Cache-Control: no-store` on token-protected pages, JSON, errors, private JSON, private chunk reads, and bundle downloads

ZIP bundle downloads also set `Content-Type: application/zip` and `Content-Disposition: attachment`.

The app does not enable `Strict-Transport-Security` by default because local development is plain HTTP. Enable HSTS only at the HTTPS reverse proxy or deployment edge after TLS is working for the production hostname. Production public exposure should also add TLS, rate limiting, reverse-proxy log redaction for `/e/{token}` paths, and private `/v1` access controls.

After deploying the public emergency viewer over HTTPS, test the exposed origin with the MDN HTTP Observatory:

```text
https://developer.mozilla.org/en-US/observatory
```

## API Summary

Private API server:

- `POST /v1/incidents`
- `GET /v1/incidents/{incident_id}`
- `GET /v1/incidents/{incident_id}/download`
- `POST /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks`
- `GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}`
- `POST /v1/incidents/{incident_id}/streams`
- `GET /v1/incidents/{incident_id}/streams`
- `GET /v1/incidents/{incident_id}/streams/{stream_id}`
- `POST /v1/incidents/{incident_id}/streams/{stream_id}/complete`
- `POST /v1/incidents/{incident_id}/streams/{stream_id}/fail`
- `GET /v1/incidents/{incident_id}/streams/{stream_id}/download`
- `POST /v1/incidents/{incident_id}/checkins`
- `POST /v1/incidents/{incident_id}/close`
- `POST /v1/incidents/{incident_id}/emergency-tokens`
- `POST /v1/emergency-tokens/{token_id}/revoke`

Public emergency viewer server:

- `GET /e/{token}`
- `GET /e/{token}/data`
- `GET /e/{token}/streams/{stream_id}/download`
- `GET /e/{token}/incident/download`

See [docs/api.md](docs/api.md) for request and response examples.
See [docs/threat-model.md](docs/threat-model.md) for current security assumptions and limitations.

## Next Steps

- WireGuard-only bind/firewall
- iOS client
- client-side encryption
- client-side decryption and key sharing
- playable media export
- dead-man switch
- reverse proxy/TLS hardening for the emergency viewer
- public deployment hardening and `/v1` access control
