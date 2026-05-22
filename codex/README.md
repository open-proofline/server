# Codex Prompts

This directory records the Codex prompt workflow used for AI-assisted development.

Codex output is treated as maintainer-reviewed work, not as endorsement, audit, certification, security review, or maintenance by OpenAI.

## Prompt categories

Reusable prompts live in `codex/prompts/`.

Historical one-off prompts live in:

```text
codex/archive/
codex/features/
codex/refactors/
codex/work-orders/
```

Historical prompts are reference material only. Do not re-run historical prompts without checking them against the current `README.md`, `AGENTS.md`, `SECURITY.md`, and relevant docs.

## Normal reusable prompt order

Use prompts in this rough order:

### Context and readiness

1. `00-project-context-check.md`
2. `05-codex-change-control.md`

### Maintenance, review, and design

3. `10-readability-maintenance.md`
4. `20-code-review.md`
5. `30-security-review.md`
6. `35-key-custody-and-emergency-access-design.md`
7. `36-update-codex-key-custody-guardrails.md`
8. `37-browser-decryption-design-spike.md`
9. `38-break-glass-and-dead-mans-switch-key-access-design.md`
10. `40-documentation-update.md`
11. `50-mdn-web-security-header-review.md`, for web-facing changes
12. `60-simulator-maintenance.md`, for API/client-flow changes

### Issue and PR workflow

13. `70-work-on-github-issue.md`
14. `75-create-draft-pr-from-current-branch.md`
15. `76-request-codex-pr-review.md`

### Backlog workflow

16. `80-backlog-scan-issue-drafts.md`
17. `85-create-github-issues-from-drafts.md`

### Release workflow

18. `90-release-check.md`

## Current project constraints

Treat `README.md`, `AGENTS.md`, `SECURITY.md`, and the `docs/` directory as the current source of truth.

Core constraints:

- Keep the backend small, boring, and testable.
- Prefer Go standard library where practical.
- Do not expose private `/v1` write/admin APIs publicly.
- Keep private `/v1` routes and public emergency viewer routes on separate listener groups and muxes.
- Treat uploaded chunks as immutable.
- Evidence bundles are encrypted chunk bundles, not decrypted/playable media exports.
- Do not log raw emergency tokens, request bodies, uploaded bytes, Authorization headers, plaintext, raw keys, or future token-like values.
- Use stable, documented crypto libraries only. Do not implement cryptographic primitives.
- Preserve the current backend ciphertext-only implementation unless a task explicitly concerns key custody, emergency access, or decryption design.
- Do not introduce backend decryption, raw server-held decryption keys, key escrow, browser decryption, or key-sharing behaviour incidentally.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.
- Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, public admin dashboards, Docker Compose, Kubernetes, or cloud integrations unless explicitly requested.
- Put newly discovered future work into issues/backlog items unless it is required for the current task.
- Backlog scanning creates draft Markdown files first, not GitHub issues directly.
- Do not create public GitHub issues from backlog drafts until the maintainer has reviewed them.
- Never put raw tokens, secrets, private deployment details, exploit details, or user safety data into public issue drafts.

## Key custody prompt use

Use `35-key-custody-and-emergency-access-design.md` when making the next encryption/key architecture decision.

Use `36-update-codex-key-custody-guardrails.md` when updating prompt wording, docs, or `AGENTS.md` so that "no backend decryption/no server keys" does not become a permanent absolute rule.

Use `37-browser-decryption-design-spike.md` for browser-side emergency viewer decryption design.

Use `38-break-glass-and-dead-mans-switch-key-access-design.md` for server escrow, dead-man-switch, or break-glass key access design.

## Validation

Before accepting Codex changes that touch Go code:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

For docs-only changes, inspect the relevant Markdown and links manually. Go tests are not required unless code changed.

For simulator/API flow changes, also run the simulator smoke test when practical:

```bash
cd server
go run ./cmd/api
```

In another terminal:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```
