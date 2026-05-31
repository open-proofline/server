# Main API Public Exposure Listener Split Design

This document defines the planning boundary for a future listener split that
prepares Proofline's main API routes for public exposure without exposing admin
or operator surfaces.

This is not implemented yet. The current defaults still keep `/v1` and
`/admin` on the private API listener at `127.0.0.1:8080`, and the read-only
incident viewer on the public incident viewer listener at `127.0.0.1:8081`.
Do not expose the current `/v1` listener publicly as-is.

## Goals

- Define the target main `8080` listener route set.
- Define the target private `8081` admin-dashboard listener route set.
- Move the public incident viewer route group onto the main listener in a
  later implementation issue.
- Keep `/admin`, `/admin/...`, `/v1/admin/...`, private health/readiness, and
  first-admin bootstrap off the public-facing main listener.
- Define required app-level rate-limit route classes before any main API route
  is publicly exposed.
- Preserve token redaction, ciphertext-only storage, server-controlled ZIP
  paths, and public viewer read-only behavior.
- Document browser credential, CSRF, audit, logging, configuration, and test
  gates for implementation issues.

## Non-Goals

- No listener, route, configuration, schema, or auth behavior changes in this
  issue.
- No declaration that Proofline is production-ready public infrastructure.
- No OAuth, JWT, external identity provider, public account portal, public
  admin dashboard, web-client, iOS-client, Android-client, or protocol
  implementation.
- No backend decryption, browser decryption, raw server-held media keys, key
  escrow, break-glass access, or key-sharing behavior.
- No emergency-services integration, trusted-contact workflow, SMS, Messenger,
  email, push notification, or cloud deployment automation.

## Current Topology

The current server starts two handler trees:

| Listener group | Default address | Current routes | Current exposure |
|---|---:|---|---|
| Private API | `127.0.0.1:8080` | `/v1/...`, `/v1/admin/...`, `/v1/health/live`, `/v1/health/ready`, `/admin`, `/admin/...`, `/admin/static/...` | Localhost, LAN, WireGuard, firewall, or strict private reverse proxy only. |
| Public incident viewer | `127.0.0.1:8081` | `/i/{token}`, `/i/{token}/data`, `/i/{token}/streams/{stream_id}/download`, `/i/{token}/incident/download`, legacy `/e/{token}` aliases, `/static/...` | Public HTTPS/reverse proxy may expose this read-only viewer. |

Current `/v1` routes use local username/password accounts and opaque
server-side sessions. That is a private API control, not a complete public
product API security model.

## Target Listener Topology

The target implementation should keep two route trees, but their purpose should
change:

| Target listener | Target default address | Route groups | Exposure |
|---|---:|---|---|
| Main API and viewer | `127.0.0.1:8080` | Public-ready non-admin product API routes plus read-only incident viewer routes. | Public HTTPS/reverse proxy only after authentication, authorization, rate limits, logging redaction, browser credential rules, and deployment guidance are implemented and tested. |
| Private admin dashboard | `127.0.0.1:8081` | Private `/admin` dashboard routes, admin-only API routes, first-admin bootstrap, and private operator health/readiness checks. | Localhost, LAN, WireGuard, VPN, firewall, or strict private reverse proxy only. Still authenticated and authorized. |

The target `8080` main listener is public-facing only after the implementation
issue has added the required controls. It must not become a public catch-all for
every current private route.

The target `8081` private admin-dashboard listener must not serve public
incident viewer routes, product upload routes, public link routes, or general
account-owner product traffic.

## Target Main `8080` Route Set

The main listener may serve these route classes after public hardening exists:

| Route group | Target placement | Required controls before exposure |
|---|---|---|
| `/i/{token}`, `/i/{token}/data`, `/i/{token}/streams/{stream_id}/download`, `/i/{token}/incident/download` | Main listener. | Keep read-only, bearer-token scoped, fail-closed, no-store, strict browser headers, route-pattern logging, and viewer rate limits. |
| Legacy `/e/{token}` aliases | Main listener while compatibility is retained. | Same as `/i/...`; prefer canonical `/i/...` for new links. |
| `/static/...` incident viewer assets | Main listener. | Token-neutral static assets only; no incident data, tokens, keys, or private deployment details. |
| `/v1/auth/login`, `/v1/auth/logout` | Main listener only if treated as public product authentication. | Per-route login abuse limits, audit, redacted errors, TLS, browser credential decision, and tests that the returned session cannot reach absent admin routes on the main listener. |
| `/v1/account`, `/v1/account/password` | Main listener as account-owner product routes. | Authenticated account scope, password-change rate limits, session revocation behavior, and CSRF protection if browser cookies are used. |
| `/v1/incidents`, `/v1/incidents/{incident_id}`, `/v1/incidents/{incident_id}/close`, owner-scoped `/v1/incidents/{incident_id}/deletion` | Main listener as account-owner product routes. | Owner/admin authorization review for public use, action/data-class policy, incident-mode non-escalation guarantees, audit, route limits, and deletion fail-closed tests. |
| `/v1/incidents/{incident_id}/chunks`, `/v1/incidents/{incident_id}/chunks/reconcile`, chunk metadata and private chunk byte reads | Main listener as capture/account-owner product routes. | Body-size limits, upload rate limits, idempotency-key redaction, reconciliation response limits, immutable chunk guards, and slow-upload timeout review. |
| `/v1/incidents/{incident_id}/streams`, stream state routes, private stream/incident encrypted bundle downloads | Main listener as account-owner product routes. | State-transition authorization, download limits, no-store ZIP headers, server-controlled ZIP entry names, and encrypted-bundle wording. |
| `/v1/incidents/{incident_id}/checkins` | Main listener as capture/account-owner product route. | Check-in rate limits, body limits, actor binding, and no notification side effects from labels alone. |
| `/v1/incidents/{incident_id}/incident-tokens`, `/v1/incident-tokens/{token_id}/revoke` | Main listener as account-owner sharing routes. | Grant-creation limits, token hash storage, raw token returned once, token-label redaction guidance, revoke audit, and no admin/operator grant management on the main listener. |

