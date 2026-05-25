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

When `Issue handling mode` is `drafts_only`, issue drafts must be scoped to the current checked-out branch and must include complete issue metadata.

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

## Required issue draft structure

Every public issue draft created from the report must use this structure and section order:

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
- Every public draft must include at least `backlog` under `## Labels`.
- Include one or more type/topic labels that match the issue.
- Use likely existing labels only: `backlog`, `bug`, `maintenance`, `security`, `docs`, `deployment`, `testing`, `simulator`, `ios`, `ci`, `planning`.
- Do not invent new labels unless the maintainer explicitly asked for label creation.

The older `## Classification` section is optional. If included, it must not replace `## Priority`, `## Type`, `## Labels`, or `## Branch scope`.

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

## Repository-claim validation

For every report finding and major claim:

1. Locate the repository evidence.
2. Confirm the cited file exists.
3. Confirm the behavior is implemented as described.
4. Confirm the report distinguishes current implementation from future design/planning.
5. Confirm the report does not turn documented out-of-scope features into false defects.
6. If a claim is unsupported, either remove it, downgrade it, or rewrite it as an uncertainty.

Validate future-planning claims separately:

1. Confirm that documents such as `docs/key-custody.md`, `docs/browser-decryption.md`, `docs/break-glass-key-access.md`, and `docs/ios-local-recorder-prototype.md` are described as planning/design docs unless implementation exists.
2. Confirm iOS recorder claims do not imply an iOS app exists unless `ios/` or Swift/Xcode files exist.
3. Confirm Swift, AVFoundation/AVFAudio, iOS lifecycle, BackgroundTasks, URLSession background transfer, CryptoKit, Keychain, file protection, and App Store claims cite Apple or Swift primary sources.
4. Confirm the report does not claim Apple/App Store approval or legal compliance.
5. Confirm planning recommendations distinguish "prototype can test this" from "production design is solved."

Pay special attention to these common failure modes:

- claiming `server/.dockerignore` is missing when it exists
- claiming GitHub repository settings were audited when only repository files were reviewed
- claiming production readiness
- claiming backend decryption exists when current code remains ciphertext-only
- claiming absence of iOS/user accounts/OAuth/JWT/SMS/push/browser decryption is a defect when docs mark those out of scope
- claiming future planning documents are implemented features
- claiming the iOS recorder prototype exists as code when only `docs/ios-local-recorder-prototype.md` exists
- claiming Keychain-only prototype storage solves production key custody when docs say the iPhone may be unavailable
- claiming background execution permits indefinite recording/uploading without testing and Apple-source support
- claiming background camera/video capture is supported without Apple-source support
- claiming URLSession background transfers solve all upload/retry requirements without lifecycle caveats
- claiming App Store acceptance, legal compliance, or safety certification
- claiming Deep Research ran tests, Docker builds, containers, or simulator smoke tests
- using external standards as decorative citations when repository code or docs are the real evidence
- leaving raw Deep Research citation tokens in Markdown
- leaving repository URLs pinned to `main` instead of the reviewed commit SHA
- leaving repository URLs pinned to a branch name such as `<REVIEWED_BRANCH_OR_REF>` instead of the reviewed commit SHA
- creating issue drafts without priority, type, GitHub labels, or branch scope

## Editing requirements

Create a cleaned public report.

Required edits:

- Add or verify title metadata:
  - repository
  - reviewed branch/ref, if supplied
  - reviewed commit SHA
  - target release/version
  - review date
  - final report status
  - citation format note
  - AI-assisted review disclosure
  - public-disclosure note
- Remove `Claims check` sections from the final public report unless explicitly requested.
- Remove `Verify before sending` sections.
- Remove ChatGPT-rendered citations and replace them with portable keys.
- Use neutral report wording, not first-person model wording.
- Preserve useful findings.
- Remove unsupported findings.
- Downgrade or reframe findings when repository evidence contradicts the initial draft.
- Reframe future-planning issues as planning/source-support gaps unless the reviewed tree implements the feature.
- Keep all citations portable.
- Pin all repository links to the reviewed commit SHA.
- Keep external source links canonical.
- Ensure every `[R-*]`, `[S-*]`, `[I-*]`, and `[V-*]` citation key has a definition.
- Keep the report suitable for public `docs/reports/` publication.
- Create branch-scoped draft issues only when requested by `Issue handling mode`.
- Ensure every public issue draft includes `## Priority`, `## Type`, `## Labels`, and `## Branch scope`.

