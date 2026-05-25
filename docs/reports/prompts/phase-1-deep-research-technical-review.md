# Phase 1 Deep Research Prompt: Public Technical Review Report

Use this prompt in ChatGPT Deep Research, not in Codex.

This prompt creates the first-pass source-cited technical review report. The output is expected to be validated and cleaned by the Phase 2 Codex workflow before publication.

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

Target release / version:

```text
<TARGET_RELEASE_OR_VERSION>
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

## Test and validation evidence policy

Deep Research cannot run repository tests, containers, Docker builds, local shell commands, or simulator smoke tests.

Do not claim that Deep Research personally ran tests, built containers, executed Go commands, started the API server, ran the simulator, inspected live GitHub repository settings, or validated a Docker image by executing it.

Use only supplied validation evidence, public CI results, uploaded logs, maintainer-supplied summaries, Codex-supplied command output, or repository workflow files when discussing test/build status.

If no validation evidence is supplied for a command, state that the command was not independently verified by this report.

When validation evidence is supplied, distinguish clearly between:

- repository workflow configuration
- public GitHub Actions / CI run results
- maintainer-supplied local command output
- Codex-supplied command output
- uploaded validation summaries
- inferred expectations from source files or workflow definitions

Do not treat maintainer-supplied logs as proof beyond what they actually show. Do not infer that unobserved commands passed merely because related commands passed.

Recommended evidence to request or use when available:

- exact reviewed branch/ref
- exact reviewed commit SHA
- GitHub Actions run URLs for the reviewed commit
- local or Codex output for `cd server && gofmt -w .`
- local or Codex output for `cd server && go test ./...`
- local or Codex output for `cd server && go vet ./...`
- local or CI output for `docker build -t safety-recorder-backend ./server`
- local or Codex output for the simulator smoke test:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

If validation evidence is available, use wording like:

```markdown
Validation evidence supplied for the reviewed commit indicates that `<COMMAND>` passed in `<ENVIRONMENT>`. This report did not independently execute that command.
```

If validation evidence is unavailable, use wording like:

```markdown
The report reviewed the workflow configuration and source files, but did not independently execute `<COMMAND>` and no validation log was supplied for that command.
```

Do not put raw tokens, secrets, request bodies, uploaded bytes, plaintext, raw keys, private deployment details, or user-safety data into validation summaries or the public report.


## Repository context

Safety Recorder is an experimental Go backend for private personal-safety recording. It receives already-encrypted recording chunks, stores metadata in SQLite, stores encrypted blobs on local disk, and exposes a token-scoped read-only emergency viewer.

The repository may also contain future-design and planning documents for a future iOS recorder, production key custody, browser-side decryption, and break-glass/dead-man-switch access. These documents are planning inputs unless the reviewed tree contains implementation code.

Core project boundaries:

- The private `/v1` API has no public user authentication and must not be exposed publicly.
- The current backend treats uploaded bytes as opaque ciphertext.
- The current backend must not be described as production-ready public infrastructure.
- Backend decryption, browser decryption, production key custody, break-glass access, user accounts, OAuth/JWT, push notifications, SMS, Messenger, and iOS recording implementation are future or out-of-scope items unless explicitly implemented in the reviewed tree.
- Future key custody, browser decryption, break-glass, and iOS recorder documents are design/planning guardrails, not shipped implementation.
- Do not treat documented future work as a current defect merely because it is not implemented.

## Source policy

Use authoritative sources only.

Prioritize repository evidence first:

- repository files in the reviewed tree
- source code, migrations, tests, workflows, Dockerfile, and documentation pinned to `<REVIEWED_COMMIT_SHA>`

Prioritize these external source families for current backend, security, CI/CD, Docker, SQLite, and web-security claims:

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
- `doc.traefik.io`, only for Traefik reverse-proxy examples and rate-limiting guidance

Prioritize these external source families for future iOS, Swift, Apple-platform, and App Store planning claims:

- `developer.apple.com/documentation/`
- `developer.apple.com/app-store/review/guidelines/`
- `developer.apple.com/design/human-interface-guidelines/`
- `developer.apple.com/videos/`, only for Apple WWDC sessions when API reference docs are insufficient
- `swift.org`
- `docs.swift.org`

Apple/Swift topics that should use Apple or Swift primary sources include:

- Swift language behaviour and concurrency
- Swift API design conventions
- AVFoundation / AVFAudio recording and media capture
- app lifecycle, interruptions, permissions, and background execution
- URLSession and background transfer behaviour
- BackgroundTasks
- CryptoKit and AES-GCM usage
- Keychain Services
- file protection and local data protection
- App Store Review Guidelines and privacy/safety requirements
- Human Interface Guidelines when reviewing future iOS user-facing flows

Avoid relying on:

- random blogs
- Stack Overflow
- social posts
- vendor marketing pages
- AI-generated summaries
- uncited claims
- stale Apple API examples when current Apple documentation is available

If a secondary source is used, explain why no primary source was sufficient.

## Citation requirements

Use portable citation keys only.

Do not use ChatGPT internal citation tokens such as renderer-only `filecite` / `cite` blocks or raw `turnXfileY`, `turnXviewY`, `turnXsearchY`, `turnXfetchY`, or `turnXopenY` references.

Use this citation style:

```markdown
Repository fact. [R-README] [R-CI]

