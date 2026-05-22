# Codex Prompt: Security Review

Review the backend for security issues.

Do not add new features.
Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, Docker Compose, Kubernetes, cloud integrations, or public admin dashboards.

## Project context

The backend has two server surfaces:

1. Private API server
   - `/v1` write/admin/upload endpoints
   - intended for localhost, LAN, WireGuard, firewall, or strict reverse proxy boundary
   - must not be publicly exposed

2. Public emergency viewer server
   - read-only token-scoped emergency viewer
   - intended for trusted contacts through HTTPS/reverse proxy
   - must not expose private write/admin routes

The app stores already-encrypted chunks and serves encrypted ZIP evidence bundles for completed media streams/incidents.

## Review focus

Check:

- public route surface
- private `/v1` write/admin route exposure
- public/private mux separation
- emergency token generation
- token hashing
- token expiry and revocation
- raw token logging
- token leakage through URLs/referrers/logs
- upload size limits
- SHA-256 verification
- temp file cleanup
- chunk overwrite prevention
- SQLite constraints
- media stream completion validation
- uploads to completed/failed streams
- ZIP bundle download authorization
- ZIP path traversal
- ZIP entry name sanitization
- filesystem path exposure
- download response headers
- `Cache-Control: no-store`
- `Referrer-Policy: no-referrer`
- `X-Content-Type-Options: nosniff`
- HTML template escaping
- panic/log leakage
- Docker bind exposure risks
- plural bind address parsing
- stale documentation that overpromises production readiness

## Output format

Return findings with severity:

- Critical
- High
- Medium
- Low
- Informational

For each finding, include:

- what is wrong
- why it matters
- minimal recommended fix
- affected files/routes if known

If fixes are needed, make only minimal fixes and run:

```bash
gofmt -w .
go test ./...
```

Do not make broad refactors during a security review unless required to fix a critical issue.
