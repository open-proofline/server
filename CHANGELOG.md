# Changelog

## Unreleased

## v0.3.0

- Added media streams for grouping uploaded chunks.
- Added stream completion/failure state transitions.
- Added encrypted ZIP evidence bundle downloads for completed streams and completed incident streams.
- Updated the emergency viewer to show completed-stream download buttons.
- Updated the simulator to create and complete streams and optionally verify bundle download.
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