External-source fact. [S-GITHUB-ACTIONS-SECURE]

Apple-platform planning fact. [S-APPLE-AVFOUNDATION] [S-SWIFT-DOCS]
```

At the end of the report, include Markdown reference definitions for every citation key:

```markdown
[R-README]: https://github.com/TheSilkky/safety-recorder/blob/<REVIEWED_COMMIT_SHA>/README.md
[S-GITHUB-ACTIONS-SECURE]: https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions
[S-APPLE-AVFOUNDATION]: https://developer.apple.com/documentation/avfoundation
[S-SWIFT-DOCS]: https://www.swift.org/documentation/
```

Repository citations must be pinned to `<REVIEWED_COMMIT_SHA>`, not `main`, unless the commit SHA is genuinely unavailable. If the SHA is unavailable, clearly mark the report as a draft and include a warning that repository URLs must be commit-pinned before publication.
If reviewing a release-prep branch, provide both the branch name and the exact commit SHA. The branch name is workflow context; the commit SHA is the citation target. Repository citations must still be pinned to `<REVIEWED_COMMIT_SHA>` so the report remains stable if the branch moves.


## Review scope

Review these repository areas when present in the reviewed tree:

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

Pay special attention to future-design and planning documents when present:

- `docs/key-custody.md`
- `docs/browser-decryption.md`
- `docs/break-glass-key-access.md`
- `docs/ios-local-recorder-prototype.md`
- any future `docs/ios*.md`, `docs/apple*.md`, or client-planning documents
- any future `ios/` client code, Swift package files, Xcode project files, entitlement files, or App Store metadata files if they exist in the reviewed tree

Technical focus areas:

1. Documentation consistency and public-readiness wording
2. Go backend structure and idiomatic implementation
3. HTTP API behavior and private/public route separation
4. Emergency token generation, hashing, storage, expiry, redaction, and viewer behavior
5. Logging, metrics, proxy examples, and sensitive data exposure
6. Upload handling, hash verification, immutable storage, upload limits, and stream-scoped chunk identity
7. SQLite migrations, foreign keys, WAL mode, schema migration tracking, and data integrity
8. ZIP bundle generation, manifest completeness, fail-closed behaviour, and path traversal handling
9. Crypto-adjacent simulator envelope:
   - AES-GCM use
   - nonce generation and uniqueness assumptions
   - associated data construction
   - key generation and simulator key handling
   - ciphertext-only backend boundary
10. Future-design boundary:
   - production key custody
   - browser/client-side decryption
   - break-glass / dead-man-switch access
   - trusted-contact recovery
   - server escrow or server-side decryption as explicit future modes only
11. Future iOS recorder planning:
   - whether the plan accurately distinguishes planning from implementation
   - AVFoundation / AVFAudio feasibility claims
   - foreground/background recording assumptions
   - iOS lifecycle, interruption, and permission constraints
   - URLSession/background transfer assumptions
   - local encrypted staging and file protection assumptions
   - Keychain and CryptoKit assumptions
   - App Store safety/privacy review considerations
   - mapping from local recorder state to current backend stream/chunk APIs
12. Deployment guidance:
   - Docker local/private exposure
   - WireGuard/private-network patterns
   - Traefik HTTPS emergency viewer exposure
   - deployment-edge rate limiting
   - proxy logging of token-bearing paths
   - no `/v1` public exposure
13. Docker/GHCR/GitHub Actions/supply-chain hygiene
14. Public issue/report safety

## Finding rules

For every finding, include:

- finding ID
- severity: Critical / High / Medium / Low / Informational
- confidence: High / Medium / Low
- current implementation vs future design
- affected files and functions, or affected planning documents
- repository evidence citation
- authoritative external citation, if applicable
- why it matters
- minimal actionable fix
- suggested GitHub issue title
- acceptance criteria

Do not inflate severity merely because a finding is security-adjacent. If a limitation is already documented as out of scope, classify it as a non-finding or future-work item unless there is a contradiction between docs and code.

Do not claim “missing” if a file or control exists. If a control exists but is incomplete, describe the narrower hardening task.

For future-design documents, classify issues as planning gaps, ambiguity, source-support gaps, or future-work risks unless the reviewed tree actually implements the feature.

For iOS planning documents, do not claim the iOS client exists unless `ios/` implementation files exist in the reviewed tree. Review whether the plan is plausible, source-supported, scoped, and explicit about platform limitations.

## Required report structure

Use this structure:

```markdown
# Technical Review of Safety Recorder <TARGET_RELEASE_OR_VERSION>

