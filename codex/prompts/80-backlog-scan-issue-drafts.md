# Codex Prompt: Backlog Scan and Branch-Scoped Issue Drafts

Scan the repository and create reviewed backlog issue drafts.

This prompt is reusable. It must discover the current repo state each time it runs rather than relying on stale hard-coded candidate issues.

Do **not** change application code.
Do **not** change application behaviour.
Do **not** create GitHub issues directly in this task.
Do **not** run `gh issue create`.
Do **not** modify workflows, Dockerfiles, SQL migrations, Go code, generated files, binaries, database files, or uploaded blob data.
Do **not** add features.

## Goal

Review the current repository, current documentation, current issue backlog, current branch, and recent merged work.

Produce a small set of high-quality branch-scoped backlog issue drafts for future work.

## Branch scope

Before scanning, record:

```bash
git status --short --branch --untracked-files=all
git branch --show-current
git rev-parse HEAD
```

Use the current checked-out branch as the draft issue scope. Create a filesystem-safe branch slug by replacing `/` and other non-alphanumeric separators with `-`.

Create drafts under:

```text
.backlog-drafts/YYYY-MM-DD/<branch-slug>/
```

or:

```text
.backlog-drafts/current/<branch-slug>/
```

Every issue draft must include:

```md
## Branch scope

- Current branch: `<CURRENT_BRANCH>`
- Current HEAD: `<CURRENT_HEAD>`
- Target branch, if known: `<TARGET_BRANCH_OR_UNKNOWN>`
- Scope classification: `release-blocker-current-branch` / `follow-up-after-merge` / `revalidate-on-main-or-develop` / `planning-only`
- Scope note: This draft was generated from the branch above. Revalidate against the target branch before creating or closing public GitHub issues if the branch has moved or has not yet merged.
```

## Source of truth to inspect

Read current repository files where present, including `README.md`, `AGENTS.md`, `CHANGELOG.md`, `SECURITY.md`, `docs/`, `server/`, `.github/`, and `codex/prompts`.

Also inspect current GitHub issues and PRs if GitHub CLI is available.

## Requirements

- Keep issues specific and actionable.
- Do not include secrets, raw tokens, private deployment info, exploit details, or user safety data.
- Do not include issue drafts already covered by open issues.
- Do not claim the project is production-ready.
- Include branch scope in every issue draft.
- Keep all output as Markdown.

## Validation

After creating drafts:

```bash
git diff --stat
git diff -- .backlog-drafts
```

Check branch scope:

```bash
python3 - <<'PY'
from pathlib import Path
import sys

bad = []
for path in Path(".backlog-drafts").rglob("*.md"):
    if path.name == "README.md" or "private-notes" in path.parts:
        continue
    text = path.read_text(encoding="utf-8")
    if "## Branch scope" not in text:
        bad.append(str(path))

print("drafts missing branch scope:", bad)
if bad:
    sys.exit(1)
PY
```

Do not run Go tests unless code was changed.

If any files outside `.backlog-drafts/` changed, stop and explain why.
