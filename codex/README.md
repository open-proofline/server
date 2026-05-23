# Codex Prompts

This directory records the Codex prompt workflow used for AI-assisted development.

Codex output is treated as maintainer-reviewed work, not as endorsement, audit, certification, security review, or maintenance by OpenAI.

## Directory Structure

Keep the Codex workflow in this structure:

```text
codex/
  README.md
  prompts/
  archive/
  features/
  refactors/
  work-orders/
```

Do not add extra prompt directories without a clear workflow reason. Generated
local review artifacts belong outside `codex/`.

## Prompt Categories

Reusable prompts live in `codex/prompts/`. They are scoped workflows that can be
run again against the current repository after reading current source-of-truth
docs.

Historical prompts live in:

```text
codex/archive/
codex/features/
codex/refactors/
codex/work-orders/
```

Historical prompts are reference material only. Do not re-run historical prompts
without checking them against the current `README.md`, `AGENTS.md`,
`SECURITY.md`, relevant docs, and reusable prompts.

## Naming Conventions

Reusable prompts use this filename pattern:

```text
NN-short-kebab-title.md
```

Rules:

- two-digit numeric prefix
- kebab-case title
- `.md` extension
- no spaces
- no date prefix
- one reusable workflow per file

Historical prompts use this filename pattern:

```text
YYYY-MM-DD-short-kebab-title.md
```

Rules:

- date prefix
- kebab-case title
- `.md` extension
- no numeric reusable-workflow prefix
- each file should be clearly marked historical/reference-only near the top

## Generated Artifacts

Generated local artifacts should not be placed under `codex/`.

Current generated artifact locations:

- `.backlog-drafts/YYYY-MM-DD/` or `.backlog-drafts/current/` for backlog issue drafts
- `.issue-review-drafts/YYYY-MM-DD/` or `.issue-review-drafts/current/` for open-issue review drafts
- `scripts/create-backlog-issues.sh` only when explicitly generated from reviewed backlog drafts

Backlog and issue-review drafts must not include raw tokens, secrets, private
deployment details, exploit details, or user safety data. Public GitHub issues
must not be created from drafts until the maintainer reviews them.

## Normal reusable prompt order

Use prompts in this rough order:

### Context and readiness

1. `00-project-context-check.md`
2. `05-codex-change-control.md`

### Maintenance, review, and design

3. `10-readability-maintenance.md`
4. `15-codex-structure-and-naming-maintenance.md`
5. `20-code-review.md`
6. `30-security-review.md`
7. `35-key-custody-and-emergency-access-design.md`
8. `36-update-codex-key-custody-guardrails.md`
9. `37-browser-decryption-design-spike.md`
10. `38-break-glass-and-dead-mans-switch-key-access-design.md`
11. `40-documentation-update.md`
12. `50-mdn-web-security-header-review.md`, for web-facing changes
13. `60-simulator-maintenance.md`, for API/client-flow changes

### Issue and PR workflow

14. `70-work-on-github-issue.md`
15. `75-create-draft-pr-from-current-branch.md`
16. `76-request-codex-pr-review.md`

### Backlog workflow

17. `80-backlog-scan-issue-drafts.md`
18. `81-backlog-drafts-structure-and-hygiene.md`
19. `82-review-open-issues-for-stale-or-fixed.md`
20. `85-create-github-issues-from-drafts.md`

### Release workflow

21. `90-release-check.md`
22. `95-validate-deep-research-report.md`, for Phase 2 validation of public technical review reports

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
- Key custody/decryption changes must be explicit security-sensitive work and update the threat model, security model, encryption docs, tests, and operational guidance before or alongside implementation.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.
- Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, public admin dashboards, Docker Compose, Kubernetes, or cloud integrations unless explicitly requested.
- Put newly discovered future work into issues/backlog items unless it is required for the current task.
- Backlog scanning creates draft Markdown files first, not GitHub issues directly.
- Do not create public GitHub issues from backlog drafts until the maintainer has reviewed them.
- Never put raw tokens, secrets, private deployment details, exploit details, or user safety data into public issue drafts.

## Issue And PR Workflow

Use `70-work-on-github-issue.md` for scoped implementation work tied to one
GitHub issue.

Use `75-create-draft-pr-from-current-branch.md` when a reviewed local branch
should become a draft pull request.

Use `76-request-codex-pr-review.md` for a code-review pass over an existing
pull request.

## Backlog And Issue Review Workflow

Use `80-backlog-scan-issue-drafts.md` to generate timestamped backlog drafts
under `.backlog-drafts/`.

Use `81-backlog-drafts-structure-and-hygiene.md` to review or clean up backlog
draft structure. It should not create or close GitHub issues.

Use `82-review-open-issues-for-stale-or-fixed.md` to create local issue review
drafts under `.issue-review-drafts/`. It should not close GitHub issues unless
the maintainer explicitly asks for that follow-up action.

Only after manual review, use `85-create-github-issues-from-drafts.md` to
generate a script for GitHub issue creation. Do not execute that script unless
explicitly instructed.

## Key custody prompt use

Use `35-key-custody-and-emergency-access-design.md` when making the next encryption/key architecture decision.

Use `36-update-codex-key-custody-guardrails.md` when updating prompt wording, docs, or `AGENTS.md` so that "no backend decryption/no server keys" does not become a permanent absolute rule.

Use `37-browser-decryption-design-spike.md` for browser-side emergency viewer decryption design.

Use `38-break-glass-and-dead-mans-switch-key-access-design.md` for server escrow, dead-man-switch, or break-glass key access design.

## Technical review report workflow

Use `docs/reports/prompts/phase-1-deep-research-technical-review.md` outside
Codex to draft a source-cited public technical review report.

Use `95-validate-deep-research-report.md` in Codex to verify repository claims,
remove draft-only material, pin repository citations, check public-safety
constraints, and produce a cleaned report under `docs/reports/`.

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
