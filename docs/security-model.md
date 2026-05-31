# Security Model

This document summarizes the current Proofline backend security assumptions and controls. For a threat-oriented view, see [threat-model.md](threat-model.md). For planned incident-mode behavior, see [incident-modes.md](incident-modes.md). For `/v1` role and grant boundaries, see [v1-access-control.md](v1-access-control.md). For future production key custody and emergency access design, see [key-custody.md](key-custody.md), the simulator-only wrapped-key metadata prototype in [contact-wrapped-key-metadata-simulator.md](contact-wrapped-key-metadata-simulator.md), [browser-decryption.md](browser-decryption.md), [live-partial-stream-access-boundary.md](live-partial-stream-access-boundary.md), and [break-glass-key-access.md](break-glass-key-access.md). For vulnerability reporting, see [../SECURITY.md](../SECURITY.md).

## Maturity

Proofline is experimental and not production-ready public infrastructure. The private `/v1` API has local username/password accounts and opaque server-side sessions. It still has no OAuth, no JWT protection, no public product API hardening, and no public account portal.

The current backend stores incidents owned by local accounts. Incidents are
generic by default and may include optional incident-mode, capture-profile,
escalation-policy, and sharing-state metadata. Those fields are not behavior
flags and do not grant access, send notifications, change retention, change key
custody, expose trusted-contact workflows, or change public viewer and bundle
behavior. The backend does not yet implement trusted-contact accounts,
dead-man switch notifications, mode-driven sharing, or public account-based
product access.

The `/v1` access-control direction is documented in
[v1-access-control.md](v1-access-control.md). The current implementation covers
local account sessions, owner-scoped incident access, admin account routes, and
private route authentication. It does not make `/v1` safe to expose publicly as
a product API. Future topology still separates a public authenticated product
API from a separately bound private admin API listener that requires
authentication and authorization.

## Listener Boundary

The API binary starts separate listener groups:

| Listener group | Routes | Intended exposure |
|---|---|---|
| Private API | Authenticated `/v1/...`, unauthenticated private `/v1/health/live` and `/v1/health/ready`, plus private `/admin` web routes | Localhost, LAN, WireGuard, firewall, or strict reverse proxy only. |
| Public incident viewer | `/i/{token}` and related read-only routes, plus pre-rename `/e/{token}` compatibility aliases | HTTPS/reverse proxy when exposed. |

Private write/admin routes must not be mounted on public incident viewer listeners. Incident viewer routes are read-only.

## Account And Token Handling

Local accounts are stored in the configured metadata backend. Passwords are
stored as bcrypt password hashes, not plaintext. Session tokens are opaque
server-side bearer credentials. The raw session token is returned only by
login; the metadata backend stores only a SHA-256 hash. Sessions expire after
`SAFE_SESSION_TTL`, defaulting to 12 hours, and can be revoked by logout,
account password change, admin password reset, or admin session revocation.

The server fails closed on startup unless an admin account exists or
`SAFE_AUTH_BOOTSTRAP_SECRET` is set for the one-time bootstrap route. The
bootstrap route is disabled once an admin account exists. Treat the bootstrap
secret, account passwords, session tokens, and Authorization headers as
secrets.

`GET /v1/health/live` and `GET /v1/health/ready` are unauthenticated because
they are intended for local Docker checks, private reverse-proxy upstream
checks, and operator troubleshooting without storing a session token. They are
mounted only on the private API mux. Their responses are coarse and must not
expose DSNs, credentials, bucket names, object keys, stored paths, local
filesystem paths, private hostnames, tokens, request bodies, uploaded bytes,
plaintext, raw keys, private deployment details, or underlying error strings.

The private `/admin` page, login form, bootstrap form, and account password
workflows are mounted only on the private API mux, not on the public incident
viewer mux. The browser flow uses the same server-side session store as `/v1`
authentication and stores the raw session token in an HttpOnly SameSite=Strict
cookie scoped to `/admin`. Authenticated `/admin` pages require the `admin`
role. Authenticated state-changing admin web forms use a session-bound CSRF
token. The token-neutral CSS under `/admin/static/...` is unauthenticated
because it is public source code and contains no incident data, secrets, tokens,
keys, or deployment details. This does not add a public admin dashboard or
public product API exposure model.

Incident viewer tokens are scoped to one incident. The raw token is returned only at creation time; the configured metadata backend stores only a SHA-256 hash. Tokens created without an explicit `expires_at` default to a 24-hour lifetime unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is configured differently. Expired, revoked, and invalid tokens return the same public error.

Viewer URLs contain bearer tokens and should be treated as secrets. Reverse proxies and operational logs should avoid recording raw `/i/{token}` paths. During upgrades from pre-rename releases, `/e/{token}` compatibility links may also reach the edge proxy and should be redacted.

## Upload And Storage Controls

