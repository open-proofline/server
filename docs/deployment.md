# Deployment

Proofline is experimental and not production-ready public infrastructure. Treat the private `/v1` API as an authenticated but still private admin/write control plane.

> **Do not expose `/v1` publicly as-is.**
>
> Keep private listeners behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. Separate bind addresses are a deployment boundary, not a complete security model.

The `/v1` access-control direction is documented in
[v1-access-control.md](v1-access-control.md). Current local account sessions
do not change the deployment rule: `/v1` routes must remain private. Future
admin/operator routes should use their own private listener that can be bound
to loopback, LAN, WireGuard, VPN, firewall, or a private reverse proxy, but
that private placement must not replace admin authentication.

The current module and artifact names use the `open-proofline/server` repository namespace. The published GHCR image is `ghcr.io/open-proofline/server`, local examples use the `proofline-server` image name, and release binaries use `proofline-server-*` names. Compatibility identifiers such as the v1 encryption envelope scheme and default SQLite filename may still use earlier `safety-recorder` names until separate protocol or data-layout migrations are explicitly performed.

## Local Development

From the repository root:

```bash
SAFE_AUTH_BOOTSTRAP_SECRET='replace-with-local-bootstrap-secret' \
go run ./cmd/api
```

Defaults:

| Listener | Address |
|---|---|
| Private API | `127.0.0.1:8080` |
| Public incident viewer | `127.0.0.1:8081` |

The server fails closed until an admin account exists. For a new local
database, create the first admin while `SAFE_AUTH_BOOTSTRAP_SECRET` is set:

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/bootstrap/admin \
  -H 'Content-Type: application/json' \
  -H 'X-Proofline-Bootstrap-Secret: replace-with-local-bootstrap-secret' \
  -d '{"username":"admin","password":"replace-with-a-long-local-password"}'
```

After bootstrap, remove `SAFE_AUTH_BOOTSTRAP_SECRET` and restart. The
bootstrap route is disabled after an admin account exists. Treat the bootstrap
secret, account passwords, raw session tokens, raw idempotency keys, and
Authorization headers as secrets.

The private listener also exposes unauthenticated liveness and readiness
checks for local operators:

```bash
curl -fsS http://127.0.0.1:8080/v1/health/live
curl -fsS http://127.0.0.1:8080/v1/health/ready
```

`/v1/health/live` checks only that the process is serving requests.
`/v1/health/ready` checks the selected metadata, blob, and coordination
backends and returns only coarse backend type and `ok` or `unavailable`
statuses. It does not expose DSNs, credentials, bucket names, object keys,
stored paths, local filesystem paths, private hostnames, tokens, request
bodies, uploaded bytes, raw idempotency keys, plaintext, raw keys, or private
deployment details.
Keep these routes on the private listener; they do not make `/v1` safe for
public exposure.

The deletion worker starts automatically by default and processes durable
incident deletion decisions every minute. Set
`SAFE_DELETION_WORKER_INTERVAL=0` only when an operator intentionally wants to
pause deletion processing. Closed-incident retention is disabled by default;
set `SAFE_CLOSED_INCIDENT_RETENTION` to a positive duration only after the
deployment has reviewed backup expiry and restore implications.

The same private listener serves the admin web interface at:

```text
http://127.0.0.1:8080/admin
```

When no admin exists and `SAFE_AUTH_BOOTSTRAP_SECRET` is set, `/admin` shows a
first-admin bootstrap screen. After an admin exists, it shows an admin login
screen and stores the resulting admin web session in an HttpOnly SameSite
cookie scoped to `/admin`. Authenticated admin pages list local accounts and
provide logout, password-change, and account password-reset forms with CSRF
checks. The CSS under `/admin/static/...` is unauthenticated because it is
token-neutral static source, but the admin pages and form handlers remain
private-listener routes.

This is not a public admin dashboard. Do not expose `/admin`, `/admin/...`, or
`/v1` outside the private boundary.

## Docker

Build from the repository root:

```bash
docker build -t proofline-server .
```

Run with localhost-only port publishing when everything that talks to the backend is on the same host:

```bash
docker run --rm \
  -e SAFE_AUTH_BOOTSTRAP_SECRET='replace-with-local-bootstrap-secret' \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v proofline-server-data:/data \
  proofline-server
