# AGENTS.md

## Project rules

- Keep the backend small, boring, and testable.
- Prefer Go standard library where practical.
- Do not add React, Node, npm, Docker Compose, Kubernetes, OAuth, JWT, user accounts, SMS, Messenger, push notifications, cloud services, or public admin dashboards unless explicitly requested.
- Treat uploaded chunks as immutable.
- Never overwrite stored chunks or evidence bundle contents.
- Never log raw emergency tokens, request bodies, uploaded file bytes, Authorization headers, or future token-like values.
- Keep private `/v1` write/admin routes and public emergency viewer routes on separate listener groups and separate muxes.
- Do not mount private write/admin routes on public emergency viewer servers.
- Emergency viewer routes must remain read-only.
- ZIP bundle download routes must not expose filesystem paths or accept client-provided stored paths.
- Generated ZIP entry names must be controlled by the server.
- Completed evidence bundles are encrypted chunk bundles, not decrypted/playable media exports.
- Use stable, documented crypto libraries only. Do not implement cryptographic primitives. Do not create custom AEAD, block modes, padding, MAC, KDF, or random generator logic.
- Preserve the current deployment model: private API behind localhost/LAN/WireGuard/firewall; public emergency viewer behind HTTPS/reverse proxy when exposed.
- Separate bind addresses are a deployment boundary, not a complete security model.

## Current project shape

- Go backend only.
- SQLite metadata.
- Local disk blob storage.
- Private API listener group for `/v1` routes.
- Public emergency viewer listener group for `/e/{token}` routes.
- Uploaded chunks may be grouped into media streams.
- Media streams can be marked `open`, `complete`, or `failed`.
- Completed streams and incidents can be downloaded as encrypted ZIP evidence bundles.
- Simulator CLI exists for incident upload/check-in flows.
- Docker and GitHub Actions/GHCR publishing exist, but deployment expansion should not be added unless explicitly requested.

## Commands

From `server/`:

```bash
gofmt -w .
go test ./...
```

Use `go vet ./...` when reviewing larger changes:

```bash
go vet ./...
```

## Review expectations

Before accepting Codex changes, check:

- tests pass
- generated code stays in scope
- private/public route separation is preserved
- raw tokens are not logged
- ZIP downloads use safe headers and controlled paths
- documentation still matches `README.md`
- no public-production readiness is implied unless deployment hardening has actually been implemented
