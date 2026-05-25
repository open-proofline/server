Core rule: public issue drafts must include `## Priority`, `## Type`, `## Labels`,
and `## Branch scope`. Issue-creation scripts must fail closed when those sections
are missing and must pass labels to `gh issue create`.

# Codex Prompt Patch: 81 Backlog Drafts Structure and Hygiene

Apply this patch to `codex/prompts/81-backlog-drafts-structure-and-hygiene.md`.

## Required public issue draft metadata

Every public issue draft must include:

```md
## Priority

P0 / P1 / P2 / P3

## Type

bug / maintenance / security-hardening / documentation / feature / deployment / ci / testing / planning

## Labels

Suggested GitHub labels:

- `backlog`
- one or more topic/type labels

## Branch scope

...
```

Every public issue draft must include `backlog` under `## Labels`.

Private notes do not need GitHub labels, but must never be used for public issue creation.

## Audit checks

Check whether public issue drafts include:

- `## Priority`
- `## Type`
- `## Labels`
- `## Branch scope`
- backtick-wrapped `backlog` label
- at least one additional topic/type label

Also check whether issue creation scripts:

- parse labels from `## Labels`
- pass labels to `gh issue create`
- exclude drafts when labels are missing
- exclude drafts when referenced labels do not exist
- refuse to silently create unlabeled issues

## Validation

Run this metadata check when auditing drafts:

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
