# Codex Prompt: Scan Repository and Draft Backlog Issues

Scan the repository and create reviewed backlog issue drafts.

Do **not** change application code.
Do **not** change application behaviour.
Do **not** create GitHub issues directly in this task.
Do **not** run `gh issue create`.
Do **not** modify workflows, Dockerfiles, SQL migrations, Go code, generated files, binaries, database files, or uploaded blob data.
Do **not** add features.

## Goal

Review the current repository and produce a set of backlog issue drafts for future work.

The drafts should be high quality enough that the maintainer can review them and later create GitHub issues manually or with GitHub CLI.

## Project context

Safety Recorder is an experimental Go backend for a private personal-safety recording system.

Current project shape:

- Go backend only
- private `/v1` write/admin API listener group
- public read-only emergency viewer listener group
- SQLite metadata
- local disk encrypted chunk storage
- immutable chunk uploads
- media streams that can be marked `open`, `complete`, or `failed`
- completed encrypted stream and incident ZIP evidence bundle downloads
- emergency viewer tokens
- simulator CLI
- documented v1 AES-256-GCM simulator encryption envelope
- Docker image build
- GitHub Actions / GHCR publishing
- AGPL-3.0-only license
- repository security policy
- Codex prompt library under `codex/`

Evidence bundles are encrypted chunk bundles, not decrypted/playable media exports.

The project is experimental and not production-ready public infrastructure.

## Source of truth

Read:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- `docs/README.md`
- `docs/api.md`
- `docs/architecture.md`
- `docs/configuration.md`
- `docs/deployment.md`
- `docs/encryption.md`
- `docs/security-model.md`
- `docs/threat-model.md`
- `docs/code-map.md`
- `docs/development.md`
- `server/internal/config`
- `server/internal/db`
- `server/internal/httpapi`
- `server/internal/incidents`
- `server/internal/storage`
- `server/internal/envelope`
- `server/cmd/api`
- `server/cmd/simclient`
- `.github/workflows`
- `.github/ISSUE_TEMPLATE`, if present
- `codex/prompts`

## Areas to scan

Look for future work in these categories:

1. Correctness
2. Security hardening
3. Deployment hardening
4. Database/migration maturity
5. Configuration
6. Testing gaps
7. Documentation gaps
8. Simulator/dev tooling
9. iOS-client prerequisites
10. Operational readiness
11. Release/CI polish

## Known candidate issues

Include these if they are still relevant:

- Fix streamed chunk-index semantics so streamed chunks require `chunk_index >= 1`.
- Add explicit `schema_migrations` tracking.
- Add configurable private/public HTTP server timeouts.
- Add default emergency-token expiry policy.
- Add reverse proxy / WireGuard deployment examples.
- Add rate-limiting guidance or reverse-proxy examples.
- Add branch protection / required CI documentation.
- Add emergency viewer DOM updates if polling currently does not update visible data.
- Add retention / backup / secure deletion policy.
- Add `go vet ./...` to CI.
- Add production key-sharing/key-access design document before iOS implementation.
- Add iOS local recorder prototype planning issue.

Do not invent issues that contradict current documented scope.

## Output directory

Create:

```text
.backlog-drafts/
```

Inside it, create one Markdown file per proposed issue.

Filename format:

```text
NNN-short-kebab-title.md
```

Example:

```text
001-fix-streamed-chunk-index-semantics.md
```

Also create:

```text
.backlog-drafts/README.md
```

This index should list all drafted issues grouped by priority/category.

## Issue draft format

Each issue draft must have this structure:

```md
# <Issue title>

## Priority

P0 / P1 / P2 / P3

## Type

bug / maintenance / security-hardening / documentation / feature / deployment

## Labels

Suggested labels:

- `backlog`
- one or more of: `bug`, `maintenance`, `security`, `docs`, `deployment`, `testing`, `simulator`, `ios`, `ci`

## Summary

One or two sentences.

## Context

Why this matters and what repo files/docs support it.

## Proposed change

What should change.

## Acceptance criteria

- [ ] ...
- [ ] ...
- [ ] ...

## Tests / validation

- [ ] `cd server && go test ./...`
- [ ] `cd server && go vet ./...`, if relevant
- [ ] simulator smoke test, if relevant
- [ ] docs updated, if relevant

## Out of scope

What this issue must not include.

## Notes

Any references to files, docs, or related future work.
```

## Requirements

- Keep issues specific and actionable.
- Do not create huge umbrella issues unless they are clearly planning/docs issues.
- Prefer 5 to 12 high-quality issues over 40 vague ones.
- Do not include secrets, raw tokens, private deployment info, or user safety data.
- Do not include exploit details in public issue drafts.
- Security vulnerabilities should point to `SECURITY.md` instead of becoming public issue drafts.
- If something is a sensitive vulnerability, create a private note draft instead of a public issue draft and mark it clearly.

## Validation

After creating drafts:

```bash
git diff --stat
git diff -- .backlog-drafts
```

Do not run Go tests unless code was changed.

## Output

Summarize:

1. number of issue drafts created
2. priority breakdown
3. categories covered
4. issues that should be reviewed before public creation
5. issues that may be sensitive and should not be public
6. suggested next command for creating GitHub issues manually
