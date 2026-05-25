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

Run with localhost-only port publishing when everything that talks to the
backend is on the same host:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

In this shape both listeners are reachable only through the host loopback
interface. It is useful for local testing, SSH port forwarding, or a same-host
reverse proxy. It does not expose the private `/v1` API or the emergency viewer
directly to the network.

Container defaults:

| Variable | Container default |
|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `0.0.0.0:8080` |
| `SAFE_PUBLIC_BIND_ADDRS` | `0.0.0.0:8081` |
| `SAFE_DATA_DIR` | `/data` |
| `SAFE_DB_PATH` | `/data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |

Inside containers, bind to container addresses such as `0.0.0.0`, then restrict host exposure with Docker port publishing, firewall rules, WireGuard, or a reverse proxy.

## Private API Through WireGuard Or A Private Network

For a private API reachable from a WireGuard peer or private LAN, publish or
bind `/v1` only on that private interface. This example uses `10.66.0.1` as a
placeholder WireGuard interface address:

```bash
docker run --rm \
  -p 10.66.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

Only devices that can reach `10.66.0.1:8080` through the private boundary should
be able to call `/v1`. Keep host firewalls aligned with that assumption. Do not
publish `8080` on `0.0.0.0` or a public interface.

The same shape can be run without Docker by binding the private API to both
loopback and a private interface while keeping the emergency viewer local to a
same-host proxy:

```bash
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080 \
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081 \
go run ./cmd/api
```

This does not add authentication to `/v1`; it only chooses where the
unauthenticated private API listens.

## Timeout Tuning

The private API defaults keep read and write timeouts disabled so large or slow uploads and private downloads are not interrupted. The public emergency viewer has finite read/write timeouts by default, including a generous write timeout for encrypted ZIP downloads.

Reverse proxies should still set their own connection, request, and upstream
timeouts. If completed evidence bundles are large or clients are slow, tune
`SAFE_PUBLIC_WRITE_TIMEOUT` together with the reverse proxy timeout so the proxy
does not cut off an encrypted ZIP download that the Go server is still willing
to stream.

## Public Emergency Viewer Exposure

If exposing any part of the system publicly, expose only the emergency viewer listener unless `/v1` has a separate authenticated control plane in front of it.

Production-style public exposure still needs:

- TLS at the edge
- rate limiting and abuse controls
- reverse-proxy log redaction for `/e/{token}` paths
- private `/v1` access controls
- deployment-specific retention, backup, and deletion enforcement based on
  [retention-backup-deletion.md](retention-backup-deletion.md)
- operational monitoring and restore testing
- review of emergency token sharing, expiry defaults, and revocation workflows

The Go app still has no built-in app-level rate limiter. Apply rate limits at
the deployment edge for now, and tune them for the expected recording,
polling, and download patterns.

Future server-assisted break-glass or dead-man-switch key access would add
stronger operator and deployment trust requirements. It should remain disabled
unless explicitly designed and configured; see
[break-glass-key-access.md](break-glass-key-access.md).

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP. Enable HSTS at the HTTPS reverse proxy only after TLS is working for the production hostname.

After deploying the public emergency viewer over HTTPS, test the exposed origin with the MDN HTTP Observatory:

```text
https://developer.mozilla.org/en-US/observatory
```

### HTTPS Emergency Viewer With Traefik

The reverse proxy should route only the public emergency viewer listener. The
private `/v1` listener should stay on localhost, WireGuard, LAN, or another
private boundary.

One same-host shape is:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

Then configure Traefik to forward the public HTTPS hostname to
`http://127.0.0.1:8081` only. This example is documentation, not a maintained
deployment file; review it against the Traefik version you run before use:

```yaml
# traefik.yml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  file:
    filename: "/etc/traefik/dynamic/safety-recorder.yml"

certificatesResolvers:
  letsencrypt:
    acme:
      email: "admin@example.invalid"
      storage: "/var/lib/traefik/acme.json"
      httpChallenge:
        entryPoint: web

accessLog:
  format: json
  fields:
    defaultMode: keep
    names:
      RequestPath: drop
      RequestLine: drop
    headers:
      defaultMode: drop
```

