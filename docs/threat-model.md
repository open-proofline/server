# Threat Model

This document describes the current Proofline backend-only security posture. It is intentionally conservative and does not claim production readiness.

Planned future incident modes include emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. Those modes are not implemented yet. Current controls apply to generic incidents, encrypted chunk uploads, checkins, viewer tokens, and encrypted evidence bundles.

## Assets

- Already-encrypted uploaded chunk files under `SAFE_DATA_DIR` for local storage, or committed encrypted objects in the configured S3-compatible bucket
- Incident, media stream, chunk, checkin, and viewer/incident-token metadata in SQLite by default or optional PostgreSQL
- Optional chunk `original_filename` display metadata. The server strips it to a
  basename, but it can still contain user-supplied contextual or personal
  information and may appear in viewer summaries and bundle manifests.
- Optional PostgreSQL metadata schema, migration, transaction, test, and
  restore expectations are documented in
  [postgresql-metadata-migration.md](postgresql-metadata-migration.md)
- Optional Valkey/Redis-compatible coordination is startup-checked when
  explicitly configured, but it is short-lived coordination state only and is
  not durable evidence storage
- Future cluster-safe upload operation semantics are planned but not
  implemented; idempotency, retry-success, conflict, and cleanup expectations
  are documented in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md)
- Future resumable upload and upload lease behavior is planned but not
  implemented; a local desktop recorder simulator client should use complete
  encrypted chunk retries as documented in
  [resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md)
- Cluster backup, restore, and failure-mode guidance for optional PostgreSQL
  metadata, S3-compatible encrypted blobs, and Valkey/Redis-compatible
  coordination is documented in
  [cluster-backup-restore-runbook.md](cluster-backup-restore-runbook.md)
- On-demand encrypted evidence ZIP bundles generated from completed streams
- Raw viewer/incident tokens returned once at creation time
- Incident viewer URLs containing bearer tokens
- Simulator-only local encryption key files when developers opt into `--key-file`
- Future mobile/web recordings, interaction-record metadata, safety-check
  state, account-owner access, trusted-contact access, production client-side
  keys, key sharing, browser decryption, and break-glass key access are out of
  scope for the current implementation. Planned incident modes are documented
  in [incident-modes.md](incident-modes.md), future role and grant boundaries
  are documented in [v1-access-control.md](v1-access-control.md), the intended
  future key custody direction is documented in [key-custody.md](key-custody.md),
  browser decryption constraints are documented in
  [browser-decryption.md](browser-decryption.md), and server-assisted access
  design is documented in [break-glass-key-access.md](break-glass-key-access.md).

## Trust Boundaries

- The private API server binds separately from the public incident viewer server. By default it listens on `127.0.0.1:8080`, and it can listen on multiple addresses through `SAFE_PRIVATE_BIND_ADDRS`.
- The public incident viewer server binds separately from the private API server. By default it listens on `127.0.0.1:8081`, and it can listen on multiple addresses through `SAFE_PUBLIC_BIND_ADDRS`.
- `/v1` routes are private/admin routes. They can create incidents, create streams, upload chunks, complete/fail streams, close incidents, create viewer tokens, revoke tokens, and read encrypted bytes. They are mounted only on the private API server.
- `/i/{token}`, `/i/{token}/data`, and viewer bundle download routes are public-shaped read-only routes gated by a bearer token. Pre-rename `/e/{token}` viewer, data, and download paths remain as compatibility aliases. These routes are mounted only on the public incident viewer server.
- Static assets under `/static/` are embedded and token-neutral.

## Current Controls

- Uploads stream to `data/tmp` while computing SHA-256 and enforcing `SAFE_MAX_UPLOAD_BYTES`.
- Upload-limit configuration rejects non-positive, sub-byte, invalid, and oversized values before request-size limits are applied.
- Uploaded bytes are committed only after hash verification.
- Final local chunk storage uses no-overwrite hard links. Optional S3-compatible storage uses conditional no-overwrite final object writes.
- The simulator encrypts fake chunk plaintext by default using the documented v1 AES-256-GCM envelope.
- Encryption keys remain client-side; they are not uploaded, stored in SQLite, or added to evidence bundles.
- SQLite and optional PostgreSQL metadata enforce media type, chunk index, byte size, SHA-256 shape, foreign keys, and unique chunk identity.
- Optional Valkey/Redis-compatible coordination fails closed at startup when
  explicitly configured but unavailable.
- Media streams must be open before new chunks can be attached. The repository rechecks incident and stream state when chunk metadata is inserted.
- Stream completion verifies contiguous chunks plus readable stored files, and the repository revalidates chunk rows before committing the stream to `complete`.
- Viewer tokens use 256 bits from `crypto/rand`; only SHA-256 token hashes are stored. Tokens created without an explicit `expires_at` default to a 24-hour lifetime unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is configured differently.
- Expired, revoked, and invalid viewer tokens return the same public error.
- Incident summaries do not expose `stored_path`. Viewer summaries and bundle
  manifests may expose user-supplied `original_filename` basenames when clients
  provided them. Viewer bundle downloads expose only encrypted chunk bytes and
  generated manifests for completed streams.
- ZIP bundle entry names are server-controlled and generated from metadata; clients do not provide stored paths for download.
- Public viewer responses use a strict same-origin `Content-Security-Policy` with `frame-ancestors 'none'`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, and a restrictive camera/microphone/geolocation `Permissions-Policy`.
- Token-protected pages, JSON, errors, private responses, private chunk reads, and bundle downloads use `Cache-Control: no-store`.
- Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, raw viewer tokens, raw incident tokens, plaintext, or raw keys.
- Templates use Go `html/template` escaping.
- Storage rejects absolute paths, `..`, slash-containing path segments, and backslash traversal. S3 object keys are derived from server-controlled stored paths and an optional safe prefix.