**Repository:** `TheSilkky/safety-recorder`
**Reviewed branch/ref:** `<REVIEWED_BRANCH_OR_REF>`
**Reviewed commit SHA:** `<REVIEWED_COMMIT_SHA>`
**Target release/version:** `<TARGET_RELEASE_OR_VERSION>`
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

## Current implementation vs future planning boundary

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

Do not include raw secrets, raw tokens, private deployment details, exploit payloads, user-safety data, raw keys, plaintext media, or private vulnerability details.

## Required disclaimer wording

Include this near the top:

```markdown
**AI-assisted review disclosure:** This report was generated with assistance from OpenAI ChatGPT Deep Research using <MODEL_NAME>, then reviewed and edited by the maintainer. It is not a formal security audit, penetration test, compliance certification, legal review, App Store review, or production-readiness endorsement. Findings should be verified against the reviewed commit, cited sources, and current project scope before being relied on.

**Public-disclosure note:** This report is intended for public project documentation. It intentionally avoids raw tokens, secrets, private deployment details, exploit payloads, user-safety data, raw keys, plaintext media, and private vulnerability details.
```

## Quality gates before returning the draft

Before returning the report, check and state whether the draft satisfies:

- no ChatGPT internal citation tokens
- no `blob/main` repository URLs when a reviewed commit SHA was supplied
- all repository citations are pinned to `<REVIEWED_COMMIT_SHA>`
- reviewed branch/ref is named separately from the commit SHA when supplied
- every bracket citation key has a reference definition
- every reference definition is used or intentionally retained
- no raw tokens, secrets, private deployment details, exploit payloads, user-safety data, raw keys, plaintext media, or private vulnerability details
- no production-readiness claim
- no formal audit/certification claim
- no claim that Deep Research executed tests or containers unless actual execution evidence is from Codex/CI/local logs and is attributed correctly
- no legal/App Store approval claim
- current implementation and future design/planning are clearly separated
- future iOS/key-custody/browser-decryption planning documents are not described as implemented features unless implementation exists
- no “Claims check” section
- no “Verify before sending” section

## Output

Return the full Markdown report.

Then include a short Phase 2 handoff summary:

```text
Phase 2 handoff:
- validation evidence supplied:
- commands not independently verified:
- reviewed branch/ref:
- reviewed commit SHA:
- target release/version:
- highest-confidence findings:
- future-planning claims that need maintainer verification:
- iOS/Swift/Apple-platform claims that need source verification:
- citations that may need cleanup:
- possible duplicate or existing issues to check:
```
