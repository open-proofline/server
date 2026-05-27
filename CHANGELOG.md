# Changelog

## Unreleased

- Renamed the product in documentation to Proofline while preserving current repository, module, Docker, GHCR, route, and compatibility names.
- Documented the broader incident-capture direction, including emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes.
- Added `Phase 0` Deep Research prompt. Loads report instructions and plans research prior to running `Phase 1`
- Documented the `develop` and `release/v*` repository rulesets, branch model,
  and PR base-branch guidance.

## v0.5.0 - 2026-05-26

- Automated creating a minimal GitHub Release when needed and uploading the Linux amd64 binary as a Release asset for `v*` tag workflows.
- Added release binary and GHCR image artifact attestations to the CI workflow.
- Verified SQLite WAL startup by checking the returned journal mode and failing when WAL cannot be enabled.
- Aligned Docker base-image digest refresh documentation with the runtime Alpine tag family used by the Dockerfile.
- Pinned Docker base images by digest, added Dependabot Docker monitoring, and documented base-image digest refresh review steps.
- Broadened the Docker build-context ignore policy for local-only artifacts under `server/`.
- Pinned GitHub Actions workflow dependencies to full commit SHAs and documented the review process for action updates.
- Added an iOS local recorder prototype plan covering chunking, encrypted staging, retry behavior, and current stream API mapping.
- Added a retention, backup, restore, and secure deletion policy design document.
- Added deployment-edge rate-limiting guidance and Traefik route-group examples.
- Added deployment examples for localhost-only Docker, WireGuard/private-network `/v1` access, and Traefik HTTPS emergency viewer exposure.
- Added a configurable default 24-hour emergency-token expiry for omitted `expires_at` values.
- Added a public technical review report and report-validation prompt workflow.
- Documented the active branch protection ruleset, required checks, and tag/release expectations.
- Scoped GitHub Actions package write permission to the trusted Docker publish job while keeping workflow defaults read-only.
- Added break-glass, browser-decryption, and production key-custody design documents.
- Hardened streamed chunk identity, upload race handling, stream completion, schema migration tracking, and server timeout configuration.

## v0.5.0-rc.2 - 2026-05-26

- Added release binary and GHCR image artifact attestations to the CI workflow.
- Verified SQLite WAL startup by checking the returned journal mode and failing when WAL cannot be enabled.
- Aligned Docker base-image digest refresh documentation with the runtime Alpine tag family used by the Dockerfile.

## v0.5.0-rc.1 - 2026-05-25

- Pinned Docker base images by digest, added Dependabot Docker monitoring, and documented base-image digest refresh review steps.
- Broadened the Docker build-context ignore policy for local-only artifacts under `server/`.
- Pinned GitHub Actions workflow dependencies to full commit SHAs and documented the review process for action updates.
- Added an iOS local recorder prototype plan covering chunking, encrypted staging, retry behavior, and current stream API mapping.
- Added a retention, backup, restore, and secure deletion policy design document.
- Added deployment-edge rate-limiting guidance and Traefik route-group examples.
- Added deployment examples for localhost-only Docker, WireGuard/private-network `/v1` access, and Traefik HTTPS emergency viewer exposure.
- Added a configurable default 24-hour emergency-token expiry for omitted `expires_at` values.
- Added a public technical review report and report-validation prompt workflow.
- Documented the active branch protection ruleset, required checks, and tag/release expectations.
- Scoped GitHub Actions package write permission to the trusted Docker publish job while keeping workflow defaults read-only.
- Added a break-glass and dead-man-switch key access design document.
- Added a browser-side emergency viewer decryption design spike.
- Added a production key custody and emergency access design document covering the future hybrid trusted-contact model.
- Rejected non-positive chunk indexes for streamed uploads while preserving legacy unstreamed compatibility.
- Hardened chunk upload and stream completion against incident/stream state races.
- Added explicit schema migration tracking with `schema_migrations`.
- Added configurable private/public HTTP server timeout settings.
- Refactored streamed chunk identity and storage paths to be stream-scoped while preserving legacy unstreamed chunk reads.

## v0.4.0 - 2026-05-23

- Added documented v1 client-side chunk encryption envelope using AES-256-GCM for simulator/test flows.
- Updated the simulator to encrypt fake chunks by default, support local key files, and decrypt-verify downloaded stream bundles.
- Added non-secret bundle manifest encryption hints while keeping backend storage and downloads opaque.
- Split the simulator client into smaller files and added focused simulator helper tests.
- Added MDN-aligned security header test coverage for emergency pages, JSON responses, static assets, ZIP downloads, and private API JSON responses.
- Updated API, encryption, simulator, security, deployment, and code-map documentation to match the current backend.
- Known limitation: the backend still does not decrypt chunks, share keys, produce playable media exports, or provide public-production hardening.

## v0.3.0

- Added media streams for grouping uploaded chunks.
- Added stream completion/failure state transitions.
- Added encrypted ZIP evidence bundle downloads for completed streams and completed incident streams.
- Updated the emergency viewer to show completed-stream download buttons.
- Hardened emergency viewer and API response security headers.
- Hardened `SAFE_MAX_UPLOAD_BYTES` parsing and upload-limit overflow handling.
- Added AGPL-3.0-only license.
- Added repository security policy.
- Documented current security assumptions, private/public listener separation, Docker/GHCR publishing, and evidence-bundle limitations.
- Known limitation: evidence bundles remain encrypted chunk bundles, not decrypted or playable media exports, and the backend is not production-ready public infrastructure.

## v0.2.1

- Added multiple private/public bind address support.
- Updated documentation.
- Verified CI, binary build, and Docker image publishing.

## v0.2.0

- Added incident simulator client.

## v0.1.0

- Initial backend ingest API.
- Emergency viewer.
- Docker/GHCR publishing.
