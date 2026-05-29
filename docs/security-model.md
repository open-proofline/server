# Security Model

This document summarizes the current Proofline backend security assumptions and controls. For a threat-oriented view, see [threat-model.md](threat-model.md). For planned incident-mode behavior, see [incident-modes.md](incident-modes.md). For future `/v1` role and grant boundaries, see [v1-access-control.md](v1-access-control.md). For future production key custody and emergency access design, see [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md). For vulnerability reporting, see [../SECURITY.md](../SECURITY.md).

## Maturity

Proofline is experimental and not production-ready public infrastructure. The private `/v1` API has no public user authentication, no user accounts, no OAuth, and no JWT protection.

The current backend stores generic incidents only. It does not yet implement first-class incident types, escalation policies, trusted-contact accounts, dead-man switch notifications, or account-based access to personal incident data.

The future `/v1` access-control direction is documented in
[v1-access-control.md](v1-access-control.md). That document is planning-only
and does not make the current unauthenticated `/v1` API safe to expose
publicly. Its future topology separates a public authenticated product API from
a separately bound private admin API listener that still requires
authentication and authorization.

## Listener Boundary

The API binary starts separate listener groups:

| Listener group | Routes | Intended exposure |
|---|---|---|
| Private API | `/v1/...` | Localhost, LAN, WireGuard, firewall, or strict reverse proxy only. |
| Public incident viewer | `/i/{token}` and related read-only routes, plus pre-rename `/e/{token}` compatibility aliases | HTTPS/reverse proxy when exposed. |

Private write/admin routes must not be mounted on public incident viewer listeners. Incident viewer routes are read-only.

## Token Handling

Incident viewer tokens are scoped to one incident. The raw token is returned only at creation time; the configured metadata backend stores only a SHA-256 hash. Tokens created without an explicit `expires_at` default to a 24-hour lifetime unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is configured differently. Expired, revoked, and invalid tokens return the same public error.

Viewer URLs contain bearer tokens and should be treated as secrets. Reverse proxies and operational logs should avoid recording raw `/i/{token}` paths. During upgrades from pre-rename releases, `/e/{token}` compatibility links may also reach the edge proxy and should be redacted.

## Upload And Storage Controls

- Uploads are streamed to a temp directory while SHA-256 is computed.
- Upload file bytes are limited by `SAFE_MAX_UPLOAD_BYTES`.
- Final chunk storage happens only after hash verification.
- Stored chunks are immutable and never overwritten.
- Local storage commits use no-overwrite hard links. Optional S3-compatible storage commits final objects with conditional no-overwrite writes.
- Streamed uploads require positive chunk indexes, while legacy unstreamed uploads may still use index `0`.
- The simulator can wrap chunks in the documented v1 AES-256-GCM client-side encryption envelope before upload.
- The backend validates and stores ciphertext bytes only; it does not store encryption keys or decrypt chunk contents.
- SQLite and optional PostgreSQL metadata enforce media type, chunk index, byte size, SHA-256 shape, foreign keys, and unique chunk identity.
- Chunk metadata inserts recheck incident and stream state in the repository so uploads racing with close or completion are rejected.
- Media stream completion verifies contiguous chunks and readable stored files, then rechecks chunk rows transactionally before committing completion.

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

Future cluster-safe upload operation semantics are planned separately in
[cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md), but no
idempotency-key or upload-operation API is implemented yet.
Resumable uploads and upload leases are also planning-only; the current API
still accepts complete encrypted chunks and retries should resend the complete
chunk. See
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md).

## Bundle Controls

Completed stream and incident bundles are generated on demand as ZIP responses. ZIP entry names are controlled by the server. Manifests are generated from database metadata and do not expose server filesystem paths.

Incident bundle generation fails closed if any completed stream cannot be reconstructed. It does not silently omit inconsistent completed streams from the ZIP or manifest.

Bundles contain encrypted chunk bytes and JSON manifests only. They are not decrypted, playable, or merged media exports.

Bundle manifests may include a non-secret client-side encryption hint. They do not include keys.

## Incident Modes And Escalation Boundary

