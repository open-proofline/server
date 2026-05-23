# AGENTS.md

## Project rules

- Keep the backend small, boring, and testable.
- Prefer Go standard library where practical.
- Do not add React, Node, npm, Docker Compose, Kubernetes, OAuth, JWT, user accounts, SMS, Messenger, push notifications, cloud services, or public admin dashboards unless explicitly requested.
- Treat uploaded chunks as immutable.
- Never overwrite stored chunks or evidence bundle contents.
- Never log raw emergency tokens, request bodies, uploaded file bytes, Authorization headers, plaintext, raw keys, or future token-like values.
- Keep private `/v1` write/admin routes and public emergency viewer routes on separate listener groups and separate muxes.
- Do not mount private write/admin routes on public emergency viewer servers.
- Emergency viewer routes must remain read-only.
- ZIP bundle download routes must not expose filesystem paths or accept client-provided stored paths.
- Generated ZIP entry names must be controlled by the server.
- Completed evidence bundles are encrypted chunk bundles, not decrypted/playable media exports.
- Use stable, documented crypto libraries only. Do not implement cryptographic primitives. Do not create custom AEAD, block modes, padding, MAC, KDF, or random generator logic.
- Preserve the current backend ciphertext-only implementation unless a task explicitly concerns key custody, emergency access, or decryption design.
- Do not introduce backend decryption, raw server-held decryption keys, key escrow, browser decryption, or key-sharing behaviour incidentally.
- Key custody/decryption changes must be explicit security-sensitive work and update the threat model, security model, encryption docs, tests, and operational guidance before or alongside implementation.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.
- Preserve the current deployment model: private API behind localhost/LAN/WireGuard/firewall; public emergency viewer behind HTTPS/reverse proxy when exposed.
- Separate bind addresses are a deployment boundary, not a complete security model.
- Treat Codex prompts as scoped change requests, not open-ended permission to expand the project.
- Do not implement newly discovered future work during an unrelated task; document it as an issue/backlog item instead.
- For larger changes, start from a clean working tree or an explicit checkpoint commit.
- Backlog scanning should create draft Markdown files first, not GitHub issues directly.
- Do not create public GitHub issues from backlog drafts until the maintainer has reviewed them.
- Never put raw tokens, secrets, private deployment details, exploit details, or user safety data into public issue drafts.

## Current project shape

- Go backend only.
- SQLite metadata.
- Local disk blob storage.
- Private API listener group for `/v1` routes.
- Public emergency viewer listener group for `/e/{token}` routes.
- Uploaded chunks may be grouped into media streams.
- Media streams can be marked `open`, `complete`, or `failed`.
- Completed streams and incidents can be downloaded as encrypted ZIP evidence bundles.
- Simulator CLI exists for incident upload/check-in/encryption test flows.
- The current simulator encryption envelope is development/test oriented.
- Future encryption direction should be a hybrid key custody model.
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
- plaintext and raw keys are not logged
- ZIP downloads use safe headers and controlled paths
- documentation still matches `README.md`
- key custody/decryption changes are explicit and security-reviewed
- no public-production readiness is implied unless deployment hardening has actually been implemented
