# Codex Prompt: Backlog Drafts Structure, Branch Scope, and Hygiene

Review `.backlog-drafts/` structure and backlog draft workflow hygiene.

This is a documentation/process maintenance task.

Do **not** change application code.
Do **not** create GitHub issues.
Do **not** close GitHub issues.
Do **not** execute issue creation scripts.
Do **not** delete files unless explicitly requested.

## Goal

Ensure backlog drafts are clearly treated as generated/reviewable local artifacts, not as the source of truth once GitHub issues exist.

Standardize `.backlog-drafts/` structure, naming, and branch-scoping.

## Standard `.backlog-drafts/` structure

Preferred generated output for branch-scoped drafts:

```text
.backlog-drafts/
  YYYY-MM-DD/
    <branch-slug>/
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
.backlog-drafts/current/<branch-slug>/
```

## Branch scope metadata

Every public issue draft should include:

```md
## Branch scope

- Current branch: `<CURRENT_BRANCH>`
- Current HEAD: `<CURRENT_HEAD>`
- Target branch, if known: `<TARGET_BRANCH_OR_UNKNOWN>`
- Reviewed branch/ref, if applicable: `<REVIEWED_BRANCH_OR_REF>`
- Reviewed commit SHA, if applicable: `<REVIEWED_COMMIT_SHA>`
- Target release/version, if applicable: `<TARGET_RELEASE_OR_VERSION>`
- Scope classification: `release-blocker-current-branch` / `follow-up-after-merge` / `revalidate-on-main-or-develop` / `planning-only`
- Scope note: This draft was generated from the branch above. Revalidate against the target branch before creating or closing public GitHub issues if the branch has moved or has not yet merged.
```

## Audit checks

Check whether `.backlog-drafts/` exists, whether new drafts are branch-scoped under `<branch-slug>/`, whether drafts include `## Branch scope`, and whether issue creation scripts select branch-scoped directories.

## Validation

After changes, if any:

```bash
git diff --stat
git diff -- .gitignore .backlog-drafts codex docs scripts
```

If only Markdown and `.gitignore` changed, Go tests are not required.
