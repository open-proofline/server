# Codex Prompt: Prepare GitHub Issue Creation Commands from Reviewed Branch-Scoped Backlog Drafts

Read reviewed backlog drafts and generate a script that can create GitHub issues with GitHub CLI.

Do **not** modify application code.
Do **not** create GitHub issues directly unless explicitly told to run commands.
Do **not** execute the generated script.
Do **not** include sensitive vulnerability details in public issues.

## Goal

Generate a shell script that creates GitHub issues from maintainer-reviewed backlog draft files.

Create:

```text
scripts/create-backlog-issues.sh
```

## Draft directory

Backlog drafts should live under a branch-scoped timestamped directory such as:

```text
.backlog-drafts/YYYY-MM-DD/<branch-slug>/
```

or:

```text
.backlog-drafts/current/<branch-slug>/
```

Before generating the script:

1. Prefer an explicitly provided draft directory.
2. Otherwise, inspect `.backlog-drafts/` and choose the newest timestamped branch-scoped directory.
3. If multiple directories exist and the newest or intended branch scope is unclear, stop and ask which draft directory to use.
4. Do not use `private-notes/` directories for public issue creation.

## Branch scope requirements

Verify every included draft contains:

```md
## Branch scope
```

If a draft is missing branch scope, exclude it and list it in the output summary.

If a draft was generated from a release-prep or feature branch, preserve the `Branch scope` section in the GitHub issue body unless the maintainer explicitly requests removal.

If a draft says it must be revalidated on `main`, `develop`, or another target branch before public issue creation, exclude it unless the maintainer explicitly confirms revalidation is complete.

## Requirements

- Use `gh issue create`.
- Derive title from the first Markdown heading.
- Use the entire draft body, including `## Branch scope`.
- Use labels from the `## Labels` section where practical.
- Do not execute the script.
- Quote file paths and arguments safely.

## Validation

After generating the script:

```bash
git diff --stat
git diff -- scripts/create-backlog-issues.sh .backlog-drafts
```

Do not run Go tests unless code was changed.
