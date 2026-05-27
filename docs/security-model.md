# Security Model

This document summarizes the current Proofline backend security assumptions and controls. For a threat-oriented view, see [threat-model.md](threat-model.md). For planned incident-mode behavior, see [incident-modes.md](incident-modes.md). For future production key custody and emergency access design, see [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md). For vulnerability reporting, see [../SECURITY.md](../SECURITY.md).

## Maturity

Proofline is experimental and not production-ready public infrastructure. The private `/v1` API has no public user authentication, no user accounts, no OAuth, and no JWT protection.

The current backend stores generic incidents only. It does not yet implement first-class incident types, escalation policies, trusted-contact accounts, dead-man switch notifications, or account-based access to personal incident data.

## Listener Boundary

The API binary starts separate listener groups:

| Listener group | Routes | Intended exposure |
|---|---|---|
| Private API | `/v1/...` | Localhost, LAN, WireGuard, firewall, or strict reverse proxy only. |
| Public incident viewer | `/i/{token}` and related read-only routes | HTTPS/reverse proxy when exposed. |

Private write/admin routes must not be mounted on public incident viewer listeners. Incident viewer routes are read-only.

## Token Handling

Incident viewer tokens are scoped to one incident. The raw token is returned only at creation time; SQLite stores only a SHA-256 hash. Tokens created without an explicit `expires_at` default to a 24-hour lifetime unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is configured differently. Expired, revoked, and invalid tokens return the same public error.

Viewer URLs contain bearer tokens and should be treated as secrets. Reverse proxies and operational logs should avoid recording raw `/i/{token}` paths.

## Upload And Storage Controls

- Uploads are streamed to a temp directory while SHA-256 is computed.
- Upload file bytes are limited by `SAFE_MAX_UPLOAD_BYTES`.
- Final chunk storage happens only after hash verification.
- Stored chunks are immutable and never overwritten.
- Streamed uploads require positive chunk indexes, while legacy unstreamed uploads may still use index `0`.
- The simulator can wrap chunks in the documented v1 AES-256-GCM client-side encryption envelope before upload.
- The backend validates and stores ciphertext bytes only; it does not store encryption keys or decrypt chunk contents.
- SQLite enforces media type, chunk index, byte size, SHA-256 shape, foreign keys, and unique chunk identity.
- Chunk metadata inserts recheck incident and stream state in the repository so uploads racing with close or completion are rejected.
- Media stream completion verifies contiguous chunks and readable stored files, then rechecks chunk rows transactionally before committing completion.

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

## Logging And Headers

Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, raw viewer tokens, raw incident tokens, plaintext, or raw keys.

The Go app sets these headers on public incident viewer pages, JSON responses, static assets, and ZIP downloads:

- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `X-Frame-Options: DENY`

Token-protected incident pages, JSON responses, errors, and ZIP downloads also use `Cache-Control: no-store`. Private API JSON responses use `Content-Type: application/json`, `X-Content-Type-Options: nosniff`, and `Cache-Control: no-store`.

HSTS is not enabled by default in the Go app because local development uses plain HTTP and HSTS should only be sent over HTTPS. Set `Strict-Transport-Security` at the production HTTPS reverse proxy after TLS is established for the public hostname. After deployment, test the public incident viewer with the MDN HTTP Observatory.

HTTP server timeouts are configurable separately for private and public listener groups. Private read/write timeouts default to disabled for large uploads/downloads; public viewer timeouts are finite by default and should be coordinated with reverse-proxy timeouts.

The Go app does not include an app-level rate limiter. Deployment-edge rate limiting guidance is documented in [deployment.md](deployment.md), but those proxy controls do not replace private `/v1` access boundaries or future application-level authorization.

## Retention, Backups, And Deletion

Retention, backup, restore, secure deletion limits, and disk encryption posture are documented in [retention-backup-deletion.md](retention-backup-deletion.md). The current backend preserves accepted evidence by default and does not automatically expire incidents or expose incident deletion APIs.

Normal file removal is not treated as guaranteed secure erasure. Deployments that store real incident evidence should use encrypted disks or volumes for SQLite, encrypted blobs, logs, and backups, then rely on explicit backup expiry and encryption-key retirement for stronger deletion outcomes.

## Known Security Gaps

- No public authentication or authorization model for `/v1`
- No built-in TLS
- No built-in app-level rate limiting or abuse throttling
- No implemented first-class incident types, escalation policies, trusted-contact accounts, dead-man switch notifications, or account-based access model
- No implemented production client key storage, key sharing, browser decryption, server-assisted break-glass key access, or emergency-contact key access model; the future designs are documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- No automated retention/deletion enforcement or built-in disk encryption; the operational policy is documented in [retention-backup-deletion.md](retention-backup-deletion.md)
- No malware/content scanning for uploaded encrypted blobs
- No multi-user authorization model
