# Codex Prompt: Backlog Drafts Structure and Hygiene

Review `.backlog-drafts/` structure and backlog draft workflow hygiene.

This is a documentation/process maintenance task.

Do **not** change application code.
Do **not** create GitHub issues.
Do **not** close GitHub issues.
Do **not** execute issue creation scripts.
Do **not** delete files unless explicitly requested.
Do **not** modify workflows, Dockerfiles, SQL, Go code, generated binaries, local DBs, or uploaded blob data.

## Goal

Ensure backlog drafts are clearly treated as generated/reviewable local artifacts, not as the source of truth once GitHub issues exist.

Standardize `.backlog-drafts/` structure and naming.

Check whether reusable prompts enforce that structure.

## Source of truth

Read:

- `README.md`
- `AGENTS.md`
- `docs/codex-change-control.md`, if present
- `codex/README.md`
- `codex/prompts/80-backlog-scan-issue-drafts.md`
- `codex/prompts/85-create-github-issues-from-drafts.md`
- `codex/prompts/82-review-open-issues-for-stale-or-fixed.md`, if present
- `.backlog-drafts/`, if present
- `scripts/create-backlog-issues.sh`, if present
- `.gitignore`

## Standard `.backlog-drafts/` structure

Preferred generated output:

```text
.backlog-drafts/
  YYYY-MM-DD/
    README.md
    001-short-kebab-title.md
    002-short-kebab-title.md
    create-issues-review.md
    private-notes/
      README.md
      001-private-note.md
```

Fallback if date is unavailable:

```text
.backlog-drafts/current/
```

## Naming rules

Issue drafts:

```text
NNN-short-kebab-title.md
```

Rules:

- three-digit numeric prefix
- kebab-case title
- `.md` extension
- no spaces
- no root-level issue draft files

Index file:

```text
README.md
```

Issue creation review:

```text
create-issues-review.md
```

Private notes:

```text
private-notes/NNN-short-kebab-title.md
```

Private notes must never be used for public issue creation.

## Commit policy

Recommended default for public repositories:

- `.backlog-drafts/` should normally be ignored and not committed.
- Actual backlog source of truth should be GitHub Issues.
- Historical drafts can be moved into a clearly marked archive only if useful.

If `.backlog-drafts/` is already committed, recommend one of:

1. remove from repository and add `.backlog-drafts/` to `.gitignore`
2. move to `codex/archive/YYYY-MM-DD-initial-backlog-drafts/` with a README explaining it is historical
3. keep only if there is a clear reason and no stale/duplicate issue drafts

## Audit checks

Check:

- whether `.backlog-drafts/` exists
- whether it uses flat root-level drafts
- whether it uses timestamped directories
- whether `create-issues-review.md` is in the selected scan directory
- whether `private-notes/` exists and is excluded from issue creation
- whether `.backlog-drafts/` is ignored in `.gitignore`
- whether issue drafts duplicate already-created GitHub issues
- whether any issue creation script points to stale flat paths
- whether `80-backlog-scan-issue-drafts.md` enforces timestamped output
- whether `85-create-github-issues-from-drafts.md` selects a timestamped draft directory
- whether any draft contains raw tokens, secrets, private deployment info, exploit details, or user safety data

## Permitted changes

Only make changes if the maintainer asked for implementation, not just review.

Allowed when implementation is requested:

- update `codex/prompts/80-backlog-scan-issue-drafts.md`
- update `codex/prompts/85-create-github-issues-from-drafts.md`
- update `codex/README.md`
- update `docs/codex-change-control.md`
- update `.gitignore`
- move `.backlog-drafts/` into a historical archive
- remove stale issue creation scripts only if explicitly requested

Not allowed:

- application code changes
- creating issues
- closing issues
- executing scripts
- deleting drafts without explicit instruction

## Public repository recommendation

Before making the repo public, prefer:

```text
.gitignore includes .backlog-drafts/
committed .backlog-drafts/ is removed or archived
scripts/create-backlog-issues.sh is removed, archived, or updated to timestamped draft paths
```

## Validation

After changes, if any:

```bash
git diff --stat
git diff -- .gitignore .backlog-drafts codex docs scripts
```

If only Markdown and `.gitignore` changed, Go tests are not required.

If application code changed, stop and explain why.

## Output

Summarize:

1. current `.backlog-drafts/` structure
2. whether it matches the standard
3. stale/duplicate drafts found
4. script/path mismatches found
5. `.gitignore` recommendation
6. prompt workflow changes recommended
7. public-repo cleanup recommendation
8. any files changed, if implementation was requested
