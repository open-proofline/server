# Codex Prompts

## Reusable prompts

Reusable prompts live in `codex/prompts/`.

This directory records the Codex prompt workflow used for AI-assisted development. Codex output is treated as maintainer-reviewed work, not as endorsement, audit, certification, or maintenance by OpenAI.

Use reusable prompts in this rough order:

1. `00-project-context-check.md`
2. `05-codex-change-control.md`
3. `10-readability-maintenance.md`
4. `20-code-review.md`
5. `30-security-review.md`
6. `40-documentation-update.md`
7. `50-mdn-web-security-header-review.md`, for web-facing changes
8. `60-simulator-maintenance.md`, for API/client-flow changes
9. `90-release-check.md`, before tagging

Historical prompts are reference material only and are not part of the normal flow.

Historical one-off prompts live in `codex/archive/`, `codex/features/`, `codex/refactors/` and `codex/work-orders/`. Do not re-run historical prompts without checking them against the current project `README.md` and `AGENTS.md`.

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
- Start larger Codex tasks from a clean working tree or explicit checkpoint commit.
- Put newly discovered future work into issues/backlog items unless it is required for the current task.

## Recommended prompt types

Use these reusable prompts first:

- `codex/prompts/00-project-context-check.md`
- `codex/prompts/05-codex-change-control.md`
- `codex/prompts/10-readability-maintenance.md`
- `codex/prompts/20-code-review.md`
- `codex/prompts/30-security-review.md`
- `codex/prompts/40-documentation-update.md`
- `codex/prompts/50-mdn-web-security-header-review.md`
- `codex/prompts/60-simulator-maintenance.md`

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