## Incident Mode Risks To Preserve For Future Design

Future incident-mode work should treat these as explicit design risks rather than incidental frontend labels:

- non-emergency interaction records may include sensitive conversations with police, security, landlords, employers, service providers, or other authorities
- legal recording and sharing rules vary by jurisdiction
- sharing, export, publication, and legal submission are distinct actions and should not be collapsed into capture
- safety-check or dead-man switch notifications may create false alarms if timers, connectivity, or contact workflows are poorly designed
- trusted contacts need clear context and should decide whether to contact emergency services unless a future emergency-services integration is explicitly implemented
- account-owner, trusted-contact, admin/operator, public-link, and optional
  escrow access must be separated before public account systems exist; see
  [v1-access-control.md](v1-access-control.md)

The current backend does not implement incident-mode-specific controls yet, so future work must update this threat model before or alongside implementation.

## Known Limitations

- No implemented public authentication, user accounts, OAuth, JWT, sessions, or
  CSRF protection. The future `/v1` access-control design is planning-only in
  [v1-access-control.md](v1-access-control.md).
- Separate private/public ports reduce accidental route exposure, but they are not a complete security model.
- `/v1` must not be publicly exposed as-is.
- No iOS app, Android app, web client, local recording, production client key storage, key sharing, push notifications, SMS, Messenger integration, or public admin dashboard.
- No first-class incident types, escalation policies, interaction-record metadata, safety-check timers, dead-man switch notifications, or trusted-contact accounts.
- No built-in TLS, app-level rate limiting, abuse throttling, or IP allowlist.
- Optional PostgreSQL metadata does not change the private `/v1` boundary,
  token hashing, ciphertext-only storage, or backup/restore expectations
  described in [postgresql-metadata-migration.md](postgresql-metadata-migration.md).
  It also does not make the current upload flow cluster-safe on its own.
- Optional Valkey/Redis-compatible coordination does not change the private
  `/v1` boundary, does not hold durable evidence state, and does not make the
  current upload flow cluster-safe on its own.
- No implemented cluster-safe upload operation or idempotency API. Future
  semantics are planned in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md), but
  current duplicate uploads still use the existing `409 duplicate_chunk`
  behavior.
- No implemented resumable upload or upload lease protocol. Current clients
  should retry complete encrypted chunk uploads; the future design is planned
  in [resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md).
- Retention, backup, restore, and deletion policy is documented in
  [retention-backup-deletion.md](retention-backup-deletion.md), with future
  enforcement design in
  [incident-deletion-retention-enforcement.md](incident-deletion-retention-enforcement.md),
  but the backend does not yet implement automatic expiration, incident
  deletion APIs, built-in disk encryption, or object-bucket lifecycle policy
  enforcement.
- Cluster backup, restore, and failure runbooks are operational guidance only
  and do not make optional PostgreSQL, S3-compatible storage, or
  Valkey/Redis-compatible coordination production-cluster readiness by
  themselves.
- No malware/content scanning; uploaded bytes are assumed to be client-encrypted blobs.
- Bundle downloads are encrypted chunk bundles, not decrypted or playable media exports.
- No implemented multi-user authorization model.
- Viewer links are bearer tokens and must be shared carefully.
- No implemented production key-sharing, key recovery, Keychain storage, trusted-contact access, browser decryption, break-glass key access, or playable export. The future key custody and emergency access design is documented in [key-custody.md](key-custody.md), with browser decryption design in [browser-decryption.md](browser-decryption.md) and break-glass design in [break-glass-key-access.md](break-glass-key-access.md).

## Deployment Guidance

For local/private use, bind the private API server to localhost or a private network and restrict access with WireGuard, firewall rules, or a reverse proxy. If any part is exposed publicly today, expose only the incident viewer server. Future non-admin product routes may become public only after authenticated and authorized product API work exists. Future admin/operator routes should use a separately bound private admin API listener, configured for VPN or another private boundary where appropriate, while still requiring admin authentication. Inside Docker containers, bind to container addresses such as `0.0.0.0:8080` and restrict host exposure with port publishing, firewall rules, WireGuard, or reverse proxy configuration.

Use TLS at the edge for any network access. Apply deployment-edge rate limiting for public incident viewer routes and any private reverse-proxy boundary. Keep reverse-proxy logs, metrics, dashboards, and rate-limit keys from recording raw `/i/{token}` paths and pre-rename compatibility `/e/{token}` paths.

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP and MDN guidance expects HSTS only over HTTPS. Enable HSTS at the production HTTPS reverse proxy after the public hostname is consistently available over TLS.

## Next Security Steps

- Implement the explicit `/v1` access-control story from
  [v1-access-control.md](v1-access-control.md) before any public product API
  exposure or private admin API implementation.
- Design first-class incident types and escalation policies before implementing non-emergency interaction records, safety checks, or dead-man switch workflows.
- Define the future public product API and separately bound private admin API,
  including account-owner, trusted-contact, web-client, and admin/operator
  authorization boundaries.
- Tune deployment-edge rate limits for token guesses, uploads, downloads, and admin actions, and consider app-level rate limiting separately.
- Review viewer-token expiry tuning and revocation workflows.
- Implement the documented retention, backup, restore, and deletion operations.
- Prototype the documented hybrid key custody model without weakening the current ciphertext-only backend.
- Prototype browser decryption only after accepting the browser trust model and malicious-server limitations.
- Treat server-assisted break-glass access as an optional future mode only after explicit policy, audit, and deployment design.
- Review deployment logging so raw tokens are not captured outside the Go server.
