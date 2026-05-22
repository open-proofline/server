# Codex Change Control

This workflow keeps Codex-assisted changes practical and reversible. It is meant to feel like a small sysadmin change window, not paperwork for its own sake.

Codex can draft changes, but the maintainer remains responsible for reviewing, testing, accepting, committing, or rolling them back.

## Codex-assisted change workflow

### Before running Codex

1. Check the working tree:

   ```bash
   git status
   ```

2. If there are useful uncommitted changes, commit them or intentionally stash them before asking Codex to modify files.

3. Create a rollback point for larger tasks:

   ```bash
   git add .
   git commit -m "checkpoint before <task>"
   ```

4. Read the current source-of-truth docs:

   - `README.md`
   - `AGENTS.md`
   - relevant files in `docs/`
   - relevant prompt in `codex/prompts/`

5. Define:

   - goal
   - files likely affected
   - files that must not change
   - tests or validation commands
   - explicit out-of-scope items

### After Codex makes changes

1. Review the changed files:

   ```bash
   git diff --stat
   git diff
   ```

2. If code changed, run:

   ```bash
   cd server
   gofmt -w .
   go test ./...
   go vet ./...
   ```

3. If only Markdown changed, inspect docs and links manually. Go tests are not required unless code changed.

4. For behavior changes, run the simulator smoke test:

   ```bash
   cd server
   go run ./cmd/api
   ```

   In another terminal:

   ```bash
   cd server
   go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
   ```

5. If the change is good, commit it deliberately.

6. If the change is bad, revert or reset to the checkpoint. Do not try to rescue a sprawling bad diff by asking Codex for more sprawling changes.

### Rollback

For an uncommitted bad change:

```bash
git restore .
```

For a bad change after a checkpoint commit, inspect history first:

```bash
git log --oneline -5
```

Then either revert with a new commit:

```bash
git revert <commit>
```

or reset only if you understand the consequence and have not pushed:

```bash
git reset --hard <checkpoint>
```

Prefer `git revert` for pushed history.

## Issue-first future work

Codex tasks should stay narrow. If Codex or the maintainer discovers a new idea during a task, do not implement it unless it is necessary for the current task.

Create an issue or backlog item instead. Good issues include context, acceptance criteria, tests, docs updates, and explicit out-of-scope notes.

Security vulnerabilities should follow `SECURITY.md`, not public issue templates. Security hardening work that is not a private vulnerability can use a normal issue template.

Use this shape for backlog items:

```md
## Summary

One or two sentences.

## Context

Why this matters.

## Proposed change

What should change.

## Acceptance criteria

- [ ] ...
- [ ] ...
- [ ] ...

## Tests / validation

- [ ] `go test ./...`
- [ ] simulator smoke test, if relevant
- [ ] docs updated, if relevant

## Out of scope

What this issue should not include.
```