Use this status wording unless the maintainer asks otherwise:

```markdown
**Report status:** Final public report after maintainer Phase 2 validation; accepted findings were mapped to follow-up issues.
```

Use this citation note unless a better project-specific version is needed:

```markdown
**Citation format note:** This report uses portable citation keys only. Repository citations are pinned to reviewed commit `<REVIEWED_COMMIT_SHA>`; external citations resolve to canonical documentation URLs. No ChatGPT-internal citation tokens are used.
```

Use this disclosure unless the maintainer asks otherwise:

```markdown
**AI-assisted review disclosure:** This report was generated with assistance from OpenAI ChatGPT Deep Research using <MODEL_NAME>, then reviewed and edited by the maintainer. It is not a formal security audit, penetration test, compliance certification, legal review, App Store review, or production-readiness endorsement. Findings should be verified against the reviewed commit, cited sources, and current project scope before being relied on.
```

## Issue handling

If `Issue handling mode` is `none`, do not create issue drafts or GitHub issues.

If `Issue handling mode` is `drafts_only`, write branch-scoped issue drafts under:

```text
.backlog-drafts/<YYYY-MM-DD>/<branch-slug>/
```

If the date is unavailable:

```text
.backlog-drafts/current/<branch-slug>/
```

Each draft should include:

- title
- priority
- type
- suggested GitHub labels
- branch scope
- summary
- context
- proposed change
- acceptance criteria
- tests / validation
- out of scope
- notes
- report finding ID

For iOS or future-planning findings, issue drafts must clearly say whether the work is planning-only, prototype implementation, backend API work, or security/key-custody design. Do not create iOS implementation issues that imply the app already exists.

If `Issue handling mode` is `create_issues`, first check for duplicates and labels:

```bash
gh issue list --repo TheSilkky/safety-recorder --state open --limit 100
gh label list --repo TheSilkky/safety-recorder --limit 200
```

Then create issues only for accepted findings that the maintainer explicitly approved for public issue tracking.

Do not create an issue from any draft unless it has `## Priority`, `## Type`, `## Labels`, and `## Branch scope`.

Preserve `## Priority`, `## Labels`, and `## Branch scope` in the public issue body.

Do not silently create unlabeled issues.
Do not silently drop missing labels.
Do not create labels unless explicitly requested.

Do not include sensitive vulnerability details, raw tokens, private deployment information, exploit payloads, raw keys, plaintext media, or user-safety data in public issues.

## Validation after editing

Run:

```bash
python3 - <<'PY'
from pathlib import Path
import re

path = Path("<OUTPUT_REPORT_PATH>")
text = path.read_text(encoding="utf-8")

checks = {
    "internal filecite tokens": "\\ue200filecite\\ue202[^\\ue201]+\\ue201",
    "internal cite tokens": "\\ue200cite\\ue202[^\\ue201]+\\ue201",
    "raw turn refs": r"turn\\d+(?:file|view|search|fetch|open)\\d+",
    "citation glyphs": "[\\ue200\\ue202\\ue201]",
    "blob main repo URLs": r"github\\.com/TheSilkky/safety-recorder/blob/main",
    "claims check": r"(?i)claims check",
    "verify before sending": r"(?i)verify before sending",
    "production-ready claim": r"(?i)production[- ]ready(?!.*not)",
    "app store approval claim": r"(?i)(app store approved|app store approval|will pass app review)",
}

for name, pat in checks.items():
    count = len(re.findall(pat, text))
    print(f"{name}: {count}")

defs = set(re.findall(r"^\\[([A-Za-z0-9_-]+)\\]:\\s+\\S+", text, flags=re.M))
uses = set(re.findall(r"(?<!^)\\[([A-Za-z0-9_-]+)\\]", text, flags=re.M))
citation_uses = {u for u in uses if u.startswith(("R-", "S-", "I-", "V-"))}
print("missing citation definitions:", sorted(citation_uses - defs))
print("unused citation definitions:", sorted(defs - citation_uses))
PY
```

Validate branch-scoped draft issues when drafts were created:

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

For Markdown-only report edits, Go tests are not required. If code changed accidentally, stop and explain the scope problem.

## Output

Summarize:

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
15. confirmation that every public issue draft includes Priority, Type, Labels, and Branch scope
16. labels used or missing
17. validation commands run
18. whether the report is suitable for public `docs/reports/` publication
19. any manual follow-up required

Do not claim the report is a formal audit. Do not claim production readiness. Do not claim legal/App Store approval.
