# Threat Model

This document describes the current Proofline backend-only security posture. It is intentionally conservative and does not claim production readiness.

Planned future incident modes include emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. Those modes are not implemented yet. Current controls apply to generic incidents, encrypted chunk uploads, checkins, viewer tokens, and encrypted evidence bundles.

## Assets

- Already-encrypted uploaded chunk files under `SAFE_DATA_DIR`
- Incident, media stream, chunk, checkin, and viewer/incident-token metadata in SQLite
- Future optional PostgreSQL metadata is planned but not implemented; schema,
  migration, transaction, test, and restore expectations are documented in
  [postgresql-metadata-migration.md](postgresql-metadata-migration.md)
- Future cluster-safe upload operation semantics are planned but not
  implemented; idempotency, retry-success, conflict, and cleanup expectations
  are documented in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md)
- On-demand encrypted evidence ZIP bundles generated from completed streams
- Raw viewer/incident tokens returned once at creation time
- Incident viewer URLs containing bearer tokens
- Simulator-only local encryption key files when developers opt into `--key-file`
- Future mobile/web recordings, interaction-record metadata, safety-check state, trusted-contact access, production client-side keys, key sharing, browser decryption, and break-glass key access are out of scope for the current implementation. Planned incident modes are documented in [incident-modes.md](incident-modes.md), the intended future key custody direction is documented in [key-custody.md](key-custody.md), browser decryption constraints are documented in [browser-decryption.md](browser-decryption.md), and server-assisted access design is documented in [break-glass-key-access.md](break-glass-key-access.md).

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
- Final chunk storage uses no-overwrite hard links.
- The simulator encrypts fake chunk plaintext by default using the documented v1 AES-256-GCM envelope.
- Encryption keys remain client-side; they are not uploaded, stored in SQLite, or added to evidence bundles.
- SQLite enforces media type, chunk index, byte size, SHA-256 shape, foreign keys, and unique chunk identity.
- Media streams must be open before new chunks can be attached. The repository rechecks incident and stream state when chunk metadata is inserted.
- Stream completion verifies contiguous chunks plus readable stored files, and the repository revalidates chunk rows before committing the stream to `complete`.
- Viewer tokens use 256 bits from `crypto/rand`; only SHA-256 token hashes are stored. Tokens created without an explicit `expires_at` default to a 24-hour lifetime unless `SAFE_DEFAULT_INCIDENT_TOKEN_TTL` is configured differently.
- Expired, revoked, and invalid viewer tokens return the same public error.
- Incident summaries do not expose `stored_path`. Viewer bundle downloads expose only encrypted chunk bytes and generated manifests for completed streams.
- ZIP bundle entry names are server-controlled and generated from metadata; clients do not provide stored paths for download.
- Public viewer responses use a strict same-origin `Content-Security-Policy` with `frame-ancestors 'none'`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, and a restrictive camera/microphone/geolocation `Permissions-Policy`.
- Token-protected pages, JSON, errors, private responses, private chunk reads, and bundle downloads use `Cache-Control: no-store`.
- Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, raw viewer tokens, raw incident tokens, plaintext, or raw keys.
- Templates use Go `html/template` escaping.
- Storage rejects absolute paths, `..`, slash-containing path segments, and backslash traversal.

## Incident Mode Risks To Preserve For Future Design

Future incident-mode work should treat these as explicit design risks rather than incidental frontend labels:

- non-emergency interaction records may include sensitive conversations with police, security, landlords, employers, service providers, or other authorities
- legal recording and sharing rules vary by jurisdiction
- sharing, export, publication, and legal submission are distinct actions and should not be collapsed into capture
- safety-check or dead-man switch notifications may create false alarms if timers, connectivity, or contact workflows are poorly designed
- trusted contacts need clear context and should decide whether to contact emergency services unless a future emergency-services integration is explicitly implemented
- account-owner, trusted-contact, admin/operator, and public-link access must be separated before public account systems exist

The current backend does not implement incident-mode-specific controls yet, so future work must update this threat model before or alongside implementation.

## Known Limitations

- No public authentication, user accounts, OAuth, JWT, sessions, or CSRF protection.
- Separate private/public ports reduce accidental route exposure, but they are not a complete security model.
- `/v1` must not be publicly exposed as-is.
- No iOS app, Android app, web client, local recording, production client key storage, key sharing, push notifications, SMS, Messenger integration, or public admin dashboard.
- No first-class incident types, escalation policies, interaction-record metadata, safety-check timers, dead-man switch notifications, or trusted-contact accounts.
- No built-in TLS, app-level rate limiting, abuse throttling, or IP allowlist.
- No implemented PostgreSQL metadata backend. Any future PostgreSQL support must
  preserve the private `/v1` boundary, token hashing, ciphertext-only storage,
  and backup/restore expectations described in
  [postgresql-metadata-migration.md](postgresql-metadata-migration.md).
- No implemented cluster-safe upload operation or idempotency API. Future
  semantics are planned in
  [cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md), but
  current duplicate uploads still use the existing `409 duplicate_chunk`
  behavior.
- Retention, backup, restore, and deletion policy is documented in [retention-backup-deletion.md](retention-backup-deletion.md), but the backend does not yet implement automatic expiration, incident deletion APIs, or built-in disk encryption.
- No malware/content scanning; uploaded bytes are assumed to be client-encrypted blobs.
- Bundle downloads are encrypted chunk bundles, not decrypted or playable media exports.
- No multi-user authorization model.
- Viewer links are bearer tokens and must be shared carefully.
- No implemented production key-sharing, key recovery, Keychain storage, trusted-contact access, browser decryption, break-glass key access, or playable export. The future key custody and emergency access design is documented in [key-custody.md](key-custody.md), with browser decryption design in [browser-decryption.md](browser-decryption.md) and break-glass design in [break-glass-key-access.md](break-glass-key-access.md).

## Deployment Guidance

For local/private use, bind the private API server to localhost or a private network and restrict access with WireGuard, firewall rules, or a reverse proxy. If any part is exposed publicly, expose only the incident viewer server unless `/v1` has an additional authenticated control plane in front of it. Inside Docker containers, bind to container addresses such as `0.0.0.0:8080` and restrict host exposure with port publishing, firewall rules, WireGuard, or reverse proxy configuration.

Use TLS at the edge for any network access. Apply deployment-edge rate limiting for public incident viewer routes and any private reverse-proxy boundary. Keep reverse-proxy logs, metrics, dashboards, and rate-limit keys from recording raw `/i/{token}` paths and pre-rename compatibility `/e/{token}` paths.

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP and MDN guidance expects HSTS only over HTTPS. Enable HSTS at the production HTTPS reverse proxy after the public hostname is consistently available over TLS.

## Next Security Steps

- Add an explicit access-control story for `/v1`.
- Design first-class incident types and escalation policies before implementing non-emergency interaction records, safety checks, or dead-man switch workflows.
- Define account-owner, trusted-contact, web-client, and admin/operator authorization boundaries.
- Tune deployment-edge rate limits for token guesses, uploads, downloads, and admin actions, and consider app-level rate limiting separately.
- Review viewer-token expiry tuning and revocation workflows.
- Implement documented retention, backup, restore, and deletion operations.
- Prototype the documented hybrid key custody model without weakening the current ciphertext-only backend.
- Prototype browser decryption only after accepting the browser trust model and malicious-server limitations.
- Treat server-assisted break-glass access as an optional future mode only after explicit policy, audit, and deployment design.
- Review deployment logging so raw tokens are not captured outside the Go server.
