# Codex Prompt: Work on GitHub Issue

Work on one GitHub issue in this repository.

## Inputs

Issue number: `<ISSUE_NUMBER>`

Repository:

```text
TheSilkky/safety-recorder
```

## Rules

- Use the current checked-out branch.
- Do not create or checkout another branch.
- Do not create a pull request unless explicitly requested.
- Keep the change scoped to the issue.
- Do not add unrelated features.
- Do not change public API behaviour unless the issue requires it.
- Do not weaken security warnings.
- Do not expose `/v1` publicly.
- Preserve private/public listener separation.
- If the issue appears security-sensitive, stop and state whether public issue handling is appropriate before making changes.

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

## First steps

Read the issue:

```bash
gh issue view <ISSUE_NUMBER> --repo TheSilkky/safety-recorder
```

Then read:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- relevant files in `docs/`
- relevant source files
- relevant tests

Before changing files, summarize:

1. issue goal
2. acceptance criteria from the issue
3. files likely affected
4. validation commands
5. out-of-scope items
6. whether this should be docs-only, code-only, or mixed
7. whether the issue is safe for normal public issue handling
8. whether the issue changes key custody/decryption assumptions

## Implementation

Make the smallest useful change that satisfies the issue.

Keep the diff reviewable.

If you discover unrelated future work, do not implement it. Suggest a backlog issue instead.

## Validation

If Go code changed:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

If only Markdown changed, inspect docs and links manually. Go tests are not required unless code changed.

If behaviour changed and the simulator is relevant:

```bash
cd server
go run ./cmd/api
```

In another terminal:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

## Output

Summarize:

1. files changed
2. implementation summary
3. issue acceptance criteria satisfied
4. validation commands run
5. any follow-up work
6. whether a PR should be opened
