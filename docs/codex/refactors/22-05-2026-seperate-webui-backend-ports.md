Refactor the Go backend so the private write/admin API and the public emergency viewer run on separate HTTP servers and separate ports.

Do not add new product features.

Goal:
- Private API and emergency viewer must use separate http.ServeMux instances.
- Private API must not be mounted on the public viewer server.
- Public emergency viewer must not be able to mutate incidents, chunks, checkins, or tokens.
- Keep one Go binary for now.
- Keep one SQLite database and one local blob storage root.
- Do not add React, Node, npm, Docker, Kubernetes, OAuth, JWT, or user accounts.

Configuration:
Replace or extend SAFE_BIND_ADDR with:

- SAFE_PRIVATE_BIND_ADDR, default "127.0.0.1:8080"
- SAFE_PUBLIC_BIND_ADDR, default "127.0.0.1:8081"

Keep:

- SAFE_DATA_DIR
- SAFE_DB_PATH
- SAFE_MAX_UPLOAD_BYTES

Private server:
Mount only private/dev/write/admin routes:

- POST /v1/incidents
- GET /v1/incidents/{incident_id}
- POST /v1/incidents/{incident_id}/chunks
- GET /v1/incidents/{incident_id}/chunks
- GET /v1/incidents/{incident_id}/chunks/{media_type}/{chunk_index}
- POST /v1/incidents/{incident_id}/checkins
- POST /v1/incidents/{incident_id}/close
- POST /v1/incidents/{incident_id}/emergency-tokens
- POST /v1/emergency-tokens/{token_id}/revoke

Public viewer server:
Mount only read-only emergency routes:

- GET /e/{token}
- GET /e/{token}/data

Security requirements:
- Do not log raw emergency tokens.
- Do not expose private routes on the public server.
- Do not expose public emergency routes on the private server unless explicitly needed for local testing.
- Keep existing token expiry/revocation behaviour.
- Keep HTML template escaping.
- Add tests proving private routes return 404 on the public server.
- Add tests proving public emergency routes return 404 on the private server if not mounted there.
- Add tests proving emergency viewer routes cannot mutate incident/chunk/checkin/token state.

Server lifecycle:
- Start both HTTP servers from cmd/api/main.go.
- Use graceful shutdown for both servers on SIGINT/SIGTERM.
- Log both bind addresses on startup.
- Keep request logging middleware, but make sure raw tokens are redacted or not logged.

Documentation:
Update README and docs/code-map.md to explain:
- private API server
- public emergency viewer server
- intended WireGuard/private deployment for write API
- intended reverse proxy/TLS deployment for emergency viewer
- warning that separate ports are a deployment boundary, not a complete security model

After implementation:
- run gofmt
- run go test ./...