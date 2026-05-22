# Codex Prompt: MDN-Aligned Web Security Header Review

Review the Go backend's public emergency viewer and private API response headers against Mozilla MDN web security guidance.

Do not add features.
Do not add React, Node, npm, OAuth, JWT, user accounts, Docker Compose, Kubernetes, or cloud integrations.
Do not change endpoint behaviour unless required to fix a security bug.

## Goal

Ensure the web server's security headers and browser-facing behaviour are consistent with MDN-documented best practices for a small server-rendered emergency viewer.

Use MDN guidance conceptually for:

- Content-Security-Policy
- X-Content-Type-Options
- Referrer-Policy
- Permissions-Policy
- Strict-Transport-Security
- X-Frame-Options or CSP `frame-ancestors`
- Cache-Control for token-protected emergency pages/downloads

## Important project context

The app has two server surfaces:

1. Private API server
   - write/admin/upload endpoints
   - intended for localhost, LAN, or WireGuard/private network
   - must not be publicly exposed

2. Public emergency viewer server
   - read-only token-scoped viewer
   - intended to be exposed through HTTPS/reverse proxy
   - must not expose private write/admin routes

## Review requirements

Check:

- public emergency HTML responses
- public emergency JSON responses
- ZIP bundle download responses
- private API JSON responses
- 404/error responses
- static CSS responses, if present

Look for:

- missing `Content-Type`
- missing `X-Content-Type-Options: nosniff`
- missing or weak `Referrer-Policy`
- missing or weak `Content-Security-Policy`
- missing `frame-ancestors` or equivalent clickjacking protection
- unsafe inline scripts/styles
- token leakage through referrers
- token leakage through logs
- cacheable token-protected emergency pages
- cacheable emergency downloads
- incorrect headers on ZIP downloads
- headers that should be set by reverse proxy rather than app

## Implementation guidance

For emergency viewer HTML, prefer a strict CSP such as:

```http
Content-Security-Policy: default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'
```

If inline CSS is currently used, either:

- move it to a static CSS file and keep CSP strict, or
- explicitly document why `style-src 'unsafe-inline'` is temporarily used.

Do not add inline JavaScript unless explicitly requested.

Set:

```http
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
Permissions-Policy: geolocation=(), microphone=(), camera=()
Cache-Control: no-store
```

For ZIP downloads, set:

```http
Content-Type: application/zip
Content-Disposition: attachment
X-Content-Type-Options: nosniff
Cache-Control: no-store
Referrer-Policy: no-referrer
```

For HSTS:

- Do not enable HSTS by default for localhost/dev HTTP.
- Document that HSTS should be enabled only when served over production HTTPS.
- If implemented in-app, gate it behind an explicit env var such as `SAFE_ENABLE_HSTS=true`.

## Tests

Add or update tests to verify security headers for:

- emergency viewer HTML
- emergency viewer JSON data endpoint
- stream/incident ZIP download responses
- private API JSON responses where appropriate
- 404/error responses

Tests should confirm:

- private routes are not mounted on public server
- public emergency pages use `Cache-Control: no-store`
- public emergency pages use `Referrer-Policy: no-referrer`
- public emergency pages use `X-Content-Type-Options: nosniff`
- CSP is present on emergency viewer HTML
- download responses use attachment disposition and no-store cache policy

## Documentation

Update README or `docs/security-model.md` with:

- which headers are set by the Go app
- which headers should be set by reverse proxy
- why HSTS is not enabled by default in local/dev mode
- how to test the public site with MDN HTTP Observatory after deployment

## Validation

Run:

```bash
gofmt -w .
go test ./...
```

## Summary after implementation

Summarize:

- headers added or changed
- tests added
- documentation updated
- any MDN-related security items intentionally deferred to reverse proxy/deployment