```

Create the first admin account through `POST /v1/bootstrap/admin`, then restart
without `SAFE_AUTH_BOOTSTRAP_SECRET`.

From the host, Docker deployments can use the private readiness route through
the loopback-published private port:

```bash
curl -fsS http://127.0.0.1:8080/v1/health/ready
```

Do not publish or proxy the private health routes on the public incident viewer
origin. They are intended for local Docker checks, private reverse-proxy
upstream checks, and operator troubleshooting inside the private boundary.

In this shape both listeners are reachable only through the host loopback interface. It is useful for local testing, SSH port forwarding, or a same-host reverse proxy. It does not expose the private `/v1` API or the incident viewer directly to the network.

Container defaults:

| Variable | Container default |
|---|---|
| `SAFE_PRIVATE_BIND_ADDRS` | `0.0.0.0:8080` |
| `SAFE_PUBLIC_BIND_ADDRS` | `0.0.0.0:8081` |
| `SAFE_DATA_DIR` | `/data` |
| `SAFE_DB_PATH` | `/data/safety.db` |
| `SAFE_MAX_UPLOAD_BYTES` | `250MB` |
| `SAFE_DELETION_WORKER_INTERVAL` | `1m` |
| `SAFE_CLOSED_INCIDENT_RETENTION` | `0` |

Inside containers, bind to container addresses such as `0.0.0.0`, then restrict host exposure with Docker port publishing, firewall rules, WireGuard, or a reverse proxy.

## SQLite WAL Operations

SQLite metadata remains the default backend. At startup, the server enables
foreign-key enforcement and verifies that SQLite accepted WAL journal mode.
This is a local-disk deployment shape, not a cluster database mode.

For SQLite deployments, `SAFE_DB_PATH` is the main database file. The default
path is `./data/safety.db` locally and `/data/safety.db` in the container. The
default file name still uses `safety.db` until a separate data-layout migration
is explicitly designed.

While the server is running in WAL mode, SQLite may also create sidecar files
next to the database:

```text
<SAFE_DB_PATH>-wal
<SAFE_DB_PATH>-shm
```

Keep the main database file and these sidecar files on the same local host,
local filesystem, and durable volume. Avoid network filesystems, unusual
shared volumes, or backup agents that cannot preserve SQLite locking,
shared-memory, and snapshot behavior correctly. If a deployment uses a bind
mount, virtualized volume, or storage layer with non-standard filesystem
semantics, test startup, upload, stream completion, bundle download, restart,
backup, and restore before relying on it for real evidence.

For backups, prefer one of the consistency strategies in
[retention, backup, and deletion](retention-backup-deletion.md): stop the API
process, take an atomic filesystem or volume snapshot that includes SQLite and
encrypted blobs together, or use SQLite's backup mechanism while coordinating
with a paused blob snapshot. Do not copy only the main `safety.db` file from a
running WAL-mode database and assume it is complete.

Growing deployments should watch for WAL/checkpoint pressure. Useful symptoms
include a `*-wal` file that keeps growing, low free space on the database
volume, rising write latency, repeated database busy/locked errors, or restore
tests that cannot reconstruct expected bundles from the database and encrypted
blobs.

Simple local checks can inspect file sizes and free space without exposing
incident contents:

```bash
db=${SAFE_DB_PATH:-./data/safety.db}
ls -lh "$db" "$db-wal" "$db-shm" 2>/dev/null || true
df -h "$(dirname "$db")"
```

Treat deployment paths, hostnames, screenshots, logs, and backup locations as
private operational details. Do not paste raw viewer tokens, request bodies,
uploaded bytes, raw idempotency keys, plaintext, raw keys, credentials, private
deployment details, or real user safety data into public issues or support
channels. If code-level
SQLite observability or automated checkpoint tuning is needed later, handle it
as a separate scoped implementation task with tests.

## Optional S3-Compatible Blob Storage

Local filesystem encrypted blob storage remains the default. To store committed encrypted chunks in an S3-compatible object store, explicitly set `SAFE_BLOB_BACKEND=s3` and configure the S3 endpoint and bucket:

```bash
SAFE_BLOB_BACKEND=s3 \
SAFE_S3_ENDPOINT=https://s3.example.invalid \
SAFE_S3_REGION=us-east-1 \
SAFE_S3_BUCKET=proofline-evidence \
SAFE_S3_PREFIX=prod/server \
SAFE_S3_ACCESS_KEY_ID=example-access-key \
SAFE_S3_SECRET_ACCESS_KEY=example-secret-key \
go run ./cmd/api
```

The S3 backend requires `SAFE_S3_ACCESS_KEY_ID` and `SAFE_S3_SECRET_ACCESS_KEY`. `SAFE_S3_SESSION_TOKEN` is optional. Treat static credentials, bucket names, private endpoints, and deployment-specific prefixes as private deployment details.

S3-compatible storage stores opaque encrypted chunk bytes only. It does not add backend decryption, key escrow, public `/v1` exposure, public account workflows, cloud deployment automation, or production readiness. Uploads still stage local temp files under `SAFE_DATA_DIR/tmp` before a final conditional object write, so the deployment must preserve enough local temp space for in-flight uploads and must include conservative cleanup for abandoned temp files after crashes.

Use HTTPS for S3-compatible endpoints unless the endpoint is reachable only on a
local or private test network. Before storing real evidence, verify the selected
provider honors conditional no-overwrite object writes by rejecting a second
write to the same final key.

Final object keys are derived by the server from stored chunk metadata and the optional safe prefix. Do not create proxy routes, dashboards, logs, or support workflows that expose raw object keys, bucket URLs, request bodies, uploaded bytes, plaintext, raw keys, raw viewer tokens, or private deployment details.

## Optional PostgreSQL Metadata

SQLite metadata remains the default. To use PostgreSQL for metadata in a new
deployment, explicitly set `SAFE_METADATA_BACKEND=postgresql` and provide a
PostgreSQL DSN:

```bash
SAFE_METADATA_BACKEND=postgresql \
SAFE_POSTGRES_DSN='postgres://proofline:example-password@db.example.invalid:5432/proofline?sslmode=require' \
SAFE_BLOB_BACKEND=local \
SAFE_COORDINATION_BACKEND=none \
go run ./cmd/api
```

Treat `SAFE_POSTGRES_DSN`, credentials, database hostnames, and private network
details as secret-bearing deployment data. Do not place them in public issues,
logs, dashboards, screenshots, or support tickets. PostgreSQL stores metadata
only; encrypted chunk bytes still live in the configured blob backend.

Initial PostgreSQL support is for new metadata deployments. The server does not
automatically migrate existing SQLite metadata into PostgreSQL at startup. A
SQLite-to-PostgreSQL migration should be a separate quiesced operation with
metadata and encrypted blobs backed up and verified together.

PostgreSQL does not add public `/v1` exposure, public account workflows, cloud
deployment automation, backend decryption, key escrow, or production readiness.
It can store the implemented complete-upload idempotency state, but resumable
uploads, upload leases, and broader production-cluster readiness remain
separate work. Keep private `/v1` listeners behind localhost, LAN, WireGuard,
firewall rules, or a strict private proxy.

## Optional Valkey / Redis-Compatible Coordination

No coordination backend is used by default. To connect to Valkey or another
Redis-compatible service for short-lived coordination, explicitly set the
coordination backend and connection settings:

```bash
SAFE_COORDINATION_BACKEND=valkey \
SAFE_VALKEY_ADDR=valkey.example.invalid:6379 \
SAFE_VALKEY_USERNAME=proofline \
SAFE_VALKEY_PASSWORD=example-password \
SAFE_VALKEY_TLS=true \
go run ./cmd/api
```

The server checks the configured service during startup. If Valkey is
configured but unavailable, startup fails closed instead of silently running
with a misleading cluster configuration.

Valkey coordination is not durable evidence storage and is not a backup source
of truth. Incident metadata, viewer-token metadata, committed encrypted chunks,
retention decisions, and deletion decisions remain in the metadata and blob
backends. Current upload routes do not use coordination for upload leases,
idempotency result caching, resumable uploads, or application-level rate
limiting. Complete-upload idempotency keys are durable metadata records, not
Valkey records.

Treat Valkey passwords, private hostnames, network topology, and future
coordination keys as private deployment details. Do not expose them in public
issues, logs, dashboards, screenshots, support tickets, or metrics labels.
Valkey does not add public `/v1` exposure, public account workflows, cloud
deployment automation, backend decryption, key escrow, or production readiness.

## Private API Through WireGuard Or A Private Network

For a private API reachable from a WireGuard peer or private LAN, publish or bind `/v1` only on that private interface. This example uses `10.66.0.1` as a placeholder WireGuard interface address:

```bash
docker run --rm \
  -e SAFE_AUTH_BOOTSTRAP_SECRET='replace-with-local-bootstrap-secret' \
  -p 10.66.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v proofline-server-data:/data \
  proofline-server
