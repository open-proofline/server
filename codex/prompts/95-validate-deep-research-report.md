Core rule: public issue drafts must include `## Priority`, `## Type`, `## Labels`,
and `## Branch scope`. Issue-creation scripts must fail closed when those sections
are missing and must pass labels to `gh issue create`.

# Codex Prompt Patch: 95 Validate Deep Research Report

Apply this patch to `codex/prompts/95-validate-deep-research-report.md`.

## Replace / strengthen the issue handling section

When `Issue handling mode` is `drafts_only`, write branch-scoped issue drafts under:

```text
.backlog-drafts/<YYYY-MM-DD>/<branch-slug>/
```

If the date is unavailable:

```text
.backlog-drafts/current/<branch-slug>/
```

Every public issue draft created from the report **must** use this structure and
section order:

```md
# <Issue title>

## Priority

P0 / P1 / P2 / P3

## Type

bug / maintenance / security-hardening / documentation / feature / deployment / ci / testing / planning

## Labels

Suggested GitHub labels:

- `backlog`
- `bug` / `maintenance` / `security` / `docs` / `deployment` / `testing` / `simulator` / `ios` / `ci` / `planning`

## Branch scope

- Current branch: `<CURRENT_BRANCH>`
- Current HEAD: `<CURRENT_HEAD>`
- Reviewed branch/ref: `<REVIEWED_BRANCH_OR_REF>`
- Reviewed commit SHA: `<REVIEWED_COMMIT_SHA>`
- Target release/version: `<TARGET_RELEASE_OR_VERSION>`
- Scope classification: `release-blocker-current-branch` / `follow-up-after-merge` / `revalidate-on-main-or-develop` / `planning-only` / `sensitive-do-not-publicly-file`
- Scope note: This draft was generated from a report against the branch above. Revalidate against the target branch before creating or closing public GitHub issues if the branch has moved or has not yet merged.

## Summary

One or two sentences.

## Context

Why this matters and what repo files/docs support it.

## Proposed change

What should change.

## Acceptance criteria

- [ ] ...

## Tests / validation

- [ ] ...

## Out of scope

What this issue must not include.

## Notes

- Report finding ID: `F-...`
```

## Priority and label requirements

- Do not create any public issue draft without `## Priority`.
- Do not create any public issue draft without `## Type`.
- Do not create any public issue draft without `## Labels`.
- Do not create any public issue draft without `## Branch scope`.
- Do not put priority only in prose. It must appear under `## Priority`.
- Do not put GitHub labels only in prose. They must appear under `## Labels`.
- Every public draft must include at least `- `backlog`` under `## Labels`.
- Include one or more type/topic labels that match the issue.
- Use likely existing labels only: `backlog`, `bug`, `maintenance`, `security`, `docs`,
  `deployment`, `testing`, `simulator`, `ios`, `ci`, `planning`.
- Do not invent new labels unless the maintainer explicitly asked for label creation.

## Create-issues mode

If `Issue handling mode` is `create_issues`, do not create an issue from any draft
unless it has `## Priority`, `## Type`, `## Labels`, and `## Branch scope`.

Preserve `## Priority`, `## Labels`, and `## Branch scope` in the public issue body.

Before creating issues, check repository labels:

```bash
gh label list --repo TheSilkky/safety-recorder --limit 200
```

Do not silently create unlabeled issues.
Do not silently drop missing labels.
Do not create labels unless explicitly requested.

## Validation after creating drafts

Run this check before finishing:

```bash
python3 - <<'PY'
from pathlib import Path
import sys

root = Path(".backlog-drafts")
if not root.exists():
    print("no backlog drafts")
    raise SystemExit(0)

required = ["## Priority", "## Type", "## Labels", "## Branch scope"]
bad = []
for path in root.rglob("*.md"):
    if path.name == "README.md" or "private-notes" in path.parts:
        continue
    text = path.read_text(encoding="utf-8")
    missing = [section for section in required if section not in text]
    if missing:
        bad.append((str(path), missing))
    if "## Labels" in text and "`backlog`" not in text:
        bad.append((str(path), ["missing `backlog` label"]))

for path, missing in bad:
    print(path, "missing:", ", ".join(missing))

if bad:
    sys.exit(1)
PY
```

## Output addition

In the final summary, include:

```text
- issue draft metadata validation:
- drafts excluded for missing priority/type/labels/branch scope:
- labels used:
- missing/non-existent labels:
```