These routes should be explicit public product API routes. Implementation should
avoid a mechanical move of the current private mux to the main listener.

## Target Private `8081` Route Set

The private admin-dashboard listener should serve only private admin or
operator route classes:

| Route group | Target placement | Notes |
|---|---|---|
| `/admin`, `/admin/login`, `/admin/bootstrap`, `/admin/logout`, `/admin/password`, `/admin/accounts/{account_id}/password` | Private `8081` only. | Keep HttpOnly SameSite admin cookie scoped to `/admin`, session-bound CSRF tokens for state-changing forms, no-store, and conservative browser headers. |
| `/admin/static/...` | Private `8081` only. | Token-neutral admin CSS only; no incident evidence, tokens, keys, or deployment details. |
| `/v1/admin/accounts`, `/v1/admin/accounts/{account_id}/password`, `/v1/admin/accounts/{account_id}/sessions/revoke` | Private `8081` only unless renamed to a private admin API prefix. | Do not mount these account-management routes on the main listener. |
| `GET` and `POST /v1/admin/incidents/{incident_id}/deletion` | Private `8081` only unless renamed to a private admin API prefix. | Admin-global incident deletion remains a private operator action. |
| `/v1/bootstrap/admin` | Private `8081` only. | First-admin bootstrap must never be mounted on the main public-facing listener; remove the bootstrap secret after use. |
| `/v1/health/live`, `/v1/health/ready` | Private `8081` only unless a later issue designs a separate public-safe health route. | Readiness is coarse but still operator-facing; do not publish selected backend status on the main public origin. |

If future implementation renames admin API routes from `/v1/admin/...` to a
new private prefix, it should keep compatibility or migration guidance explicit
and keep the private route tree separate from the main listener.

## Explicit Exclusions From The Main Listener

The main listener must not mount:

- `/admin` or `/admin/...`
- `/v1/admin/...`
- `/v1/bootstrap/admin`
- `/v1/health/live` or `/v1/health/ready`
- operator maintenance commands or status pages
- migration, backup, restore, deletion-worker, or support routes
- escrow, break-glass, backend decryption, browser decryption, or raw-key routes
- public dashboard routes for admin/operator workflows

Public incident viewer routes must remain read-only. They must not create,
revoke, extend, or manage viewer tokens; change incident state; expose
deletion controls; expose grant controls; expose admin/operator state; release
wrapped keys; decrypt evidence; or return raw media keys.

## Rate-Limit Route Classes

Before any main API route is exposed beyond a private boundary, implementation
must add app-level route-class limits in addition to deployment-edge controls.
The current backend implements main API route-class limits on the existing
`/v1` handler tree without changing listener defaults. Those limits are a
prerequisite for future exposure, not a complete public security model. At
minimum:

| Class | Example routes | Purpose |
|---|---|---|
| Viewer page lookup | `GET /i/{token}`, `GET /e/{token}` | Slow bearer-token guessing and token enumeration. |
| Viewer data polling | `GET /i/{token}/data`, `GET /e/{token}/data` | Bound polling and refresh traffic. |
| Viewer download | Viewer stream and incident ZIP downloads | Protect bundle generation and storage reads. |
| Static asset | `/static/...` | Keep asset floods from bypassing route accounting. |
| Login/auth | `/v1/auth/login`, `/v1/auth/logout` | Slow password guessing and session churn. |
| Account/password | `/v1/account`, `/v1/account/password` | Bound password change and account self-service traffic. |
| Incident metadata write | Incident create, close, deletion, token creation/revocation | Bound state changes and grant creation. |
| Incident metadata read | Incident, stream, chunk, check-in metadata reads | Bound authenticated metadata scraping. |
| Upload body | Chunk uploads and future resumable upload routes | Protect request body handling, temp storage, hashing, and metadata writes. |
| Upload reconciliation/idempotency | Duplicate reconciliation and idempotent retry paths | Prevent metadata comparison and replay endpoints from becoming enumeration tools. |
| Private/API download | Private chunk bytes and authenticated bundle downloads | Protect storage reads and ZIP generation for authenticated callers. |
| Admin login and admin actions | Private `/admin` and `/v1/admin/...` | Keep private admin abuse controls separate from public product controls. |

