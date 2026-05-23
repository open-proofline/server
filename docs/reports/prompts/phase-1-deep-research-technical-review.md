# Phase 1 Deep Research Prompt: Public Technical Review Report

Use this prompt in ChatGPT Deep Research, not in Codex.

This prompt creates the first-pass source-cited technical review report. The output is expected to be validated and cleaned by the Phase 2 Codex workflow before publication, because apparently even carefully cited robots still need adult supervision.

## Inputs

Repository:

```text
TheSilkky/safety-recorder
```

Reviewed commit SHA:

```text
<REVIEWED_COMMIT_SHA>
```

Review date:

```text
<YYYY-MM-DD>
```

Target report path, if known:

```text
docs/reports/<YYYY-MM-DD>-safety-recorder-technical-review.md
```

Model / tool disclosure:

```text
OpenAI ChatGPT Deep Research using <MODEL_NAME>
```

## Repository context

Safety Recorder is an experimental Go backend for private personal-safety recording. It receives already-encrypted recording chunks, stores metadata in SQLite, stores encrypted blobs on local disk, and exposes a token-scoped read-only emergency viewer.

Core project boundaries:

- The private `/v1` API has no public user authentication and must not be exposed publicly.
- The current backend treats uploaded bytes as opaque ciphertext.
- The current backend must not be described as production-ready public infrastructure.
- Backend decryption, browser decryption, production key custody, break-glass access, user accounts, OAuth/JWT, push notifications, SMS, Messenger, and iOS recording implementation are future or out-of-scope items unless explicitly implemented in the reviewed tree.
- Future key custody documents are design guardrails, not shipped implementation.

## Source policy

Use authoritative sources only.

Prioritize:

- repository files in the reviewed tree
- `go.dev`
- `pkg.go.dev`
- `csrc.nist.gov`
- `nvlpubs.nist.gov`
- `owasp.org`
- `cheatsheetseries.owasp.org`
- `docs.github.com`
- `docs.docker.com`
- `sqlite.org`
- `rfc-editor.org`
- `datatracker.ietf.org`

Avoid relying on:

- random blogs
- Stack Overflow
- social posts
- vendor marketing pages
- AI-generated summaries
- uncited claims

If a secondary source is used, explain why a primary source was not sufficient.

## Citation requirements

Use portable citation keys only.

Do not use ChatGPT internal citation tokens such as renderer-only `filecite` / `cite` blocks or raw `turnXfileY`, `turnXviewY`, or `turnXsearchY` references.

Use this citation style:

```markdown
Repository fact. [R-README] [R-CI]

External-source fact. [S-GITHUB-ACTIONS-SECURE]
```

At the end of the report, include Markdown reference definitions for every citation key:

```markdown
[R-README]: https://github.com/TheSilkky/safety-recorder/blob/<REVIEWED_COMMIT_SHA>/README.md
[S-GITHUB-ACTIONS-SECURE]: https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions
```

Repository citations must be pinned to `<REVIEWED_COMMIT_SHA>`, not `main`, unless the commit SHA is genuinely unavailable. If the SHA is unavailable, clearly mark the report as a draft and include a warning that repository URLs must be commit-pinned before publication.

## Review scope

Review these repository areas:

- `README.md`
- `SECURITY.md`
- `CHANGELOG.md`
- `AGENTS.md`
- `docs/`
- `codex/`
- `.github/`
- `server/`
- `server/migrations/`
- `server/Dockerfile`
- `server/.dockerignore`, if present
- GitHub Actions workflows and Dependabot configuration

Technical focus areas:

1. Documentation consistency and public-readiness wording
2. Go backend structure and idiomatic implementation
3. HTTP API behavior and private/public route separation
4. Emergency token generation, hashing, storage, redaction, and viewer behavior
5. Logging and sensitive data exposure
6. Upload handling, hash verification, immutable storage, and size limits
7. SQLite migrations, foreign keys, WAL mode, and data integrity
8. ZIP bundle generation, manifest completeness, and path traversal handling
9. Crypto-adjacent simulator envelope:
   - AES-GCM use
   - nonce generation and uniqueness assumptions
   - associated data construction
   - key generation and simulator key handling
   - ciphertext-only backend boundary
10. Future-design boundary:
   - key custody
   - browser decryption
   - break-glass / dead-man-switch access
11. Docker/GHCR/GitHub Actions/supply-chain hygiene
12. Public issue/report safety

## Finding rules

For every finding, include:

- finding ID
- severity: Critical / High / Medium / Low / Informational
- confidence: High / Medium / Low
- current implementation vs future design
- affected files and functions
- repository evidence citation
- authoritative external citation, if applicable
- why it matters
- minimal actionable fix
- suggested GitHub issue title
- acceptance criteria

Do not inflate severity merely because a finding is security-adjacent. If a limitation is already documented as out of scope, classify it as a non-finding or future-work item unless there is a contradiction between docs and code.

Do not claim “missing” if a file or control exists. If a control exists but is incomplete, describe the narrower hardening task.

## Required report structure

Use this structure:

```markdown
# Technical Review of Safety Recorder v0.4.x

**Repository:** `TheSilkky/safety-recorder`
**Reviewed commit SHA:** `<REVIEWED_COMMIT_SHA>`
**Review date:** `<YYYY-MM-DD>`
**Report status:** Phase 1 draft pending maintainer/Codex validation.

**Citation format note:** ...
**AI-assisted review disclosure:** ...
**Public-disclosure note:** ...

## Executive summary

## Scope and methodology

### AI assistance and review limitations

## Portable source bibliography

### Repository sources

### External authoritative sources

## Repository architecture summary

## Findings

### Findings table

### F-A — ...

## Suggested GitHub issues for follow-up

## Explicit non-findings and limitations

## Appendix: mapping findings to authoritative guidance

[reference definitions]
```

Do not include a “Claims check” section in the publishable report.

Do not include a “Verify before sending” section.

Do not include raw secrets, raw tokens, private deployment details, exploit payloads, or user-safety data.

## Required disclaimer wording

Include this near the top:

```markdown
**AI-assisted review disclosure:** This report was generated with assistance from OpenAI ChatGPT Deep Research using <MODEL_NAME>, then reviewed and edited by the maintainer. It is not a formal security audit, penetration test, compliance certification, or production-readiness endorsement. Findings should be verified against the reviewed commit, cited sources, and current project scope before being relied on.

**Public-disclosure note:** This report is intended for public project documentation. It intentionally avoids raw tokens, secrets, private deployment details, exploit payloads, and user-safety data.
```

## Quality gates before returning the draft

Before returning the report, check and state whether the draft satisfies:

- no ChatGPT internal citation tokens
- no `blob/main` repository URLs when a reviewed commit SHA was supplied
- all repository citations are pinned to `<REVIEWED_COMMIT_SHA>`
- every bracket citation key has a reference definition
- every reference definition is used or intentionally retained
- no raw tokens, secrets, private deployment details, exploit payloads, or user-safety data
- no production-readiness claim
- no formal audit/certification claim
- current implementation and future design are clearly separated
- no “Claims check” section
- no “Verify before sending” section

## Output

Return the full Markdown report.

Then include a short Phase 2 handoff summary:

```text
Phase 2 handoff:
- reviewed commit SHA:
- highest-confidence findings:
- claims that need maintainer verification:
- citations that may need cleanup:
- possible duplicate or existing issues to check:
```
