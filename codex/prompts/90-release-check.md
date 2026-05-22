# Codex Prompt: Release Check

Review the repo before tagging or publishing a release.

Do not add features.
Do not make broad refactors.
Do not change application behaviour unless required to fix a release-blocking bug.

## Goal

Confirm the repo is ready for a tagged release.

This is a final pre-release check for correctness, documentation, security warnings, tests, build metadata, and accidental committed junk.

## Project context

Safety Recorder is a Go backend for a private personal-safety recording system.

The current project shape includes:

- private `/v1` write/admin API listener group
- public read-only emergency viewer listener group
- SQLite metadata
- local disk encrypted chunk storage
- immutable chunk uploads
- media streams that can be marked `open`, `complete`, or `failed`
- completed encrypted stream and incident ZIP evidence bundle downloads
- emergency viewer tokens
- simulator CLI
- Docker image build
- GitHub Actions / GHCR publishing

Evidence bundles are encrypted chunk bundles, not decrypted or playable media exports.

## Release checklist

Check:

- all tests pass
- `gofmt` has been run
- `go vet` passes, if practical
- `README.md` version/scope is accurate
- `CHANGELOG.md` includes the release
- `docs/api.md` matches implemented routes
- `docs/code-map.md` matches package layout
- `docs/threat-model.md` or `docs/security-model.md` matches current security assumptions
- Docker/GHCR notes are current
- GitHub Actions workflow names and badges are correct
- environment variable docs match implementation
- plural bind address variables are documented:
  - `SAFE_PRIVATE_BIND_ADDRS`
  - `SAFE_PUBLIC_BIND_ADDRS`
- singular bind variables are documented only as backwards-compatible fallback, if still supported
- public/private listener separation is documented
- private `/v1` API exposure warnings are clear
- emergency viewer token behaviour is documented
- completed evidence bundle limitations are documented
- simulator commands still work
- no raw secrets/tokens are committed
- no generated binaries are committed
- no local SQLite database files are committed
- no uploaded blob data is committed
- no temporary files are committed
- no stale generated artifacts are committed
- no accidental `.env` files are committed

## Security review items

Confirm:

- private write/admin routes are not mounted on public viewer server
- public emergency viewer routes are read-only
- raw emergency tokens are not logged
- request bodies are not logged
- uploaded file bytes are not logged
- Authorization headers are not logged
- ZIP download routes do not expose filesystem paths
- ZIP entry names are controlled by the server
- ZIP downloads set safe headers
- token-protected emergency pages and downloads use `Cache-Control: no-store`
- emergency responses use `Referrer-Policy: no-referrer`
- emergency responses use `X-Content-Type-Options: nosniff`
- HSTS is not enabled by default for localhost/dev HTTP unless explicitly gated by config
- documentation does not claim production readiness

## Commands

From the repository root, run:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

If `go vet ./...` fails because of a known harmless issue, document the reason rather than silently ignoring it.

## Manual smoke tests

If practical, run the backend:

```bash
cd server
go run ./cmd/api
```

In another terminal:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

Confirm:

- the simulator creates an incident
- the simulator creates an emergency token
- chunks upload successfully
- checkins are sent
- the stream completes
- the emergency viewer URL works
- completed-stream download buttons appear
- encrypted bundle download works

## Release notes

If the release is ready, prepare or verify a `CHANGELOG.md` entry.

The entry should include:

- added features
- changed behaviour
- fixed bugs
- security-relevant changes
- documentation changes
- known limitations

Do not overstate stability or production readiness.

## Output format

Return:

1. Release readiness: ready / not ready
2. Blocking issues
3. Non-blocking issues
4. Tests and commands run
5. Documentation updates needed
6. Suggested version tag
7. Suggested changelog entry

If you make fixes:

- keep them minimal
- explain what changed
- run validation again
