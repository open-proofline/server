# Codex Prompt: Readability Maintenance

Review the Go backend for readability and maintainability.

Do **not** add features.
Do **not** change endpoint behaviour.
Do **not** add a web framework unless explicitly requested.
Do **not** reformat unrelated documentation or prompt files.

## Goal

Make the code easier for a human to understand and debug while preserving behaviour.

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
- Any key custody/decryption change must be an explicit security-sensitive task that updates the threat model, security model, encryption docs, tests, and operational guidance before or alongside implementation.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.

## Focus on

- splitting overly large files
- clearer handler names
- clearer route registration
- clearer package responsibilities
- reducing duplicated request/response helpers
- comments around non-obvious logic
- private API and emergency viewer separation
- upload/storage/hash-verification readability
- stream/bundle logic readability
- encryption-envelope and simulator-key handling readability
- simulator readability
- tests that clearly describe behaviour

## Do not

- change public JSON field names
- change database schema unless required for a bug
- change token/security model
- change encryption envelope format
- change key custody/decryption behaviour
- change bundle format
- add new dependencies unless the maintainer explicitly requests them

## Validation

After changes:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

## Output

Summarize:

1. files changed
2. readability changes made
3. behaviour-preservation notes
4. validation commands run
5. any follow-up work that should become an issue
