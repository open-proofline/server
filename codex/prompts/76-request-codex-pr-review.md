# Codex Prompt: Request Codex Review on Pull Request

Use this when a draft pull request exists and should be reviewed by Codex.

## Inputs

Pull request number: `<PR_NUMBER>`

Repository:

```text
TheSilkky/safety-recorder
```

## Option A: Comment directly on the PR

Post this PR comment:

```md
@codex review

Please review this PR for correctness, security, scope control, and consistency with README.md, AGENTS.md, SECURITY.md, and relevant docs.

Focus on:
- whether the change actually satisfies the linked issue
- whether unrelated scope was added
- tests and validation
- private/public listener separation
- raw token / request body / uploaded byte / Authorization header logging
- ZIP bundle path safety, if relevant
- encryption/key handling, if relevant
- documentation accuracy
- production-readiness claims

Do not suggest unrelated features.
```

## Option B: Ask Codex locally to review a PR

Use this in the IDE if reviewing locally:

```md
Review pull request #<PR_NUMBER> in `TheSilkky/safety-recorder`.

First fetch PR metadata and diff.

Then review for:

1. correctness
2. security
3. scope control
4. test coverage
5. documentation accuracy
6. consistency with README.md and AGENTS.md
7. whether it satisfies the linked issue

Do not modify files unless explicitly requested.

Return:
- blocking issues
- non-blocking issues
- suggested minimal fixes
- whether the PR should remain draft
- whether the PR appears ready for human review
```

## Constraints

- Do not merge the PR.
- Do not mark the PR ready for review.
- Do not approve your own changes.
- Do not request changes unless the review is genuinely blocking.
- Do not include sensitive vulnerability details in public comments.