```

Only devices that can reach `10.66.0.1:8080` through the private boundary should be able to call `/v1`. Keep host firewalls aligned with that assumption. Do not publish `8080` on `0.0.0.0` or a public interface.

The same shape can be run without Docker by binding the private API to both loopback and a private interface while keeping the incident viewer local to a same-host proxy:

```bash
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080 \
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081 \
go run ./cmd/api
```

This keeps authenticated `/v1` routes on a private network boundary. Local account sessions reduce accidental unauthenticated access, but they do not make `/v1` suitable for public exposure.

## Timeout Tuning

The private API defaults keep read and write timeouts disabled so large or slow uploads and private downloads are not interrupted. The public incident viewer has finite read/write timeouts by default, including a generous write timeout for encrypted ZIP downloads.

Reverse proxies should still set their own connection, request, and upstream timeouts. If completed evidence bundles are large or clients are slow, tune `SAFE_PUBLIC_WRITE_TIMEOUT` together with the reverse proxy timeout so the proxy does not cut off an encrypted ZIP download that the Go server is still willing to stream.

## Public Incident Viewer Exposure

If exposing any part of the current system publicly, expose only the incident
viewer listener. Future non-admin product routes may become a public
authenticated API only after satisfying the role, grant, audit, logging, and
migration expectations in [v1-access-control.md](v1-access-control.md). Future
admin/operator routes should remain on a separately bound private admin API
listener and still authenticate operators.

The checklist below is a deployment review aid. Completing it does not make
Proofline production-ready public infrastructure, and it does not make `/v1`
safe to expose publicly.

Before exposing the public incident viewer:

- [ ] The public route group forwards only to the public incident viewer
      listener, for example the listener configured by `SAFE_PUBLIC_BIND_ADDRS`.
- [ ] No public reverse-proxy route, service, wildcard rule, or fallback reaches
      the private `/v1` listener or a private API bind address.
- [ ] TLS is terminated at the deployment edge for the public hostname.
- [ ] HSTS is enabled at the HTTPS edge only after TLS is working reliably for
      the public hostname.
- [ ] Edge rate limiting covers viewer page lookup, viewer JSON polling, ZIP
      download starts, and public static assets with route-appropriate limits.
- [ ] Reverse-proxy logs, metrics, dashboards, and rate-limit keys avoid raw
      `/i/{token}` paths, legacy `/e/{token}` paths, query strings attached to
      viewer URLs, request bodies, uploaded bytes, Authorization headers,
      raw idempotency keys, plaintext, raw keys, and future token-like values.
- [ ] Viewer-token sharing, default expiry, explicit no-expiry tokens, and
      revocation workflows have been reviewed for this deployment.
- [ ] Retention, backup, restore, and deletion expectations are documented for
      this deployment and reviewed against
      [retention-backup-deletion.md](retention-backup-deletion.md).
- [ ] `SAFE_CLOSED_INCIDENT_RETENTION` is unset or set to a reviewed duration;
      backup expiry and restore reconciliation are documented before enabling
      automatic closed-incident deletion.
- [ ] Cluster backup, restore, and failure handling has been reviewed against
      [cluster-backup-restore-runbook.md](cluster-backup-restore-runbook.md)
      when optional PostgreSQL, S3-compatible storage, or Valkey/Redis
      coordination is configured.
- [ ] Restore testing confirms SQLite or PostgreSQL metadata and encrypted
      local blobs or S3 objects can be restored together without exposing `/v1`
      publicly.
- [ ] Monitoring and timeout settings cover public viewer errors, storage or
      database failures, and long encrypted ZIP downloads without logging raw
      tokens, request bodies, uploaded bytes, raw idempotency keys, plaintext,
      raw keys, or private deployment details.

The Go app still has no built-in app-level rate limiter. Apply rate limits at the deployment edge for now, and tune them for the expected recording, polling, and download patterns.

Future server-assisted break-glass, dead-man-switch key access, public account
workflows, or trusted-contact workflows would add stronger operator and
deployment trust requirements. They should remain disabled unless explicitly
designed and configured; see [v1-access-control.md](v1-access-control.md),
[break-glass-key-access.md](break-glass-key-access.md),
[key-custody.md](key-custody.md), and [incident-modes.md](incident-modes.md).

Optional PostgreSQL metadata deployment remains experimental. Schema parity,
migration tracking, transaction boundaries, configuration shape, integration
test setup, and restore expectations are documented in
[PostgreSQL metadata migration path](postgresql-metadata-migration.md).
PostgreSQL and Valkey support must not be treated as production-cluster
readiness until operation-level coordination behavior, backup/restore drills,
access-control, and operational hardening are also addressed.

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP. Enable HSTS at the HTTPS reverse proxy only after TLS is working for the production hostname.

After deploying the public incident viewer over HTTPS, test the exposed origin with the MDN HTTP Observatory:

```text
https://developer.mozilla.org/en-US/observatory
```

### HTTPS Incident Viewer With Traefik

The reverse proxy should route only the public incident viewer listener. The private `/v1` listener should stay on localhost, WireGuard, LAN, or another private boundary.

One same-host shape is:

```bash
docker run --rm \
  -e SAFE_AUTH_BOOTSTRAP_SECRET='replace-with-local-bootstrap-secret' \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v proofline-server-data:/data \
  proofline-server
