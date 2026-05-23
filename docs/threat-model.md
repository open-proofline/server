# Threat Model

This document describes the current backend-only security posture. It is intentionally conservative and does not claim production readiness.

## Assets

- Already-encrypted uploaded chunk files under `SAFE_DATA_DIR`
- Incident, media stream, chunk, checkin, and emergency-token metadata in SQLite
- On-demand encrypted evidence ZIP bundles generated from completed streams
- Raw emergency tokens returned once at creation time
- Emergency viewer URLs containing bearer tokens
- Simulator-only local encryption key files when developers opt into `--key-file`
- Future iOS recordings, production client-side keys, key sharing, browser decryption, and break-glass key access are out of scope for the current implementation. The intended future key custody direction is documented in [key-custody.md](key-custody.md), browser decryption constraints are documented in [browser-decryption.md](browser-decryption.md), and server-assisted access design is documented in [break-glass-key-access.md](break-glass-key-access.md).

## Trust Boundaries

- The private API server binds separately from the public emergency viewer server. By default it listens on `127.0.0.1:8080`, and it can listen on multiple addresses through `SAFE_PRIVATE_BIND_ADDRS`.
- The public emergency viewer server binds separately from the private API server. By default it listens on `127.0.0.1:8081`, and it can listen on multiple addresses through `SAFE_PUBLIC_BIND_ADDRS`.
- `/v1` routes are private/admin routes. They can create incidents, create streams, upload chunks, complete/fail streams, close incidents, create emergency tokens, revoke tokens, and read encrypted bytes. They are mounted only on the private API server.
- `/e/{token}`, `/e/{token}/data`, and emergency bundle download routes are public-shaped read-only routes gated by an emergency token. They are mounted only on the public emergency viewer server.
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
- Emergency tokens use 256 bits from `crypto/rand`; only SHA-256 token hashes are stored.
- Expired, revoked, and invalid emergency tokens return the same public error.
- Emergency summaries do not expose `stored_path`. Emergency bundle downloads expose only encrypted chunk bytes and generated manifests for completed streams.
- ZIP bundle entry names are server-controlled and generated from metadata; clients do not provide stored paths for download.
- Emergency/public viewer responses use a strict same-origin `Content-Security-Policy` with `frame-ancestors 'none'`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, and a restrictive camera/microphone/geolocation `Permissions-Policy`.
- Token-protected pages, JSON, errors, private JSON, private chunk reads, and bundle downloads use `Cache-Control: no-store`.
- Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, or raw emergency tokens.
- Templates use Go `html/template` escaping.
- Storage rejects absolute paths, `..`, slash-containing path segments, and backslash traversal.

## Known Limitations

- No public authentication, user accounts, OAuth, JWT, sessions, or CSRF protection.
- Separate private/public ports reduce accidental route exposure, but they are not a complete security model.
- `/v1` must not be publicly exposed as-is.
- No iOS app, local recording, production client key storage, key sharing, push notifications, SMS, Messenger integration, or public admin dashboard.
- No built-in TLS, rate limiting, abuse throttling, or IP allowlist.
- No default emergency-token expiry policy; callers choose `expires_at`.
- No retention, backup, secure deletion, or disk encryption policy.
- No malware/content scanning; uploaded bytes are assumed to be client-encrypted blobs.
- Bundle downloads are encrypted chunk bundles, not decrypted or playable media exports.
- No multi-user authorization model.
- Emergency links are bearer tokens and must be shared carefully.
- No implemented production key-sharing, key recovery, Keychain storage, emergency-contact access, browser decryption, break-glass key access, or playable export. The future key custody and emergency access design is documented in [key-custody.md](key-custody.md), with browser decryption design in [browser-decryption.md](browser-decryption.md) and break-glass design in [break-glass-key-access.md](break-glass-key-access.md).

## Deployment Guidance

For local/private use, bind the private API server to localhost or a private network and restrict access with WireGuard, firewall rules, or a reverse proxy. If any part is exposed publicly, expose only the emergency viewer server unless `/v1` has an additional authenticated control plane in front of it. Inside Docker containers, bind to container addresses such as `0.0.0.0:8080` and restrict host exposure with port publishing, firewall rules, WireGuard, or reverse proxy configuration.

Use TLS at the edge for any network access. Keep reverse-proxy logs from recording raw `/e/{token}` paths.

The Go app does not set `Strict-Transport-Security` by default because local development uses plain HTTP and MDN guidance expects HSTS only over HTTPS. Enable HSTS at the production HTTPS reverse proxy after the public hostname is consistently available over TLS.

## Next Security Steps

- Add an explicit access-control story for `/v1`.
- Add rate limiting for token guesses, uploads, and admin actions.
- Decide default emergency-token expiry and revocation workflows.
- Define retention, backup, and deletion policy.
- Prototype the documented hybrid key custody model without weakening the current ciphertext-only backend.
- Prototype browser decryption only after accepting the browser trust model and malicious-server limitations.
- Treat server-assisted break-glass access as an optional future mode only after explicit policy, audit, and deployment design.
- Review deployment logging so raw tokens are not captured outside the Go server.