- Uploads are streamed to a temp directory while SHA-256 is computed.
- Upload file bytes are limited by `SAFE_MAX_UPLOAD_BYTES`.
- Final chunk storage happens only after hash verification.
- Stored chunks are immutable and never overwritten.
- Local storage commits use no-overwrite hard links. Optional S3-compatible storage commits final objects with conditional no-overwrite writes.
- Streamed uploads require positive chunk indexes, while legacy unstreamed uploads may still use index `0`.
- `original_filename` is optional client-supplied display metadata. The server
  strips it to a basename and may return it in private chunk metadata,
  token-scoped public incident viewer summaries, and bundle manifests. Future
  clients should omit it by default or use a generic basename unless preserving
  filename context is an explicit user or protocol decision.
- The simulator can wrap chunks in the documented v1 AES-256-GCM client-side encryption envelope before upload. Desktop-recorder mode can stage encrypted chunks locally and retry complete encrypted uploads without adding server-visible partial upload state.
- The backend validates and stores ciphertext bytes only; it does not store encryption keys or decrypt chunk contents.
- SQLite and optional PostgreSQL metadata enforce media type, chunk index, byte size, SHA-256 shape, foreign keys, and unique chunk identity.
- Complete chunk uploads can include an `Idempotency-Key` header. The backend
  stores only a SHA-256 hash of the key in durable metadata, binds it to the
  normalized chunk identity and immutable request fingerprint, and can return
  `200 OK` with `Idempotency-Replayed: true` for equivalent retries without
  overwriting chunks or evidence metadata.
- The private duplicate chunk reconciliation route compares a requested
  normalized chunk identity and expected immutable fingerprint against accepted
  chunk metadata without re-uploading ciphertext, reading stored bytes, or
  returning stored paths, object keys, uploaded bytes, plaintext, raw keys, raw
  tokens, or conflicting stored values.
- Chunk metadata inserts recheck incident and stream state in the repository so uploads racing with close or completion are rejected.
- Media stream completion verifies contiguous chunks and readable stored files, then rechecks chunk rows transactionally before committing completion.
- Local account authorization binds private incident access to the
  authenticated account, the incident owner, and the role. Current private
  incident routes also pass route-level action and data-class labels, but all
  current incident actions share the same owner-or-admin policy. Regular users
  can access their own incidents. Admins can access incidents across accounts.
  Legacy unowned incidents are admin-only until a future migration or
  reassignment workflow exists.

Optional S3-compatible storage preserves ciphertext-only behavior for committed
encrypted chunks. It uses server-controlled object keys, does not expose object
store URLs in evidence bundles, and does not add backend decryption or key
custody.

Optional PostgreSQL metadata support preserves these controls with equivalent
or stronger constraints, duplicate guards, token-hash storage, and row-locking
transaction boundaries. The implementation and remaining migration limits are
documented in [postgresql-metadata-migration.md](postgresql-metadata-migration.md).

Optional Valkey/Redis-compatible coordination can be configured for
short-lived coordination checks. It is not durable evidence storage, does not
hold incident metadata, viewer-token metadata, committed encrypted bytes,
retention decisions, plaintext, or keys, and does not change the private
`/v1` boundary.

Private readiness checks can report only coarse metadata, blob, and coordination
backend status. They are operator checks, not public diagnostics, metrics,
support dashboards, or evidence-inspection routes.

Cluster-safe upload operation semantics are documented in
[cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md). The
implemented path is limited to complete-upload idempotency keys; resumable
uploads and upload leases are still planning-only. The current API still
accepts complete encrypted chunks and retries should resend the complete chunk.
See
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md).

## Bundle Controls

Completed stream and incident bundles are generated on demand as ZIP responses. ZIP entry names are controlled by the server. Manifests are generated from database metadata and do not expose server filesystem paths.

Bundle manifests may include `original_filename` basenames because those values
are current chunk display metadata. They are user/client metadata, not server
stored paths, staging paths, object-storage keys, ZIP entry names, or download
paths.

Incident bundle generation fails closed if any completed stream cannot be reconstructed. It does not silently omit inconsistent completed streams from the ZIP or manifest.

Bundles contain encrypted chunk bytes and JSON manifests only. They are not decrypted, playable, or merged media exports.

Bundle manifests may include a non-secret client-side encryption hint. They do not include keys.

## Incident Modes And Escalation Boundary

Incident-mode metadata is currently limited to optional private incident fields.
Emergency incidents, interaction records, safety checks, and evidence notes must
not weaken the current storage, encryption, listener, or logging boundaries. The
schema keeps incident mode, capture profile, escalation policy, and sharing state
separate; see [incident-modes.md](incident-modes.md).

Future escalation policies should keep capture separate from notification and emergency response:

