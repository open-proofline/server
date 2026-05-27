# Codex Prompt: Validate Deep Research Technical Review Report

Validate, clean, and public-harden a Deep Research technical review report for this repository.

This is the Phase 2 workflow after a source-cited Deep Research draft. Phase 1 produces a broad report and portable source registry. Phase 2 checks the report against the reviewed repository commit, converts citations into public-safe Markdown, separates future design from implemented behavior, and scopes any generated draft issues to the current branch.

## Inputs

Repository:

```text
TheSilkky/safety-recorder
```

Reviewed branch or ref:

```text
<REVIEWED_BRANCH_OR_REF>
```

Reviewed commit SHA:

```text
<REVIEWED_COMMIT_SHA>
```

Report path:

```text
<REPORT_PATH>
```

Target release / version:

```text
<TARGET_RELEASE_OR_VERSION>
```

Output report path:

```text
docs/reports/<YYYY-MM-DD>-proofline-<TARGET_RELEASE_OR_VERSION>-technical-review.md
```

Issue handling mode:

```text
drafts_only
```

Allowed values:

- `drafts_only`: create or update local branch-scoped issue drafts only
- `create_issues`: create GitHub issues only when the maintainer explicitly requested it
- `none`: do not create issue drafts or GitHub issues

## Product Context

Product documentation now uses the name Proofline. Repository paths, Go module paths, Docker image names, GHCR package names, current route names, and compatibility names may still use `safety-recorder` or `emergency` until explicit migrations are performed.

Proofline's planned product scope includes emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. The current backend stores generic incidents unless the reviewed tree explicitly implements first-class incident types or escalation policies.

## Rules

- Use the current checked-out branch.
- Pin repository citations and report metadata to `<REVIEWED_COMMIT_SHA>`, not to a moving branch name.
- Keep changes scoped to report validation, citation cleanup, and branch-scoped draft issues if requested.
- Do not change application code, CI behavior, repository settings, or GitHub issues unless explicitly requested.
- Keep the report and any issue drafts public-safe according to `SECURITY.md`.
- Do not weaken security warnings.
- Do not claim production readiness, platform-store approval, legal review, compliance certification, penetration test, or formal audit.
- Do not treat absence of future-design features as a defect when source-of-truth docs mark them out of scope.
- Preserve the current backend ciphertext-only implementation boundary unless the report identifies implemented behavior that contradicts it.
- Treat future incident-mode, web/iOS/Android client, key-custody, browser-decryption, and break-glass documents as planning unless implementation files exist in the reviewed tree.

## First Steps

Check repository state:

```bash
git status --short --branch --untracked-files=all
git branch --show-current
git rev-parse HEAD
git log --oneline -5
```

Read source-of-truth docs:

```bash
sed -n '1,240p' README.md
sed -n '1,220p' SECURITY.md
sed -n '1,260p' AGENTS.md
sed -n '1,280p' docs/README.md
sed -n '1,260p' docs/incident-modes.md
sed -n '1,320p' docs/security-model.md
sed -n '1,340p' docs/threat-model.md
sed -n '1,360p' docs/api.md
sed -n '1,340p' docs/encryption.md
sed -n '1,360p' docs/deployment.md
```

Read future-design docs when present:

```bash
test -f docs/key-custody.md && sed -n '1,360p' docs/key-custody.md
test -f docs/browser-decryption.md && sed -n '1,360p' docs/browser-decryption.md
test -f docs/break-glass-key-access.md && sed -n '1,360p' docs/break-glass-key-access.md
test -f docs/ios-local-recorder-prototype.md && sed -n '1,360p' docs/ios-local-recorder-prototype.md
```

Read the report:

```bash
sed -n '1,420p' <REPORT_PATH>
```

Before editing, summarize:

1. reviewed branch/ref and reviewed commit SHA
2. current branch and current `HEAD`
3. target release/version
4. whether citations are portable and commit-pinned
5. whether a source registry exists and supports material claims
6. whether future-planning docs are separated from implemented behavior
7. whether Proofline naming and compatibility-name notes are represented correctly
8. whether incident-mode planning is represented as planning unless implemented
9. whether issue drafts should be created and what branch scope they should use
10. likely files to update and docs-review checks

## Branch-Scoped Issue Drafts

When `Issue handling mode` is `drafts_only`, issue drafts must be scoped to the current branch.

Drafts belong under:

```text
.backlog-drafts/<YYYY-MM-DD>/<branch-slug>/
```

If the date is unavailable:

```text
.backlog-drafts/current/<branch-slug>/
```

Every public issue draft should include priority, type, labels, branch scope, summary, context, proposed change, acceptance criteria, tests/validation, out-of-scope notes, and report reference notes.

Use only existing labels. If a good topic label does not exist, use the closest existing label and note the mismatch.

## Report Validation Checklist

Check and fix, if needed:

- Product name is Proofline where describing current docs/product direction.
- Compatibility names remain when describing current artifacts, APIs, routes, config, or packages.
- Repository facts are pinned to `<REVIEWED_COMMIT_SHA>`.
- Future incident modes are marked as planning unless implemented.
- Current `/v1` private boundary and public incident-viewer separation remain clear.
- Current backend ciphertext-only behavior is represented accurately.
- Historical report names are not rewritten as if they used the new product name at the time.
- ChatGPT internal citation tokens are removed or converted to portable citation keys.
- Source Registry entries support material claims.
- External-source omissions are disclosed and affected claims are marked not independently verified.

## Common False Positives To Remove Or Downgrade

- Missing public `/v1` authentication when docs state `/v1` is private and unauthenticated.
- Missing web/iOS/Android clients when docs mark them as future work.
- Missing first-class incident types, escalation policies, or dead-man switch when docs mark them as future work.
- Missing browser decryption, production key custody, or break-glass behavior when docs mark them as future work.
- Preserved artifact or route names treated as stale after a docs-only Proofline rename.
- Interaction-record planning treated as current implementation.
- Backend decryption or server-held keys assumed from future design docs.

## Validation

For report/docs-only cleanup:

```bash
git diff --stat
git diff --check
```

If Go code changed unexpectedly, stop and report scope creep.

## Output

Summarize:

1. current branch
2. reviewed branch/ref and commit SHA
3. report path updated
4. issue drafts created, if any
5. citation and public-safety cleanup performed
6. Proofline naming and incident-mode boundary corrections made
7. validation/docs-review commands run
8. follow-up work
