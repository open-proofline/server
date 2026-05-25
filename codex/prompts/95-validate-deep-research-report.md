# Codex Prompt: Validate Deep Research Technical Review Report

Validate, citation-convert, clean, and public-harden a Deep Research technical review report for this repository.

This is the Phase 2 workflow after a source-cited Deep Research draft. Phase 1 produces a broad report and a portable source registry. Phase 2 checks whether the report is actually true against the repository, converts citations into public-safe Markdown, ensures future-design claims are clearly separated from implemented behavior, and scopes any generated draft issues to the current branch.

## Inputs

Repository:

```text
TheSilkky/safety-recorder
```

Reviewed branch or ref:

```text
<REVIEWED_BRANCH_OR_REF>
```

Report path:

```text
<REPORT_PATH>
```

Reviewed commit SHA:

```text
<REVIEWED_COMMIT_SHA>
```

Target release / version:

```text
<TARGET_RELEASE_OR_VERSION>
```

Output report path:

```text
docs/reports/<YYYY-MM-DD>-safety-recorder-<TARGET_RELEASE_OR_VERSION>-technical-review.md
```

Issue handling mode:

```text
drafts_only
```

Allowed values:

- `drafts_only`: create or update local branch-scoped issue drafts only
- `create_issues`: create GitHub issues only if the maintainer explicitly requested it in the current task
- `none`: do not create issue drafts or GitHub issues

## Rules

- Use the current checked-out branch.
- If `Reviewed branch or ref` is supplied, verify the current checked-out branch or `HEAD` corresponds to that ref before editing. If the current branch does not match and the maintainer did not explicitly approve using the current checkout, stop and ask for clarification.
- Treat the branch/ref as workflow context only. Pin repository citations and report metadata to `<REVIEWED_COMMIT_SHA>`, not to a moving branch name.
- Scope local issue drafts to the current checked-out branch. Do not create branch-ambiguous draft issues from a release-prep or feature branch.
- Do not create or checkout another branch unless explicitly requested.
- Do not change application code.
- Do not change GitHub repository settings.
- Do not change CI workflow behavior unless explicitly requested.
- Do not create GitHub issues unless `Issue handling mode` is `create_issues` and the maintainer explicitly asked for issue creation.
- Keep changes scoped to validating, citation-converting, cleaning the report, and creating branch-scoped draft issues if requested.
- Keep the report and issue drafts public-safe.
- Do not include raw tokens, secrets, private deployment details, exploit payloads, user-safety data, raw keys, plaintext media, or private vulnerability details.
- Do not weaken security warnings.
- Do not claim production readiness.
- Do not claim App Store approval, legal review, or compliance certification.
- Do not treat absence of future-design features as an undisclosed defect when repository docs clearly mark them out of scope.
- Preserve the current backend ciphertext-only implementation boundary unless the report explicitly identifies implemented behavior that contradicts it.
- Treat future iOS/key-custody/browser-decryption/break-glass documents as planning documents unless implementation files exist in the reviewed tree.
- If the report appears to contain sensitive vulnerability details unsafe for public documentation, stop and summarize privately-safe remediation steps.

## First steps

Check repository state:

```bash
git status --short --branch --untracked-files=all
git branch --show-current
git rev-parse HEAD
git log --oneline -5
```

Read source-of-truth docs:

```bash
sed -n '1,220p' README.md
sed -n '1,220p' SECURITY.md
sed -n '1,240p' AGENTS.md
sed -n '1,260p' docs/README.md
sed -n '1,260p' docs/development.md
sed -n '1,320p' docs/security-model.md
sed -n '1,320p' docs/threat-model.md
sed -n '1,320p' docs/api.md
sed -n '1,320p' docs/encryption.md
sed -n '1,320p' docs/deployment.md
```

Read future-design/planning docs when present:

```bash
test -f docs/key-custody.md && sed -n '1,360p' docs/key-custody.md
test -f docs/browser-decryption.md && sed -n '1,320p' docs/browser-decryption.md
test -f docs/break-glass-key-access.md && sed -n '1,320p' docs/break-glass-key-access.md
test -f docs/ios-local-recorder-prototype.md && sed -n '1,360p' docs/ios-local-recorder-prototype.md
```

If an `ios/` directory or Swift/Xcode files exist, inspect them as current implementation rather than future planning:

```bash
find ios -maxdepth 3 -type f 2>/dev/null | sort | sed -n '1,200p'
find . -maxdepth 4 \( -name '*.swift' -o -name 'Package.swift' -o -name '*.xcodeproj' -o -name '*.xcworkspace' -o -name '*.entitlements' -o -name 'Info.plist' \) -print | sort | sed -n '1,200p'
```

Read the report:

```bash
sed -n '1,360p' <REPORT_PATH>
```

Before editing, summarize:

1. reviewed branch/ref supplied
2. current branch
3. reviewed commit SHA in the report
4. current `HEAD`
5. whether current branch and reviewed branch/ref match or are intentionally different
6. whether the maintainer supplied a reviewed commit SHA
7. target release/version
8. whether repository citations are pinned to a commit
9. whether the report includes a portable source registry
10. whether the report uses portable citation keys in the body
11. whether internal ChatGPT citation tokens are present
12. whether the report contains public-safety risks
13. whether future-planning documents are clearly separated from implemented behavior
14. whether iOS/Swift/Apple-platform claims cite authoritative Apple or Swift sources
15. whether issue drafts should be created and what branch scope they should use
16. likely files to update
17. validation commands

If the report has no reviewed commit SHA and the maintainer did not explicitly say current `HEAD` is accurate, stop and ask for the reviewed commit SHA. If the maintainer explicitly said current `HEAD` is accurate, use `git rev-parse HEAD` as the reviewed commit SHA and state that assumption in the report.

## Branch-scoped issue draft policy

When `Issue handling mode` is `drafts_only`, issue drafts must be scoped to the current checked-out branch.

Determine:

```bash
CURRENT_BRANCH="$(git branch --show-current)"
CURRENT_HEAD="$(git rev-parse HEAD)"
```

Use a filesystem-safe branch slug for draft output. Replace `/` and other non-alphanumeric separators with `-`.

Examples:

```text
release/v0.5.0-prep -> release-v0.5.0-prep
feature/foo -> feature-foo
```

Create branch-scoped report issue drafts under:

```text
.backlog-drafts/<YYYY-MM-DD>/<branch-slug>/
```

If the date is unavailable:

```text
.backlog-drafts/current/<branch-slug>/
```

Each draft issue created from the report must include this section near the top:

```md
## Branch scope

- Current branch: `<CURRENT_BRANCH>`
- Current HEAD: `<CURRENT_HEAD>`
- Reviewed branch/ref: `<REVIEWED_BRANCH_OR_REF>`
- Reviewed commit SHA: `<REVIEWED_COMMIT_SHA>`
- Target release/version: `<TARGET_RELEASE_OR_VERSION>`
- Scope note: This draft was generated from a report against the branch above. Revalidate against the target branch before creating or closing public GitHub issues if the branch has moved or has not yet merged.
```

Branch-specific findings must be classified as one of:

```text
release-blocker-current-branch
follow-up-after-merge
revalidate-on-main-or-develop
planning-only
sensitive-do-not-publicly-file
```

Do not create public GitHub issues directly from branch-scoped drafts unless the maintainer explicitly requests `create_issues` and the draft has been reviewed for current target-branch relevance.

If `Issue handling mode` is `create_issues`, first confirm the intended target branch for the public issue body. Preserve the `Branch scope` section in the issue body unless the maintainer explicitly asks to remove it.

## Citation conversion workflow

Deep Research may produce ChatGPT-rendered citations such as `cite...`, `filecite...`, raw `turn...` references, or other UI-only source markers.

The final public report must not contain any of those internal citations.

Use the Phase 1 portable source registry as the primary mapping source for citation conversion.

If the source registry is missing, incomplete, or cannot support a claim:

1. Locate a portable repository URL pinned to `<REVIEWED_COMMIT_SHA>` or a canonical external URL.
2. Add a stable citation key to the source registry or reference definitions.
3. Replace the internal citation with the portable key.
4. If no portable source can be found, remove or downgrade the claim.

Use these key families:

- `R-*` for repository sources pinned to `<REVIEWED_COMMIT_SHA>`
- `S-*` for external authoritative sources
- `I-*` for issue, PR, or report-follow-up references
- `V-*` for validation evidence

Repository source definitions must use this form:

```markdown
[R-README]: https://github.com/TheSilkky/safety-recorder/blob/<REVIEWED_COMMIT_SHA>/README.md
```

Do not use `blob/main` or branch names in final repository citation URLs.

External source definitions must use canonical source URLs, not ChatGPT renderer IDs.

Validation source definitions must point to public CI URLs, uploaded validation summaries, or documented maintainer/Codex evidence. Do not include raw tokens, secrets, private deployment details, request bodies, uploaded bytes, plaintext, raw keys, or user-safety data.

## Output

Create the cleaned report, branch-scoped issue drafts if requested, and summarize:

1. report input path
2. cleaned report output path
3. reviewed branch/ref used
4. current branch used for issue draft scope
5. reviewed commit SHA used
6. target release/version used
7. ChatGPT-rendered citations removed or converted
8. source registry gaps or corrections
9. unsupported claims removed or corrected
10. implementation claims vs future-planning claims corrected
11. iOS/Swift/Apple-platform claims corrected or source-pinned
12. findings retained, removed, downgraded, or reframed
13. issue drafts or GitHub issues created, if any
14. branch-scoped draft directory used, if any
15. validation commands run
16. whether the report is suitable for public `docs/reports/` publication
17. any manual follow-up required

Do not claim the report is a formal audit. Do not claim production readiness. Do not claim legal/App Store approval.
