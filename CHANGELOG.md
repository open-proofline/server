# Changelog

## Unreleased

- Introduced a narrow metadata repository interface around the existing SQLite
  incident repository implementation.
- Introduced a narrow blob-store interface around the existing local filesystem
  encrypted blob storage implementation.
- Added backend-selection configuration scaffolding for the current SQLite,
  local filesystem, and no-coordination backends while rejecting unsupported
  future backend values.
- Added a PostgreSQL metadata backend migration-path design covering schema
  parity, migrations, transaction boundaries, tests, and restore expectations.
- Added CI runtime smoke tests for the built Linux binary and Docker image.
- Added a public incident viewer deployment checklist covering public route
  exposure, TLS/HSTS, edge rate limiting, proxy log redaction, viewer-token
  review, and retention/restore expectations.
- Sanitized internal filesystem error logging

## v0.7.0 - 2026-05-28

- Moved the Go module and backend source tree to the repository root as
  `github.com/open-proofline/server`, and normalized new module, Docker, GHCR,
  and release binary artifact references after the `open-proofline/server`
  transfer.
- Updated CI, Docker, development, deployment, prompt, and report-workflow
  references for the repository-root server layout and `proofline-server-*`
  release artifacts.
- Updated the GitHub Actions `download-artifact` dependency while preserving
  full-SHA action pinning.
- Fixed the README Go version badge after the root-module migration.

## v0.6.1 - 2026-05-28

- Updated repository, GHCR badge, and prompt references after the
  `open-proofline/server` transfer.
- Targeted Dependabot updates to the `develop` integration branch for the
  post-release branch model.

## v0.6.0 - 2026-05-27

- Added CI vulnerability and coverage signals for release review, with release
  publishing gated on the vulnerability scan and coverage kept advisory.
- Hardened private API and public token-path security headers for unsupported
  method/error responses.
- Renamed legacy viewer/token terminology to incident-viewer and incident-token terminology, including breaking route/config/schema names for the upcoming release while migrating existing token rows.
- Retained legacy `/e/{token}` public viewer route aliases for already shared pre-rename links.
- Renamed the product in documentation to Proofline while preserving current repository, module, Docker, GHCR, route, and compatibility names.
- Updated active issue templates and reusable Codex prompts to match the
  Proofline product name.
- Documented the planned `open-proofline` multi-repo layout and clarified that this repository is the Go server backend only.
- Documented the broader incident-capture direction, including emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes.
- Added `Phase 0` Deep Research prompt. Loads report instructions and plans research prior to running `Phase 1`
- Documented Go readability standards and aligned the readability-maintenance Codex prompt with them.
- Refactored `server/cmd/api` server lifecycle helpers into a focused file without changing startup or listener behaviour.
- Refactored `server/cmd/simclient` simulator flow helpers into a focused file without changing CLI behaviour.
- Refactored `server/internal/config` bind-address, byte-size, timeout, and environment fallback parsing into focused files without changing configuration behaviour.
- Refactored `server/internal/db` connection, migration orchestration, and compatibility migration helpers into focused files for readability without changing migration behaviour.
- Refactored `server/internal/envelope` key-file, associated-data, chunk encryption, and header parsing helpers into focused files without changing the envelope format.
- Refactored `server/internal/httpapi` summary, bundle, stream-validation, and upload parsing helpers for readability without changing HTTP behaviour.
- Refactored `server/internal/incidents` repository methods into focused chunk, checkin, and incident-token files for readability without changing behaviour.
- Refactored `server/internal/storage` temp upload and immutable blob helpers into focused files for readability without changing storage behaviour.
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
- Added deployment examples for localhost-only Docker, WireGuard/private-network `/v1` access, and Traefik HTTPS incident viewer exposure.
- Added a configurable default 24-hour incident-token expiry for omitted `expires_at` values.
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
- Added deployment examples for localhost-only Docker, WireGuard/private-network `/v1` access, and Traefik HTTPS incident viewer exposure.
- Added a configurable default 24-hour incident-token expiry for omitted `expires_at` values.
- Added a public technical review report and report-validation prompt workflow.
- Documented the active branch protection ruleset, required checks, and tag/release expectations.
- Scoped GitHub Actions package write permission to the trusted Docker publish job while keeping workflow defaults read-only.
- Added a break-glass and dead-man-switch key access design document.
- Added a browser-side incident viewer decryption design spike.
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
- Updated the incident viewer to show completed-stream download buttons.
- Hardened incident viewer and API response security headers.
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
- Incident viewer.
- Docker/GHCR publishing.