```yaml
# /etc/traefik/dynamic/safety-recorder.yml
http:
  routers:
    safety-recorder-emergency:
      rule: "Host(`safety-recorder.example.invalid`)"
      entryPoints:
        - websecure
      service: safety-recorder-public
      middlewares:
        - safety-recorder-hsts
      tls:
        certResolver: letsencrypt

  services:
    safety-recorder-public:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8081"

  middlewares:
    safety-recorder-hsts:
      headers:
        stsSeconds: 31536000
        stsIncludeSubdomains: false
        stsPreload: false
```

There should be no Traefik router, service, or rule for `127.0.0.1:8080` or
`/v1`. If Traefik runs in a different container or on another host, point it at
a private address that only Traefik can reach, and keep that address off the
public internet.

Replace `admin@example.invalid` and `safety-recorder.example.invalid` with deployment-specific values before use.

### Route-Group Rate Limiting

Use different rate limits for different route groups. A single global limiter is
easy to configure, but it can either be too loose for token guessing or too
strict for legitimate bundle downloads and chunk uploads.

Suggested route groups:

| Route group | Paths | Guidance |
|---|---|---|
| Emergency page lookup | `GET /e/{token}` | Keep relatively strict because each request performs a bearer-token lookup. |
| Emergency JSON polling | `GET /e/{token}/data` | Allow normal viewer polling, but keep it lower than static assets. |
| Emergency ZIP downloads | `GET /e/{token}/streams/{stream_id}/download`, `GET /e/{token}/incident/download` | Limit download starts without cutting off long encrypted ZIP responses; coordinate with proxy and app timeouts. |
| Public static assets | `GET /static/...` | Static assets are token-neutral and can usually tolerate a looser limit. |
| Private chunk uploads | `POST /v1/incidents/{incident_id}/chunks` | If routed through a private proxy, tune for expected chunk cadence and upload retries. |
| Private incident, stream, check-in, token, and admin-style actions | Other `/v1/...` routes | Keep behind a private boundary and use limits as an abuse backstop, not as public authentication. |

Rate limiting does not make `/v1` safe to expose publicly. Keep the private API
on localhost, LAN, WireGuard, firewall rules, or a private reverse-proxy entry
point even when limits are configured.

Exact limits are deployment-specific. Start with conservative values, watch
legitimate simulator/client behavior, then adjust. Avoid sending raw
`/e/{token}` paths to metrics, dashboards, or logs while measuring limiter
behavior.

### Traefik Rate-Limiting Example

Traefik's `rateLimit` middleware uses `average`, `period`, and `burst` to
define a token-bucket limit. Review the options for the Traefik version you run,
especially the source criterion used to group requests behind proxies.

This example replaces the single broad public emergency-viewer router from the
basic example above with grouped routers for the same public service. Do not
append these routers alongside the broad router unless you have deliberately
reviewed the resulting priorities and middleware order. The numbers are
illustrative placeholders, not production defaults:

```yaml
# /etc/traefik/dynamic/safety-recorder.yml
http:
  routers:
    safety-recorder-emergency-downloads:
      rule: "Host(`safety-recorder.example.invalid`) && Method(`GET`) && PathRegexp(`^/e/[^/]+/(streams/[^/]+/download|incident/download)$`)"
      entryPoints:
        - websecure
      service: safety-recorder-public
      middlewares:
        - safety-recorder-rate-downloads
        - safety-recorder-hsts
      priority: 120
      tls:
        certResolver: letsencrypt

    safety-recorder-emergency-data:
      rule: "Host(`safety-recorder.example.invalid`) && Method(`GET`) && PathRegexp(`^/e/[^/]+/data$`)"
      entryPoints:
        - websecure
      service: safety-recorder-public
      middlewares:
        - safety-recorder-rate-data
        - safety-recorder-hsts
      priority: 110
      tls:
        certResolver: letsencrypt

    safety-recorder-emergency-page:
      rule: "Host(`safety-recorder.example.invalid`) && Method(`GET`) && PathRegexp(`^/e/[^/]+$`)"
      entryPoints:
        - websecure
      service: safety-recorder-public
      middlewares:
        - safety-recorder-rate-page
        - safety-recorder-hsts
      priority: 100
      tls:
        certResolver: letsencrypt

    safety-recorder-static:
      rule: "Host(`safety-recorder.example.invalid`) && Method(`GET`) && PathPrefix(`/static/`)"
      entryPoints:
        - websecure
      service: safety-recorder-public
      middlewares:
        - safety-recorder-rate-static
        - safety-recorder-hsts
      priority: 90
      tls:
        certResolver: letsencrypt

  services:
    safety-recorder-public:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8081"

  middlewares:
    safety-recorder-rate-page:
      rateLimit:
        average: 20
        period: 1m
        burst: 10

    safety-recorder-rate-data:
      rateLimit:
        average: 60
        period: 1m
        burst: 20

    safety-recorder-rate-downloads:
      rateLimit:
        average: 6
        period: 1m
        burst: 3

    safety-recorder-rate-static:
      rateLimit:
        average: 120
        period: 1m
        burst: 60

    safety-recorder-hsts:
      headers:
        stsSeconds: 31536000
        stsIncludeSubdomains: false
        stsPreload: false
```