- non-emergency interaction records should not automatically alert trusted contacts by default
- safety checks should alert trusted contacts only after an explicit missed-check-in policy is implemented
- Proofline must not claim to contact emergency services unless a future jurisdiction-specific integration is explicitly designed, implemented, and documented
- sharing, export, publication, and legal submission should remain deliberate user-controlled actions

The current backend does not decide whether an incident is an emergency, does not notify trusted contacts, and does not contact emergency services.

Future incident-mode access must follow the role and grant boundaries in
[v1-access-control.md](v1-access-control.md). Incident labels, capture profiles,
or sharing-state summaries must not silently grant trusted-contact, public-link,
admin/operator, escrow, key, or plaintext access.

## Logging And Headers

Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, raw session tokens, raw viewer tokens, raw incident tokens, raw idempotency keys, plaintext, or raw keys.

The Go app sets these headers on public incident viewer pages, JSON responses, static assets, ZIP downloads, and private admin web responses:

- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `X-Frame-Options: DENY`

Token-protected incident pages, JSON responses, errors, ZIP downloads, and admin web pages also use `Cache-Control: no-store`, including automatic method-mismatch errors on token-bearing paths. Private API responses use `X-Content-Type-Options: nosniff` and `Cache-Control: no-store`; JSON responses also use `Content-Type: application/json`.

HSTS is not enabled by default in the Go app because local development uses plain HTTP and HSTS should only be sent over HTTPS. Set `Strict-Transport-Security` at the production HTTPS reverse proxy after TLS is established for the public hostname. After deployment, test the public incident viewer with the MDN HTTP Observatory.

HTTP server timeouts are configurable separately for private and public listener groups. Private read/write timeouts default to disabled for large uploads/downloads; public viewer timeouts are finite by default and should be coordinated with reverse-proxy timeouts.

The Go app does not include an app-level rate limiter. Deployment-edge rate limiting guidance is documented in [deployment.md](deployment.md), but those proxy controls do not replace private `/v1` access boundaries, local account authentication, or future public product API abuse controls.

## Retention, Backups, And Deletion

Retention, backup, restore, secure deletion limits, and disk encryption posture
are documented in
[retention-backup-deletion.md](retention-backup-deletion.md). The future
incident deletion and retention enforcement design is documented in
[incident-deletion-retention-enforcement.md](incident-deletion-retention-enforcement.md).
The current backend preserves accepted evidence by default and does not
automatically expire incidents or expose incident deletion APIs.

SQLite WAL file expectations, same-host storage constraints, checkpoint
pressure symptoms, and local file-size checks are documented in
[deployment.md](deployment.md#sqlite-wal-operations).

Cluster backup, restore, and failure-mode guidance for optional PostgreSQL
metadata, S3-compatible encrypted blobs, and Valkey/Redis-compatible
coordination is documented in
[Cluster backup, restore, and failure runbook](cluster-backup-restore-runbook.md).

Normal file or object removal is not treated as guaranteed secure erasure. Deployments that store real incident evidence should use encrypted disks, encrypted volumes, encrypted object buckets, logs, and backups, then rely on explicit backup expiry and encryption-key retirement for stronger deletion outcomes.

## Known Security Gaps

- No implemented public product API exposure model for `/v1`; local account
  sessions are a private API control, not a complete public security model
- No built-in TLS
- No built-in app-level rate limiting or abuse throttling
- PostgreSQL metadata and Valkey/Redis-compatible coordination are optional
  and experimental; they do not by themselves make the upload path cluster-safe
  or make `/v1` safe for public exposure
- Cluster backup, restore, and failure runbooks are operational guidance only;
  they do not add access control, retention enforcement, observability, abuse
  controls, or production readiness
- No resumable, partial, or leased cluster-safe upload protocol beyond the
  implemented complete-upload `Idempotency-Key` path documented in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md)
- No implemented resumable upload or upload lease protocol; the future design
  is planned in
  [resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md)
- No implemented mode-driven access, escalation, retention, sharing, key-custody,
  trusted-contact account, dead-man switch notification, or public account portal
  behavior
- No implemented production client key storage, key sharing, browser decryption, server-assisted break-glass key access, or emergency-contact key access model; the future designs are documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- No implemented live or partial stream access beyond current read-only stream
  metadata summaries and completed encrypted bundle downloads; the future
  boundary is documented in
  [live-partial-stream-access-boundary.md](live-partial-stream-access-boundary.md)
- No automated retention/deletion enforcement or built-in disk encryption; the
  operational policy is documented in
  [retention-backup-deletion.md](retention-backup-deletion.md), with future
  enforcement design in
  [incident-deletion-retention-enforcement.md](incident-deletion-retention-enforcement.md)
- No malware/content scanning for uploaded encrypted blobs
- No implemented account self-service recovery, email verification, second
  factor authentication, delegated identity provider, or public account portal
