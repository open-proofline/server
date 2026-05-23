# Codex Prompt: Review Open Issues for Stale or Fixed Work

Review open GitHub issues and identify issues that may be fixed, stale, duplicate, superseded, or still valid.

This is an issue-maintenance prompt.

Do **not** change application code.
Do **not** change documentation unless explicitly requested.
Do **not** create new issues unless explicitly requested.
Do **not** close issues unless explicitly requested.
Do **not** merge pull requests.
Do **not** make repository changes as part of the initial review.

## Goal

Review open issues against the current repository state.

For each open issue, determine whether it is:

- still valid
- fixed by current code/docs
- partially fixed
- stale
- duplicate
- superseded by another issue/design doc
- sensitive and unsuitable for public discussion

When an issue appears fixed, identify the specific commit or merged PR that resolved it.

## Repository

```text
TheSilkky/safety-recorder
```

## Source of truth

Read current repository state:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- `docs/README.md`
- relevant files in `docs/`
- relevant source files
- `.github/workflows`
- `codex/README.md`
- `codex/prompts`

Inspect GitHub issues and PRs:

```bash
gh issue list --repo TheSilkky/safety-recorder --state open --limit 100
gh issue list --repo TheSilkky/safety-recorder --state closed --limit 100
gh pr list --repo TheSilkky/safety-recorder --state all --limit 100
```

For each open issue under review:

```bash
gh issue view <ISSUE_NUMBER> --repo TheSilkky/safety-recorder
```

If the issue may be fixed, search commits and PRs:

```bash
git log --oneline --all --grep "<keywords>"
git log --oneline --all -- <relevant-file>
gh pr list --repo TheSilkky/safety-recorder --state merged --search "<keywords>"
```

Use whatever subset is available. If GitHub CLI or local git history is unavailable, say so and mark confidence lower.

## Review method

For each open issue:

1. Read the title, body, labels, acceptance criteria, and comments.
2. Identify the files/docs/code areas it references.
3. Check current repository state for those files/areas.
4. Check whether acceptance criteria are satisfied.
5. Search commits/PRs that likely resolved it.
6. Decide one status:
   - `keep-open`
   - `close-fixed`
   - `close-duplicate`
   - `close-superseded`
   - `needs-update`
   - `needs-human-review`
   - `sensitive-do-not-publicly-discuss`
7. If recommending closure, identify exact commit SHA(s) or PR number(s).
8. Draft a closure or update comment.

## Evidence requirement

Do not recommend closing an issue as fixed unless you can point to at least one of:

- specific commit SHA
- merged PR number
- current file path and section that satisfies the issue
- changelog entry documenting the fix

Prefer commit SHA plus file path.

Example:

```text
Resolved by commit c9d847ae2b26cfc22c6dbd728b491933466eca35, which added docs/key-custody.md.
```

If evidence is ambiguous, use `needs-human-review`, not `close-fixed`.

## Sensitive issue handling

If an issue contains or implies a private vulnerability:

- do not draft a public closure comment with exploit details
- do not include raw tokens, secrets, private deployment details, exploit payloads, or user safety data
- refer to `SECURITY.md`
- mark status `sensitive-do-not-publicly-discuss`

## Output directory

Create review drafts under:

```text
.issue-review-drafts/YYYY-MM-DD/
```

If the date is unavailable, use:

```text
.issue-review-drafts/current/
```

Create:

```text
.issue-review-drafts/YYYY-MM-DD/README.md
```

For each issue reviewed, create:

```text
.issue-review-drafts/YYYY-MM-DD/issue-<ISSUE_NUMBER>.md
```

## Per-issue review format

Each issue review file must use this structure:

```md
# Issue #<number>: <title>

## Recommendation

keep-open / close-fixed / close-duplicate / close-superseded / needs-update / needs-human-review / sensitive-do-not-publicly-discuss

## Confidence

high / medium / low

## Summary

One or two sentences.

## Evidence reviewed

- Issue acceptance criteria:
  - ...
- Relevant files:
  - ...
- Relevant commits or PRs:
  - ...

## Analysis

Explain whether the current repo satisfies the issue.

## Suggested maintainer action

What the maintainer should do next.

## Draft comment

Suggested GitHub issue comment.

## Safe to close automatically?

yes / no

## Notes

Any caveats.
```

## Index README format

The index should group issues by recommendation:

```md
# Issue Review

## Close as fixed

| Issue | Confidence | Evidence |
|---|---|---|

## Keep open

...

## Needs update

...

## Human review

...

## Sensitive

...
```

## Optional closure script

Do **not** create a closure script unless explicitly requested.

If explicitly requested, create:

```text
scripts/close-fixed-issues.sh
```

The script must:

- only close issues marked `Safe to close automatically? yes`
- post the draft comment first
- then close the issue
- not close sensitive issues
- not close low-confidence issues
- warn that running twice may duplicate comments

Use:

```bash
gh issue comment <number> --repo TheSilkky/safety-recorder --body-file <comment-file>
gh issue close <number> --repo TheSilkky/safety-recorder --reason completed
```

## Constraints

- Do not modify application code.
- Do not modify docs.
- Do not create or close issues unless explicitly requested.
- Do not execute closure commands.
- Do not include sensitive details in public comments.
- Keep all review output in `.issue-review-drafts/`.

## Validation

After creating drafts:

```bash
git diff --stat
git diff -- .issue-review-drafts
```

Do not run Go tests unless code changed.

If any files outside `.issue-review-drafts/` changed, stop and explain why.

## Output

Summarize:

1. draft directory created
2. issues reviewed
3. recommended closures
4. issues to keep open
5. issues needing update
6. sensitive items, if any
7. closure candidates with commit/PR evidence
8. whether any repository files outside `.issue-review-drafts/` changed