If the private API is also routed through Traefik, it should use a private-only
entry point, private address, or private network. Do not attach private `/v1`
routers to public entry points. A private-only file-provider shape can split
uploads from other private actions.

Define the private entry point in Traefik's static configuration first. This
example uses `wireguard` as a placeholder entry point name and `10.66.0.1:80`
as a placeholder private HTTP interface address:

```yaml
# traefik.yml excerpt
entryPoints:
  wireguard:
    address: "10.66.0.1:80"
```

Then reference that entry point from the dynamic file-provider configuration:

```yaml
# Private-boundary example only. Do not attach these routers to public entry points.
http:
  routers:
    safety-recorder-private-uploads:
      rule: "Host(`safety-recorder-private.example.invalid`) && Method(`POST`) && PathRegexp(`^/v1/incidents/[^/]+/chunks$`)"
      entryPoints:
        - wireguard
      service: safety-recorder-private
      middlewares:
        - safety-recorder-rate-private-uploads
      priority: 110

    safety-recorder-private-api:
      rule: "Host(`safety-recorder-private.example.invalid`) && PathPrefix(`/v1/`)"
      entryPoints:
        - wireguard
      service: safety-recorder-private
      middlewares:
        - safety-recorder-rate-private-api
      priority: 100

  services:
    safety-recorder-private:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8080"

  middlewares:
    safety-recorder-rate-private-uploads:
      rateLimit:
        average: 120
        period: 1m
        burst: 60

    safety-recorder-rate-private-api:
      rateLimit:
        average: 30
        period: 1m
        burst: 15
```

When Traefik sits behind another proxy or load balancer, review forwarded-header
trust and the rate-limit source criterion. A misconfigured source can group all
clients under one proxy IP, or trust client-supplied forwarding headers too
loosely.

### Emergency Token Paths In Proxy Logs

Emergency URLs are bearer-token URLs. The Go server logs redacted route patterns
such as `/e/{token}`, but an edge proxy can still log the raw request path before
the request reaches the Go server.

For Traefik, use an access-log format that supports field controls, then review
the fields for the version you deploy and drop or sanitize request path fields.
If path redaction is unavailable in your logging format, disable access logs for
this router or pass logs through a sanitizer before storage. Redacting headers is
not enough because the token is in the URL path.

Avoid logging:

- raw `/e/{token}` paths
- query strings attached to emergency URLs
- request bodies
- uploaded bytes
- `Authorization` headers
- rate-limit keys or metric labels containing raw emergency tokens
- plaintext, raw keys, or future token-like values

### Proxy And App Timeout Coordination

Completed stream and incident downloads can be large encrypted ZIP responses.
Keep Traefik entry point, upstream, and client-response timeouts at least as
permissive as the expected download window, and review them together with
`SAFE_PUBLIC_WRITE_TIMEOUT`.

For example, if the public viewer runs with:

```bash
SAFE_PUBLIC_WRITE_TIMEOUT=10m
```

then the Traefik route serving the emergency viewer should also allow a slow
client to receive the response for roughly that long. If the proxy timeout is
shorter than the Go server timeout, downloads may fail even though the backend is
configured to keep streaming.

## GitHub Actions And GHCR

The CI workflow:

- runs Go tests from `server/`
- builds a Linux amd64 binary artifact
- generates release binary attestations from a tag-only attestation job
- builds the Docker image from `server/Dockerfile`
- publishes `ghcr.io/thesilkky/safety-recorder` on pushes to `main` and `v*` tags
- attaches attestations to published GHCR images
- keeps workflow-level token permissions read-only and grants write permissions
  only to the tag-only binary attestation and trusted Docker publish jobs