```

Then configure Traefik to forward the public HTTPS hostname to `http://127.0.0.1:8081` only. This example is documentation, not a maintained deployment file; review it against the Traefik version you run before use:

```yaml
# traefik.yml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  file:
    filename: "/etc/traefik/dynamic/proofline.yml"

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
# /etc/traefik/dynamic/proofline.yml
http:
  routers:
    proofline-viewer:
      rule: "Host(`proofline.example.invalid`)"
      entryPoints:
        - websecure
      service: proofline-public
      middlewares:
        - proofline-hsts
      tls:
        certResolver: letsencrypt

  services:
    proofline-public:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8081"

  middlewares:
    proofline-hsts:
      headers:
        stsSeconds: 31536000
        stsIncludeSubdomains: false
        stsPreload: false
```

There should be no Traefik router, service, or rule for `127.0.0.1:8080` or `/v1`. If Traefik runs in a different container or on another host, point it at a private address that only Traefik can reach, and keep that address off the public internet.

Replace `admin@example.invalid` and `proofline.example.invalid` with deployment-specific values before use.

### Route-Group Rate Limiting

Use different rate limits for different route groups. A single global limiter is easy to configure, but it can either be too loose for token guessing or too strict for legitimate bundle downloads and chunk uploads.

Suggested route groups:

