# Codex Prompt: Backlog Scan and Issue Drafts

Scan the repository and create reviewed backlog issue drafts.

This prompt is reusable. It must discover the current repo state each time it runs rather than relying on stale hard-coded candidate issues.

Do **not** change application code.
Do **not** change application behaviour.
Do **not** create GitHub issues directly in this task.
Do **not** run `gh issue create`.
Do **not** modify workflows, Dockerfiles, SQL migrations, Go code, generated files, binaries, database files, or uploaded blob data.
Do **not** add features.

## Goal

Review the current repository, current documentation, current issue backlog, and recent merged work.

Produce a small set of high-quality backlog issue drafts for future work.

The drafts should be specific enough that the maintainer can review them and later create GitHub issues manually or with GitHub CLI.

## Repository

```text
TheSilkky/safety-recorder
```

## Project context

Safety Recorder is an experimental Go backend for a private personal-safety recording system.

The project may change over time. Do not assume the version, feature set, or backlog state from this prompt alone.

Start by reading the current repo source of truth.

## Source of truth to inspect

Read current repository files where present:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- `LICENSE`
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
- `docs/codex-change-control.md`, if present
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

Also inspect current GitHub issues and PRs if GitHub CLI is available:

```bash
gh issue list --repo TheSilkky/safety-recorder --state all --limit 100
gh pr list --repo TheSilkky/safety-recorder --state all --limit 50
```

If GitHub CLI is unavailable, continue using local repo files and mention that issue/PR duplicate detection was limited.

## Existing issue duplicate check

Before drafting any issue:

1. Check whether an existing open issue already covers it.
2. Check whether a closed issue or merged PR recently completed it.
3. If an existing issue covers it, do not create a duplicate draft.
4. If an existing issue is close but incomplete, create a draft suggesting an update/comment instead of a duplicate issue.
5. If uncertain, include a note in `.backlog-drafts/README.md` rather than generating a duplicate issue.

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
12. Codex workflow/process improvements

## Candidate discovery guidance

Do not blindly recreate the same backlog every run.

Derive candidates from current code/docs/issues.

Good candidate signals:

- documented known gaps
- TODO/FIXME comments
- missing validation compared with docs
- docs saying “future work”
- security model gaps
- threat-model mitigations not implemented
- code paths with tests missing
- deployment warnings without examples
- workflows mentioned in docs but not automated
- simulator gaps against intended client flow
- open issues that need splitting/refinement
- recently merged work that introduced follow-up tasks

Bad candidate signals:

- vague “improve code”
- duplicate of existing issue
- feature that contradicts README/AGENTS scope
- production claims beyond current maturity
- public issue containing sensitive vulnerability details
- anything requiring secrets, raw tokens, or private deployment details

## Sensitive findings

If you find a likely security vulnerability that should not be public:

- Do not create a normal public issue draft for it.
- Create a private note under:

```text
.backlog-drafts/private-notes/
```

- Clearly mark it:

```text
PRIVATE SECURITY NOTE - DO NOT CREATE PUBLIC ISSUE
```

- Do not include raw tokens, secrets, user safety data, private deployment details, or exploit payloads.
- Refer to `SECURITY.md` for reporting/handling.

## Output directory

Create a timestamped draft directory so repeated scans do not overwrite previous scans:

```text
.backlog-drafts/YYYY-MM-DD/
```

If the date is not easily available, use:

```text
.backlog-drafts/current/
```

Inside it, create one Markdown file per proposed issue.

Filename format:

```text
NNN-short-kebab-title.md
```

Example:

```text
001-add-default-token-expiry.md
```

Also create:

```text
.backlog-drafts/YYYY-MM-DD/README.md
```

The index should list all drafted issues grouped by priority/category and include any skipped duplicates.

## Number of issues

Prefer quality over volume.

Default target:

```text
5 to 12 high-quality issue drafts
```

If fewer than 5 good issues exist, create fewer and say why.

If more than 12 exist, include only the highest-value drafts and list lower-priority candidates in the index as “future scan candidates.”

## Issue draft format

Each issue draft must use this structure:

```md
# <Issue title>

## Priority

P0 / P1 / P2 / P3

## Type

bug / maintenance / security-hardening / documentation / feature / deployment / ci / testing / planning

## Labels

Suggested labels:

- `backlog`
- one or more of: `bug`, `maintenance`, `security`, `docs`, `deployment`, `testing`, `simulator`, `ios`, `ci`, `planning`

## Summary

One or two sentences.

## Context

Why this matters and what repo files/docs support it.

Mention existing related issues or PRs if relevant.

## Proposed change

What should change.

## Acceptance criteria

- [ ] ...
- [ ] ...
- [ ] ...

## Tests / validation

- [ ] `cd server && go test ./...`, if code changes
- [ ] `cd server && go vet ./...`, if code changes or CI/testing changes
- [ ] simulator smoke test, if relevant
- [ ] docs updated, if relevant

## Out of scope

What this issue must not include.

## Notes

Any references to files, docs, related issues, related PRs, or future work.
```

## Priority guide

Use:

```text
P0 = urgent correctness/security issue that should be handled before further feature work
P1 = important before real-world/private deployment or before iOS work
P2 = useful near-term improvement
P3 = polish, documentation, cleanup, or governance
```

Do not overuse P0.

## Label guide

Suggest labels only. Do not create labels in this task.

Recommended labels:

- `backlog`
- `bug`
- `maintenance`
- `security`
- `docs`
- `deployment`
- `testing`
- `simulator`
- `ios`
- `ci`
- `planning`

If a label does not exist, still suggest it if useful, but mention missing/new labels in the index.

## Requirements

- Keep issues specific and actionable.
- Do not create huge umbrella issues unless they are clearly planning/docs issues.
- Do not include secrets, raw tokens, private deployment info, or user safety data.
- Do not include exploit details in public issue drafts.
- Do not include issue drafts for work already completed by current code.
- Do not include issue drafts already covered by open issues.
- Do not add jokes or informal commentary to issue drafts.
- Do not claim the project is production-ready.
- Keep all output as Markdown.

## Validation

After creating drafts:

```bash
git diff --stat
git diff -- .backlog-drafts
```

Do not run Go tests unless code was changed.

If any files outside `.backlog-drafts/` changed, stop and explain why.

## Output

Summarize:

1. draft directory created
2. number of issue drafts created
3. priority breakdown
4. categories covered
5. existing issues/PRs checked
6. duplicates skipped
7. issues that should be reviewed before public creation
8. sensitive/private notes created, if any
9. suggested next command for creating GitHub issues manually
