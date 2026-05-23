# Deployment

Safety Recorder is experimental and not production-ready public infrastructure. Treat the private `/v1` API as unauthenticated admin/write access.

> **Do not expose `/v1` publicly as-is.**
>
> Keep private listeners behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. Separate bind addresses are a deployment boundary, not a complete security model.

## Local Development

From `server`:

```bash
go run ./cmd/api
```

Defaults:

| Listener | Address |
|---|---|
| Private API | `127.0.0.1:8080` |
| Public emergency viewer | `127.0.0.1:8081` |

## Docker

Build from the repository root:

```bash
docker build -t safety-recorder-backend ./server
```

Run with localhost-only port publishing:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

Container defaults:

| Variable | Container default |
|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `0.0.0.0:8080` |
| `SAFE_PUBLIC_BIND_ADDRS` | `0.0.0.0:8081` |
| `SAFE_DATA_DIR` | `/data` |
| `SAFE_DB_PATH` | `/data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

Inside containers, bind to container addresses such as `0.0.0.0`, then restrict host exposure with Docker port publishing, firewall rules, WireGuard, or a reverse proxy.

## Timeout Tuning

The private API defaults keep read and write timeouts disabled so large or slow uploads and private downloads are not interrupted. The public emergency viewer has finite read/write timeouts by default, including a generous write timeout for encrypted ZIP downloads.

Reverse proxies should still set their own connection, request, and upstream timeouts. If completed evidence bundles are large or clients are slow, tune `SAFE_PUBLIC_WRITE_TIMEOUT` together with the reverse proxy timeout.

## Public Emergency Viewer Exposure

If exposing any part of the system publicly, expose only the emergency viewer listener unless `/v1` has a separate authenticated control plane in front of it.

Production-style public exposure still needs:

- TLS at the edge
- rate limiting and abuse controls
- reverse-proxy log redaction for `/e/{token}` paths
- private `/v1` access controls
- retention, backup, and deletion policy
- operational monitoring and restore testing
- review of emergency token sharing, expiry defaults, and revocation workflows

Future server-assisted break-glass or dead-man-switch key access would add
stronger operator and deployment trust requirements. It should remain disabled
unless explicitly designed and configured; see
[break-glass-key-access.md](break-glass-key-access.md).

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP. Enable HSTS at the HTTPS reverse proxy only after TLS is working for the production hostname.

After deploying the public emergency viewer over HTTPS, test the exposed origin with the MDN HTTP Observatory:

```text
https://developer.mozilla.org/en-US/observatory
```

## GitHub Actions And GHCR

The CI workflow:

- runs Go tests from `server/`
- builds a Linux amd64 binary artifact
- builds the Docker image from `server/Dockerfile`
- publishes `ghcr.io/thesilkky/safety-recorder` on pushes to `main` and `v*` tags