Limiter keys must be server-controlled and must not include raw viewer tokens,
raw session tokens, Authorization headers, request bodies, uploaded bytes,
idempotency keys, plaintext, raw keys, stored paths, object keys, private
deployment details, or user safety narrative. When Valkey/Redis-compatible
coordination is configured, rate-limit keys should remain short-lived
coordination state and not durable evidence metadata.

## Browser Credentials And CSRF

Implementation must decide credential semantics per route group:

- Bearer Authorization sessions avoid browser CSRF by not relying on automatic
  cookie attachment, but they still require XSS, storage, referrer, and log
  redaction review.
- Browser cookie sessions on public product routes require `HttpOnly`,
  `Secure` on HTTPS, `SameSite` policy, CSRF tokens for every authenticated
  state-changing request, no-store responses, and tests for missing or invalid
  CSRF tokens.
- The existing admin web cookie remains scoped to `/admin` and private to the
  admin-dashboard listener.
- Public incident viewer bearer-token GET routes must stay read-only and
  should continue using `Referrer-Policy: no-referrer`, `Cache-Control:
  no-store`, `X-Content-Type-Options: nosniff`, restrictive
  `Permissions-Policy`, and strict CSP with `frame-ancestors 'none'`.

Do not add a broad browser account portal or public admin dashboard as part of
the listener split.

## Logging, Audit, And Redaction

The main listener will receive public traffic, so implementation must preserve
and extend the current redaction posture:

- Log route patterns instead of token-bearing paths.
- Redact canonical `/i/{token}` and legacy `/e/{token}` paths before token
  lookup.
- Do not log request bodies, uploaded bytes, Authorization headers, raw session
  tokens, raw viewer tokens, raw incident tokens, raw idempotency keys,
  plaintext, raw keys, stored paths, object keys, private deployment details,
  original filenames, location values, notes, or user safety narrative.
- Public errors should remain small and should not reveal whether a token is
  invalid, expired, revoked, deleting, deleted, or blocked by policy.
- Audit records for public product API actions should use non-sensitive actor,
  action, route class, incident, grant, and outcome identifiers.
- Admin/operator audit should stay private and must not expose plaintext, raw
  keys, raw tokens, or evidence bytes casually.

## Configuration Migration

Current configuration names describe the current topology:

- `SAFE_PRIVATE_BIND_ADDRS` serves `/v1` and `/admin`
- `SAFE_PUBLIC_BIND_ADDRS` serves the public incident viewer

The implementation issue should either introduce clearer names such as
`SAFE_MAIN_BIND_ADDRS` and `SAFE_ADMIN_BIND_ADDRS`, or explicitly document how
the existing names map during a compatibility period. It must update
configuration docs, deployment docs, Docker examples, simulator viewer-base
examples, and tests together.

The target default ports are:

| Target role | Target address |
|---|---:|
| Main API and viewer | `127.0.0.1:8080` |
| Private admin dashboard | `127.0.0.1:8081` |

Container examples should continue binding to container addresses and relying
on host port publishing, firewall rules, WireGuard, or reverse proxy routing to
control real exposure. The docs must not imply that binding to separate ports
is a complete security model.

## Implementation Test Requirements

Follow-up implementation issues should include tests that prove:

- main listener serves the intended public-ready product API routes
- main listener serves `/i/...`, legacy `/e/...`, and `/static/...`
- main listener returns not-found for `/admin`, `/admin/...`, `/v1/admin/...`,
  `/v1/bootstrap/admin`, `/v1/health/live`, and `/v1/health/ready`
- private admin-dashboard listener serves `/admin`, `/admin/...`,
  `/admin/static/...`, private health/readiness, bootstrap, and admin API
  routes
- private admin-dashboard listener does not serve public viewer routes or
  account-owner upload/product routes
- public incident viewer routes remain read-only
- route-class rate limits cover viewer, auth, upload, metadata, sharing, and
  download routes before exposure
- browser-facing main and admin responses keep MDN-aligned security headers
  and no-store behavior where appropriate
- ZIP downloads keep `Content-Type: application/zip`, attachment disposition,
  no-store behavior, safe server-controlled entry names, and no filesystem or
  object-key exposure
- request logging uses redacted route patterns and does not include sensitive
  values
- simulator smoke tests use the updated main/viewer base URL after listener
  defaults change

Until those tests and docs land, the current deployment rule remains:
`/v1` and `/admin` stay private, and only the read-only incident viewer may be
considered for public HTTPS exposure.
