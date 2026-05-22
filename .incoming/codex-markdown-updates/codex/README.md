# Codex Prompts

Reusable prompts live in `codex/prompts/`.

Historical one-off prompts live in `codex/archive/`, `codex/features/`, and `codex/refactors/`. Do not re-run historical prompts without checking them against the current project `README.md` and `AGENTS.md`.

## Current project constraints

- Go backend only.
- Private `/v1` API and public emergency viewer run on separate listener groups.
- Uploaded chunks are immutable.
- Media streams can be completed or failed.
- Completed streams/incidents can be downloaded as encrypted ZIP evidence bundles.
- Emergency viewer routes are read-only and token-scoped.
- No React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, public admin dashboard, Docker Compose, Kubernetes, or cloud integrations unless explicitly requested.
- Do not expose write/admin APIs publicly.
- Do not log raw tokens, request bodies, uploaded file bytes, or Authorization headers.
- Evidence bundles are encrypted chunk bundles, not decrypted or playable exports.

## Recommended prompt types

Use these reusable prompts first:

- `codex/prompts/code-review.md`
- `codex/prompts/security-review.md`
- `codex/prompts/documentation-update.md`
- `codex/prompts/readability-maintenance.md`
- `codex/prompts/mdn-web-security-header-review.md`
- `codex/prompts/simulator-maintenance.md`

Use historical prompts only as reference material.

## Validation

Before accepting Codex changes:

```bash
cd server
gofmt -w .
go test ./...
```

For larger changes:

```bash
go vet ./...
```