Planned incident modes are a future client/protocol layer. Emergency incidents, interaction records, safety checks, and evidence notes must not weaken the current storage, encryption, listener, or logging boundaries.

Future escalation policies should keep capture separate from notification and emergency response:

- non-emergency interaction records should not automatically alert trusted contacts by default
- safety checks should alert trusted contacts only after an explicit missed-check-in policy is implemented
- Proofline must not claim to contact emergency services unless a future jurisdiction-specific integration is explicitly designed, implemented, and documented
- sharing, export, publication, and legal submission should remain deliberate user-controlled actions

The current backend does not decide whether an incident is an emergency, does not notify trusted contacts, and does not contact emergency services.

Future incident-mode access must follow the role and grant boundaries in
[v1-access-control.md](v1-access-control.md). Incident labels must not silently
grant trusted-contact, public-link, admin/operator, escrow, key, or plaintext
access.

## Logging And Headers

Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, raw viewer tokens, raw incident tokens, plaintext, or raw keys.

The Go app sets these headers on public incident viewer pages, JSON responses, static assets, and ZIP downloads:

- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `X-Frame-Options: DENY`

Token-protected incident pages, JSON responses, errors, and ZIP downloads also use `Cache-Control: no-store`, including automatic method-mismatch errors on token-bearing paths. Private API responses use `X-Content-Type-Options: nosniff` and `Cache-Control: no-store`; JSON responses also use `Content-Type: application/json`.

HSTS is not enabled by default in the Go app because local development uses plain HTTP and HSTS should only be sent over HTTPS. Set `Strict-Transport-Security` at the production HTTPS reverse proxy after TLS is established for the public hostname. After deployment, test the public incident viewer with the MDN HTTP Observatory.

HTTP server timeouts are configurable separately for private and public listener groups. Private read/write timeouts default to disabled for large uploads/downloads; public viewer timeouts are finite by default and should be coordinated with reverse-proxy timeouts.

The Go app does not include an app-level rate limiter. Deployment-edge rate limiting guidance is documented in [deployment.md](deployment.md), but those proxy controls do not replace private `/v1` access boundaries or future application-level authorization.

## Retention, Backups, And Deletion

Retention, backup, restore, secure deletion limits, and disk encryption posture are documented in [retention-backup-deletion.md](retention-backup-deletion.md). The current backend preserves accepted evidence by default and does not automatically expire incidents or expose incident deletion APIs.

Cluster backup, restore, and failure-mode guidance for optional PostgreSQL
metadata, S3-compatible encrypted blobs, and Valkey/Redis-compatible
coordination is documented in
[Cluster backup, restore, and failure runbook](cluster-backup-restore-runbook.md).

Normal file or object removal is not treated as guaranteed secure erasure. Deployments that store real incident evidence should use encrypted disks, encrypted volumes, encrypted object buckets, logs, and backups, then rely on explicit backup expiry and encryption-key retirement for stronger deletion outcomes.

## Known Security Gaps

- No implemented public authentication or authorization model for `/v1`; the
  future design is planning-only in [v1-access-control.md](v1-access-control.md)
- No built-in TLS
- No built-in app-level rate limiting or abuse throttling
- PostgreSQL metadata and Valkey/Redis-compatible coordination are optional
  and experimental; they do not by themselves make the upload path cluster-safe
  or make `/v1` safe for public exposure
- Cluster backup, restore, and failure runbooks are operational guidance only;
  they do not add access control, retention enforcement, observability, abuse
  controls, or production readiness
- No implemented cluster-safe upload operation or idempotency API; the future
  semantics are only planned in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md)
- No implemented resumable upload or upload lease protocol; the future design
  is planned in
  [resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md)
- No implemented first-class incident types, escalation policies, trusted-contact accounts, dead-man switch notifications, or account-based access model
- No implemented production client key storage, key sharing, browser decryption, server-assisted break-glass key access, or emergency-contact key access model; the future designs are documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- No automated retention/deletion enforcement or built-in disk encryption; the operational policy is documented in [retention-backup-deletion.md](retention-backup-deletion.md)
- No malware/content scanning for uploaded encrypted blobs
- No implemented multi-user authorization model