| Route group | Paths | Guidance |
|---|---|---|
| Viewer page lookup | `GET /i/{token}` | Keep relatively strict because each request performs a bearer-token lookup. |
| Viewer JSON polling | `GET /i/{token}/data` | Allow normal viewer polling, but keep it lower than static assets. |
| Viewer ZIP downloads | `GET /i/{token}/streams/{stream_id}/download`, `GET /i/{token}/incident/download` | Limit download starts without cutting off long encrypted ZIP responses; coordinate with proxy and app timeouts. |
| Public static assets | `GET /static/...` | Static assets are token-neutral and can usually tolerate a looser limit. |
| Private chunk uploads | `POST /v1/incidents/{incident_id}/chunks` | If routed through a private proxy, tune for expected chunk cadence and upload retries. |
| Private incident, stream, check-in, token, and admin-style actions | Other `/v1/...` routes | Keep behind a private boundary and use limits as an abuse backstop, not as the only security control. |

Rate limiting does not make `/v1` safe to expose publicly. Keep the private API on localhost, LAN, WireGuard, firewall rules, or a private reverse-proxy entry point even when limits are configured.

Exact limits are deployment-specific. Start with conservative values, watch legitimate simulator/client behavior, then adjust. Avoid sending raw `/i/{token}` paths or pre-rename compatibility `/e/{token}` paths to metrics, dashboards, or logs while measuring limiter behavior.

