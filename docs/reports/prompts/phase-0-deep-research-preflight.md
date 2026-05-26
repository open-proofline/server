# Phase 0 Deep Research Prompt: Load Report Instructions And Plan Research

Use this prompt in ChatGPT Deep Research before running the Phase 1 technical review report.

Do **not** produce the technical review report in this Phase 0 step.

This prompt exists because the Phase 1 report prompt is long and strict. Phase 0 should load the current Phase 1 prompt from the repository, restate the governing rules, and produce a research plan for maintainer approval before the actual report is generated.

## Repository

```text
TheSilkky/safety-recorder
```

## Inputs

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

Phase 1 prompt path:

```text
docs/reports/prompts/phase-1-deep-research-technical-review.md
```

Expected Phase 2 validation prompt path:

```text
codex/prompts/95-validate-deep-research-report.md
```

Target report path, if known:

```text
docs/reports/<YYYY-MM-DD>-safety-recorder-<TARGET_RELEASE_OR_VERSION>-technical-review.md
```

Model / tool disclosure:

```text
OpenAI ChatGPT Deep Research using <MODEL_NAME>
```

## Goal

Load and internalize the current Phase 1 prompt from the reviewed repository branch/ref or reviewed commit, then produce a research plan that follows that prompt.

Phase 0 should make sure the actual Phase 1 report run will follow:

- the correct repository scope
- the correct reviewed branch/ref and commit SHA
- the correct target release/version
- the Phase 1 Source Registry requirements
- the Phase 1 citation requirements
- the Phase 1 validation-evidence policy
- the Phase 1 public-safety restrictions
- the Phase 1 implementation-vs-future-planning boundary
- the Phase 2 handoff expectations

## Required preflight steps

1. Open and read:

   ```text
   docs/reports/prompts/phase-1-deep-research-technical-review.md
   ```

   from the reviewed branch/ref or reviewed commit.

2. Confirm the Phase 1 prompt path and source used.

3. Summarize the governing Phase 1 requirements in your own words.

4. Identify the required repository files and directories to inspect.

5. Identify future-design and planning documents that must be reviewed when present, including but not limited to:

   ```text
   docs/key-custody.md
   docs/browser-decryption.md
   docs/break-glass-key-access.md
   docs/ios-local-recorder-prototype.md
   ```

6. Identify the authoritative external source categories required by the Phase 1 prompt.

7. Identify which Apple/iOS/Swift claims require Apple Developer or Swift primary documentation.

8. Identify the required Source Registry sections and what each section must contain.

9. Identify the citation requirements, including repository citation pinning to `<REVIEWED_COMMIT_SHA>`.

10. Identify the validation evidence policy, especially that Deep Research cannot run tests, containers, Docker builds, local shell commands, or simulator smoke tests.

11. Identify public-safety restrictions, including prohibited raw tokens, secrets, request bodies, uploaded bytes, plaintext, raw keys, private deployment details, exploit payloads, and user-safety data.

12. Identify common false-positive patterns the Phase 1 report should avoid.

13. Produce a proposed research plan.

14. Stop and wait for maintainer approval before running the actual Phase 1 report.

## Important constraints

Do **not** produce the technical review report in Phase 0.

Do **not** treat this preflight as the Phase 1 report.

Do **not** claim tests, containers, Docker builds, local shell commands, GitHub repository settings, or simulator smoke tests were run unless supplied validation evidence proves that.

Do **not** use ChatGPT-rendered citations as final public Markdown citations.

Do **not** rely on repository-only evidence for external standards, platform, security, CI/CD, Docker, SQLite, dependency, license, Apple/iOS, or Swift claims when the Phase 1 prompt requires authoritative external sources.

Do **not** include raw tokens, secrets, request bodies, uploaded bytes, plaintext, raw keys, private deployment details, exploit payloads, or user-safety data.

Do **not** claim production readiness.

Do **not** claim formal security audit, penetration test, compliance certification, legal review, App Store review, or production-readiness endorsement.

Do **not** describe future key custody, browser decryption, break-glass access, or iOS recorder planning documents as implemented features unless the reviewed tree contains implementation code.

## Expected source plan

The research plan should identify sources in these groups.

### Repository sources

At minimum, plan to inspect current files such as:

```text
README.md
SECURITY.md
CHANGELOG.md
AGENTS.md
docs/
codex/
.github/
server/
server/migrations/
server/Dockerfile
server/.dockerignore
```

Also include relevant future-planning documents if present:

```text
docs/key-custody.md
docs/browser-decryption.md
docs/break-glass-key-access.md
docs/ios-local-recorder-prototype.md
```

### External authoritative sources

Use the Phase 1 prompt as the source of truth for the required external source categories.

The plan should identify which claims require sources from:

