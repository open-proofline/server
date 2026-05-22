# Codex Prompt Review Summary

These markdown files are an updated prompt/documentation pack aligned to the current Safety Recorder Backend v0.2.1 README and AGENTS project rules.

## What changed

- Added a stronger `AGENTS.md`.
- Added `codex/README.md`.
- Fixed reusable prompt markdown code fences.
- Updated reusable review prompts for:
  - media streams
  - encrypted ZIP evidence bundles
  - split private/public listener groups
  - plural bind address variables
  - emergency viewer security headers
  - simulator stream-completion flow
- Moved stale one-off prompts into historical `features/`, `refactors/`, or `archive/` wording.
- Removed stale singular-bind-only wording from active prompts.
- Added clearer warnings not to expose private `/v1` routes publicly.
- Added stricter reminders around raw token logging, path traversal, and ZIP bundle download safety.

## Suggested install

Copy the directories/files into the repository root.

Recommended first commit:

```bash
git add AGENTS.md codex
git commit -m "Update Codex prompt library"
```

Then run:

```bash
cd server
gofmt -w .
go test ./...
```

Docs-only prompt changes should not alter backend behaviour, but tests remain a nice little reality anchor.