### Traefik Rate-Limiting Example

Traefik's `rateLimit` middleware uses `average`, `period`, and `burst` to define a token-bucket limit. Review the options for the Traefik version you run, especially the source criterion used to group requests behind proxies.

This example replaces the single broad public viewer router from the basic example above with grouped routers for the same public service. Do not append these routers alongside the broad router unless you have deliberately reviewed the resulting priorities and middleware order. The numbers are illustrative placeholders, not production defaults:

```yaml
# /etc/traefik/dynamic/proofline.yml
http:
  routers:
    proofline-downloads:
      rule: "Host(`proofline.example.invalid`) && Method(`GET`) && PathRegexp(`^/i/[^/]+/(streams/[^/]+/download|incident/download)$`)"
      entryPoints:
        - websecure
      service: proofline-public
      middlewares:
        - proofline-rate-downloads
        - proofline-hsts
      priority: 120
      tls:
        certResolver: letsencrypt

    proofline-data:
      rule: "Host(`proofline.example.invalid`) && Method(`GET`) && PathRegexp(`^/i/[^/]+/data$`)"
      entryPoints:
        - websecure
      service: proofline-public
      middlewares:
        - proofline-rate-data
        - proofline-hsts
      priority: 110
      tls:
        certResolver: letsencrypt

    proofline-page:
      rule: "Host(`proofline.example.invalid`) && Method(`GET`) && PathRegexp(`^/i/[^/]+$`)"
      entryPoints:
        - websecure
      service: proofline-public
      middlewares:
        - proofline-rate-page
        - proofline-hsts
      priority: 100
      tls:
        certResolver: letsencrypt

    proofline-static:
      rule: "Host(`proofline.example.invalid`) && Method(`GET`) && PathPrefix(`/static/`)"
      entryPoints:
        - websecure
      service: proofline-public
      middlewares:
        - proofline-rate-static
        - proofline-hsts
      priority: 90
      tls:
        certResolver: letsencrypt

  services:
    proofline-public:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8081"

  middlewares:
    proofline-rate-page:
      rateLimit:
        average: 20
        period: 1m
        burst: 10

    proofline-rate-data:
      rateLimit:
        average: 60
        period: 1m
        burst: 20

    proofline-rate-downloads:
      rateLimit:
        average: 6
        period: 1m
        burst: 3

    proofline-rate-static:
      rateLimit:
        average: 120
        period: 1m
        burst: 60

    proofline-hsts:
      headers:
        stsSeconds: 31536000
        stsIncludeSubdomains: false
        stsPreload: false
```