```text
go.dev
pkg.go.dev
csrc.nist.gov
nvlpubs.nist.gov
owasp.org
cheatsheetseries.owasp.org
docs.github.com
docs.docker.com
sqlite.org
rfc-editor.org
datatracker.ietf.org
doc.traefik.io
developer.apple.com
swift.org
docs.swift.org
```

Do not cite broad homepages when specific documentation pages are available.

### Validation evidence

Identify validation evidence that was supplied or missing, such as:

```text
GitHub Actions run URLs
local command output
Codex command output
uploaded validation summaries
gofmt output
go test output
go vet output
Docker build output
simulator smoke test output
attestation verification output
```

If validation evidence is not supplied, the Phase 1 report must say the command was not independently verified.

## Source Registry planning

The Phase 0 plan must explain how Phase 1 will build the Source Registry before writing findings.

The plan should include the required registry sections from Phase 1:

```markdown
## Source Registry

### Repository sources inspected

### External authoritative sources consulted

### Validation and execution evidence

### Sources, checks, and commands not available or not executed

### Generated artifacts and report outputs
```

The plan should confirm that every registry entry will include:

- source ID / citation key
- source type
- location
- commit/ref/date
- purpose in the review
- status
- limitations
- related finding IDs or report sections where applicable

## Citation planning

The Phase 0 plan must restate that final public citations are handled through portable citation keys, not ChatGPT UI citation tokens.

Portable key families:

```text
R-* for repository sources pinned to <REVIEWED_COMMIT_SHA>
S-* for external authoritative sources
I-* for issue, PR, or report-follow-up references
V-* for validation evidence
```

The Phase 0 plan must state that repository URLs must be pinned to:

```text
<REVIEWED_COMMIT_SHA>
```

not `main`, not `develop`, and not a moving release branch.

If ChatGPT-rendered citations appear in the Phase 1 draft, the plan must instruct Phase 2 Codex validation to convert or replace them using the Source Registry.

## Branch and issue-scope planning

If Phase 1 may suggest follow-up issues, the Phase 0 plan must restate the Phase 1 branch-scope rules.

The plan should require that follow-up issue suggestions distinguish:

- release blockers for the reviewed branch
- non-blocking follow-ups after merge
- findings that require revalidation on `main` or `develop`
- planning-only findings
- sensitive items that should not become public issues

The plan should confirm that any Phase 2-generated public issue drafts must include:

```text
Priority
Type
Labels
Branch scope
```

## Report scope planning

The Phase 0 plan should map the Phase 1 report to these broad areas:

1. Documentation consistency and public-readiness wording
2. Go backend structure and idiomatic implementation
3. HTTP API behavior and private/public route separation
4. Emergency token generation, hashing, storage, expiry, redaction, and viewer behavior
5. Logging, metrics, proxy examples, and sensitive data exposure
6. Upload handling, hash verification, immutable storage, upload limits, and stream-scoped chunk identity
7. SQLite migrations, WAL mode, foreign keys, schema migration tracking, and data integrity
8. ZIP bundle generation, manifest completeness, fail-closed behavior, and path traversal handling
9. Crypto-adjacent simulator envelope
10. Future key custody, browser decryption, break-glass, and trusted-contact planning
11. Future iOS recorder planning and Apple-platform assumptions
12. Deployment guidance, Traefik examples, WireGuard/private boundary, and rate limiting
13. Docker/GHCR/GitHub Actions/supply-chain hygiene
14. Public issue/report safety

## Output format

Return only a Phase 0 preflight response with this structure:

```markdown
# Phase 0 Deep Research Preflight

## Loaded prompt

- Repository:
- Reviewed branch/ref:
- Reviewed commit SHA:
- Target release/version:
- Phase 1 prompt path:
- Phase 1 prompt source used:
- Target report path:

## Summary of governing Phase 1 rules

## Repository source plan

## External authoritative source plan

## Validation evidence plan

## Source Registry plan

## Citation plan

## Branch and issue-scope plan

## Public-safety restrictions

## Current implementation vs future planning boundary

## Expected report structure

## Risks, ambiguities, or maintainer decisions needed before Phase 1

## Proposed Phase 1 execution plan

## Stop point

Ready for maintainer approval before running Phase 1.
```

Do not continue into Phase 1 until the maintainer explicitly approves the plan.

## Follow-up instruction after maintainer approval

After Phase 0 returns the plan, the maintainer may approve it with a short instruction such as:

```text
Approved. Run Phase 1 using the loaded `docs/reports/prompts/phase-1-deep-research-technical-review.md` as the governing prompt.

Reviewed branch/ref: `<REVIEWED_BRANCH_OR_REF>`
Reviewed commit SHA: `<REVIEWED_COMMIT_SHA>`
Target release/version: `<TARGET_RELEASE_OR_VERSION>`
Review date: `<YYYY-MM-DD>`

Follow the Source Registry, citation, validation-evidence, branch-scope, and public-safety rules from the loaded Phase 1 prompt.
```

Only after that approval should Deep Research produce the Phase 1 technical review report.
