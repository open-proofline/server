# Codex Prompt: Release Check

Review the repo before tagging or publishing a release.

Do **not** add features.
Do **not** make broad refactors.
Do **not** change application behaviour unless required to fix a release-blocking bug.

## Goal

Confirm the repo is ready for a tagged release.

This is a final pre-release check for correctness, documentation, security warnings, tests, build metadata, release notes, and accidental committed junk.

## Source of truth

Before making changes, read current source-of-truth files as relevant:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- `docs/README.md`
- relevant files in `docs/`
- relevant source files
- relevant tests
- relevant issue or PR, if this is issue/PR work

Do not rely on stale assumptions from this prompt if the repository has changed.
## Global constraints

- Keep changes scoped to the task.
- Do not add unrelated features.
- Do not weaken security warnings.
- Do not claim production readiness.
- Do not expose `/v1` publicly.
- Do not log raw tokens, request bodies, uploaded bytes, Authorization headers, plaintext, raw keys, or future token-like values.
- Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, Docker Compose, Kubernetes, cloud integrations, or public admin dashboard features unless explicitly requested.
- Prefer Go standard library where practical.
- Preserve private/public listener separation.
- Preserve the current backend ciphertext-only implementation unless the task explicitly concerns key custody, emergency access, or decryption design.
- Do not introduce backend decryption, raw server-held decryption keys, key escrow, browser decryption, or key-sharing behaviour as an incidental implementation detail.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.

## Release checklist

Check:

- all tests pass
- `gofmt` has been run
- `go vet` passes, if practical
- `README.md` version/scope is accurate
- `CHANGELOG.md` includes the release
- `LICENSE` exists and matches the documented SPDX identifier
- `SECURITY.md` exists and does not promise production readiness
- `README.md` links to `LICENSE` and `SECURITY.md`
- `docs/api.md` matches implemented routes
- `docs/code-map.md` matches package layout
- `docs/encryption.md` matches envelope implementation
- `docs/key-custody.md` or equivalent design docs match current roadmap, if present
- simulator encryption defaults/key-file behaviour are documented
- `docs/threat-model.md` or `docs/security-model.md` matches current security assumptions
- `docs/codex-change-control.md` and `codex/README.md` match prompt workflow, if present
- issue/PR/backlog workflow prompts are listed in `codex/README.md`
- Docker/GHCR notes are current
- GitHub Actions workflow names and badges are correct
- environment variable docs match implementation
- bind address variables are documented
- public/private listener separation is documented
- private `/v1` API exposure warnings are clear
- emergency viewer token behaviour is documented
- completed evidence bundle limitations are documented
- simulator commands still work
- no raw secrets/tokens are committed
- no simulator key files are committed
- no generated binaries are committed
- no local SQLite database files are committed
- no uploaded blob data is committed
- no temporary files are committed
- no stale generated artifacts are committed
- no accidental `.env` files are committed
- `.backlog-drafts/` contents are intentional, or excluded if they are local-only drafts

## Security review items

Confirm:

- private write/admin routes are not mounted on public viewer server
- public emergency viewer routes are read-only
- raw emergency tokens are not logged
- request bodies are not logged
- uploaded file bytes are not logged
- Authorization headers are not logged
- plaintext and raw keys are not logged
- backend key custody/decryption posture is documented accurately
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
- local decrypt verification succeeds
- no plaintext or keys are printed

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
8. Any backlog follow-ups

If you make fixes:

- keep them minimal
- explain what changed
- run validation again