If the private API is also routed through Traefik, it should use a private-only entry point, private address, or private network. Do not attach private `/v1` routers to public entry points. A private-only file-provider shape can split uploads from other private actions.

Define the private entry point in Traefik's static configuration first. This example uses `wireguard` as a placeholder entry point name and `10.66.0.1:80` as a placeholder private HTTP interface address:

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
    proofline-private-uploads:
      rule: "Host(`proofline-private.example.invalid`) && Method(`POST`) && PathRegexp(`^/v1/incidents/[^/]+/chunks$`)"
      entryPoints:
        - wireguard
      service: proofline-private
      middlewares:
        - proofline-rate-private-uploads
      priority: 110

    proofline-private-api:
      rule: "Host(`proofline-private.example.invalid`) && PathPrefix(`/v1/`)"
      entryPoints:
        - wireguard
      service: proofline-private
      middlewares:
        - proofline-rate-private-api
      priority: 100

  services:
    proofline-private:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8080"

  middlewares:
    proofline-rate-private-uploads:
      rateLimit:
        average: 120
        period: 1m
        burst: 60

    proofline-rate-private-api:
      rateLimit:
        average: 30
        period: 1m
        burst: 15
```

When Traefik sits behind another proxy or load balancer, review forwarded-header trust and the rate-limit source criterion. A misconfigured source can group all clients under one proxy IP, or trust client-supplied forwarding headers too loosely.

### Viewer Token Paths In Proxy Logs

Viewer URLs are bearer-token URLs. The Go server logs redacted route patterns such as `/i/{token}`, but an edge proxy can still log the raw request path before the request reaches the Go server. During upgrades from pre-rename releases, `/e/{token}` compatibility alias requests are also token-bearing paths and should be redacted.

For Traefik, use an access-log format that supports field controls, then review the fields for the version you deploy and drop or sanitize request path fields. If path redaction is unavailable in your logging format, disable access logs for this router or pass logs through a sanitizer before storage. Redacting headers is not enough because the token is in the URL path.

Avoid logging:

- raw `/i/{token}` paths
- pre-rename compatibility `/e/{token}` paths
- query strings attached to viewer URLs
- request bodies
- uploaded bytes
- `Authorization` headers
- raw `Idempotency-Key` values
- rate-limit keys or metric labels containing raw viewer tokens
- plaintext, raw keys, or future token-like values

### Proxy And App Timeout Coordination

Completed stream and incident downloads can be large encrypted ZIP responses. Keep Traefik entry point, upstream, and client-response timeouts at least as permissive as the expected download window, and review them together with `SAFE_PUBLIC_WRITE_TIMEOUT`.

For example, if the public viewer runs with:

```bash
SAFE_PUBLIC_WRITE_TIMEOUT=10m
```

then the Traefik route serving the incident viewer should also allow a slow client to receive the response for roughly that long. If the proxy timeout is shorter than the Go server timeout, downloads may fail even though the backend is configured to keep streaming.

## GitHub Actions And GHCR

The CI workflow:

- runs Go tests from the repository root
- builds a Linux amd64 binary artifact
- generates release binary attestations from a tag-only attestation job
- creates a minimal GitHub Release when needed and uploads the Linux amd64 binary as a Release asset for `v*` tags
- builds the Docker image from `Dockerfile` with the repository root as build context
- publishes `ghcr.io/open-proofline/server` on pushes to `main` and `v*` tags
- attaches attestations to published GHCR images
- keeps workflow-level token permissions read-only and grants write permissions only to the tag-only binary attestation, release binary upload, and trusted Docker publish jobs

The previous `ghcr.io/thesilkky/safety-recorder` package name is historical. New release and deployment references should use `ghcr.io/open-proofline/server`; deployments pinned to old images should migrate deliberately.
