Add a minimal server-rendered emergency viewer to the Go backend.

Do not add React, Node, npm, Vite, frontend frameworks, OAuth, JWT, or user accounts.

Goal:
Create a public read-only emergency viewer that can show incident status, checkins, and chunk metadata using a high-entropy random access token.

Security:
- This is the first public-facing surface.
- Do not expose existing private /v1 write/admin endpoints publicly.
- Emergency tokens must be scoped to one incident.
- Emergency tokens must be read-only.
- Store only SHA-256 hashes of emergency tokens in SQLite, not raw tokens.
- Tokens must support expiry and revocation.
- Do not log raw tokens.
- Do not put raw tokens in structured logs.
- Do not allow emergency routes to create incidents, upload chunks, close incidents, or mutate data.
- Do not implement user accounts.

Database:
Add a table:

emergency_tokens:
- id
- incident_id
- token_hash
- label nullable
- created_at
- expires_at nullable
- revoked_at nullable
- last_used_at nullable

Add repository methods:
- CreateEmergencyToken(incidentID, label, expiresAt) returns raw token once
- LookupEmergencyToken(rawToken) returns token metadata and incident ID if valid
- RevokeEmergencyToken(tokenID)
- UpdateEmergencyTokenLastUsed(tokenID)

Token requirements:
- Generate at least 32 random bytes.
- Encode using URL-safe base64 without padding.
- Hash token using SHA-256 before storing.
- Use constant-time comparison where relevant.

Routes:
- POST /v1/incidents/{incident_id}/emergency-tokens
  Creates an emergency read-only token for an incident.
  This remains private/dev API for now.

- POST /v1/emergency-tokens/{token_id}/revoke
  Revokes an emergency token.
  This remains private/dev API for now.

- GET /e/{token}
  Server-rendered HTML page showing:
  - incident status
  - client label
  - created_at
  - updated_at
  - latest checkin
  - battery percent if available
  - network if available
  - location if available
  - chunk count by media type
  - latest chunk per media type
  - clear warning to call emergency services if concerned

- GET /e/{token}/data
  JSON endpoint returning the same read-only data for polling.

Frontend:
- Use Go html/template.
- Add simple CSS.
- No frontend build system.
- No external JS dependencies.
- Optional tiny vanilla JS polling every 10 seconds from /e/{token}/data.
- Keep the page readable on mobile.

Tests:
Add tests for:
- creating emergency token
- raw token is not stored
- valid token can read incident data
- expired token is rejected
- revoked token is rejected
- invalid token is rejected
- emergency token cannot mutate incident/chunk/checkin data
- /e/{token}/data returns expected read-only JSON

Documentation:
Update README with:
- emergency viewer warning
- how to create a token
- how to open the emergency page
- warning that public deployment still needs TLS, rate limiting, logging review, and careful firewall/reverse proxy config.

After implementation:
- run gofmt
- run go test ./...
- do not add unrelated features