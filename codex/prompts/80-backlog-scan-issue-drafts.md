Core rule: public issue drafts must include `## Priority`, `## Type`, `## Labels`,
and `## Branch scope`. Issue-creation scripts must fail closed when those sections
are missing and must pass labels to `gh issue create`.

# Codex Prompt Patch: 80 Backlog Scan and Issue Drafts

Apply this patch to `codex/prompts/80-backlog-scan-issue-drafts.md`.

## Required issue draft structure

Every issue draft must use this structure and section order:

```md
# <Issue title>

## Priority

P0 / P1 / P2 / P3

## Type

bug / maintenance / security-hardening / documentation / feature / deployment / ci / testing / planning

## Labels

Suggested GitHub labels:

- `backlog`
- one or more of: `bug`, `maintenance`, `security`, `docs`, `deployment`, `testing`, `simulator`, `ios`, `ci`, `planning`

## Branch scope

- Current branch: `<CURRENT_BRANCH>`
- Current HEAD: `<CURRENT_HEAD>`
- Target branch, if known: `<TARGET_BRANCH_OR_UNKNOWN>`
- Scope classification: `release-blocker-current-branch` / `follow-up-after-merge` / `revalidate-on-main-or-develop` / `planning-only`
- Scope note: This draft was generated from the branch above. Revalidate against the target branch before creating or closing public GitHub issues if the branch has moved or has not yet merged.

## Summary

One or two sentences.

## Context

Why this matters and what repo files/docs support it.

## Proposed change

What should change.

## Acceptance criteria

- [ ] ...

## Tests / validation

- [ ] `cd server && go test ./...`, if code changes
- [ ] `cd server && go vet ./...`, if code changes or CI/testing changes
- [ ] simulator smoke test, if relevant
- [ ] docs updated, if relevant
- [ ] revalidate on target branch before public issue creation, if branch-scoped

## Out of scope

What this issue must not include.

## Notes

Any references to files, docs, related issues, related PRs, branch scope, or future work.
```

## Requirements

- Do not create any public issue draft without `## Priority`, `## Type`, `## Labels`, and `## Branch scope`.
- Every draft must include `backlog` under `## Labels`.
- Every draft must include at least one additional topic/type label.
- Keep labels as backtick-wrapped Markdown bullets.
- Do not invent new labels unless the maintainer explicitly requested label creation.
- If no suitable label exists, use the closest existing label and note the uncertainty under `## Notes`.

## Validation

After creating drafts, run:

```bash
python3 - <<'PY'
from pathlib import Path
import sys

bad = []
required = ["## Priority", "## Type", "## Labels", "## Branch scope"]
for path in Path(".backlog-drafts").rglob("*.md"):
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
