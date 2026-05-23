# Security Model

This document summarizes the current security assumptions and controls. For a threat-oriented view, see [threat-model.md](threat-model.md). For future production key custody and emergency access design, see [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md). For vulnerability reporting, see [../SECURITY.md](../SECURITY.md).

## Maturity

Safety Recorder is experimental and not production-ready public infrastructure. The private `/v1` API has no public user authentication, no user accounts, no OAuth, and no JWT protection.

## Listener Boundary

The API binary starts separate listener groups:

| Listener group | Routes | Intended exposure |
|---|---|---|
| Private API | `/v1/...` | Localhost, LAN, WireGuard, firewall, or strict reverse proxy only. |
| Public emergency viewer | `/e/{token}` and related read-only routes | HTTPS/reverse proxy when exposed. |

Private write/admin routes must not be mounted on public emergency viewer listeners. Emergency viewer routes are read-only.

## Token Handling

Emergency viewer tokens are scoped to one incident. The raw token is returned only at creation time; SQLite stores only a SHA-256 hash. Expired, revoked, and invalid tokens return the same public error.

Emergency URLs contain bearer tokens and should be treated as secrets. Reverse proxies and operational logs should avoid recording raw `/e/{token}` paths.

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

Bundles contain encrypted chunk bytes and JSON manifests only. They are not decrypted, playable, or merged media exports.

Bundle manifests may include a non-secret client-side encryption hint. They do not include keys.

## Logging And Headers

Request logging records method, redacted route pattern, status, byte count, and duration. It does not log request bodies, uploaded bytes, Authorization headers, or raw emergency tokens.

The Go app sets these headers on public emergency viewer pages, JSON responses, static assets, and ZIP downloads:

- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `X-Frame-Options: DENY`

Token-protected emergency pages, JSON responses, errors, and ZIP downloads also use `Cache-Control: no-store`. Private API JSON responses use `Content-Type: application/json`, `X-Content-Type-Options: nosniff`, and `Cache-Control: no-store`.

HSTS is not enabled by default in the Go app because local development uses plain HTTP and HSTS should only be sent over HTTPS. Set `Strict-Transport-Security` at the production HTTPS reverse proxy after TLS is established for the public hostname. After deployment, test the public emergency viewer with the MDN HTTP Observatory.

HTTP server timeouts are configurable separately for private and public listener groups. Private read/write timeouts default to disabled for large uploads/downloads; public viewer timeouts are finite by default and should be coordinated with reverse-proxy timeouts.

## Known Security Gaps

- No public authentication or authorization model for `/v1`
- No built-in TLS
- No built-in rate limiting or abuse throttling
- No default emergency-token expiry policy
- No implemented production client key storage, key sharing, browser decryption, server-assisted break-glass key access, or emergency-contact key access model; the future designs are documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- No retention, backup, secure deletion, or disk encryption policy
- No malware/content scanning for uploaded encrypted blobs
- No multi-user authorization model
