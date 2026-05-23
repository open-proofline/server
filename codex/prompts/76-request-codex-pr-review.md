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
- whether the change satisfies the linked issue acceptance criteria
- whether unrelated scope was added
- tests and validation
- private/public listener separation
- raw token / request body / uploaded byte / Authorization header logging
- plaintext/key logging
- backend decryption, server-side key access, or key custody changes accidentally introduced or introduced without explicit design scope
- whether explicit key custody/decryption changes update threat model, security model, encryption docs, tests, and operational guidance
- ZIP bundle path safety, if relevant
- encryption/key handling, if relevant
- documentation accuracy
- production-readiness claims

Do not suggest unrelated features.
Do not include sensitive vulnerability details in a public comment.
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
7. whether it satisfies the linked issue acceptance criteria
8. whether it should remain draft
9. whether it changes key custody/decryption assumptions, and whether those changes are explicitly designed and documented

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
