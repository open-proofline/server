# Technical Review of Safety Recorder v0.5.0-rc.1

**Repository:** `TheSilkky/safety-recorder`
**Reviewed branch or ref:** `release/v0.5.0-prep`
**Reviewed commit SHA:** `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289`
**Target release/version:** `v0.5.0-rc.1`
**Review date:** 2026-05-25
**Phase 2 validation date:** 2026-05-26
**Report status:** Final public report after Codex Phase 2 validation. Findings were converted into branch-scoped local draft issues only.

**Citation format note:** This report uses portable citation keys only. Repository citations are pinned to reviewed commit `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289`; external citations resolve to canonical documentation URLs. No ChatGPT-internal citation tokens are used.

**AI-assisted review disclosure:** This report began as an OpenAI ChatGPT Deep Research draft and was validated, corrected, and public-hardened with Codex. It is not a formal security audit, penetration test, compliance certification, legal review, App Store review, or production-readiness endorsement.

**Public-disclosure note:** This report is intended for public project documentation. It intentionally avoids raw tokens, secrets, private deployment details, exploit payloads, raw keys, plaintext media, and user-safety data.

## Executive Summary

The reviewed tree at commit `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289` has no critical or high-severity static blockers for `v0.5.0-rc.1`. The backend remains tightly scoped: private `/v1` routes are documented as unauthenticated private/admin routes, public emergency viewer routes are separate and read-only, uploaded media bytes are treated as opaque ciphertext, completed evidence bundles are encrypted ZIP bundles rather than playable exports, and future key-custody/browser-decryption/break-glass work is clearly separated from the current implementation. [R-README] [R-SECURITY-MODEL] [R-THREAT-MODEL] [R-ROUTES] [R-BUNDLES]

The review retained three low-severity follow-up findings. First, SQLite WAL mode is requested with `PRAGMA journal_mode = WAL`, but the startup path does not read back the pragma result. SQLite documents that this pragma returns the resulting mode and can leave the database in its prior journal mode if WAL cannot be enabled. [R-DB] [S-SQLITE-WAL]

Second, the CI/publish workflow is already strong on full-SHA action pinning and narrow package-write permissions, but it does not yet generate artifact attestations for published binaries or container images. GitHub documents artifact attestations as a provenance mechanism for build artifacts and container images. [R-CI] [S-GITHUB-ATTESTATIONS]

Third, the Docker digest-refresh guidance in `docs/development.md` still tells maintainers to inspect `alpine:3.22`, while the reviewed Dockerfile uses `alpine:3.23@sha256:...`. This is documentation drift rather than an application defect, but it is the kind of release-process mismatch that can send a maintainer to the wrong manifest during a digest refresh. [R-DEVELOPMENT] [R-DOCKERFILE] [S-DOCKER-DIGESTS]

The release verdict is therefore: **release candidate acceptable with non-blocking follow-up drafts**. This report does not claim production readiness. The private `/v1` API still must not be exposed publicly as-is. [R-README] [R-DEPLOYMENT] [R-SECURITY]

## Scope And Methodology

This Phase 2 pass validated the Deep Research draft against repository files, the reviewed commit SHA, current source-of-truth documentation, selected authoritative external sources, and available validation evidence. The reviewed branch name is workflow context only; repository citations are pinned to the reviewed commit so the report remains stable if `release/v0.5.0-prep` moves. [R-README] [R-DEVELOPMENT]

The current checked-out branch during Phase 2 was `release/v0.5.0-prep`, but `HEAD` had advanced to `1d31f19817fc846dd1e9ac80fdbbf0d4bf178142`. The commits after the reviewed SHA changed prompt/workflow documentation and ignore metadata, not the backend implementation being assessed here. Findings and repository citations remain scoped to `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289`.

Phase 2 removed sandbox-only draft links, converted the report to portable citation keys, corrected validation claims, and created local branch-scoped draft issues under `.backlog-drafts/2026-05-25/release-v0.5.0-prep/`. No GitHub issues were created.

## Source Registry

### Repository Sources Inspected

| Key | Source type | Location | Purpose | Status and limitations |
|---|---|---|---|---|
| R-README | Repository file | `README.md` at reviewed commit | Project scope, release warnings, current feature claims | Inspected; does not prove deployment configuration. |
| R-SECURITY | Repository file | `SECURITY.md` at reviewed commit | Supported version and vulnerability reporting posture | Inspected; does not prove triage process. |
| R-CHANGELOG | Repository file | `CHANGELOG.md` at reviewed commit | Release notes for `v0.5.0-rc.1` | Inspected; release notes are maintainer documentation. |
| R-DOCS-README | Repository file | `docs/README.md` at reviewed commit | Docs index and current/future scope boundary | Inspected. |
| R-DEVELOPMENT | Repository file | `docs/development.md` at reviewed commit | Release, CI, branch protection, Docker digest review process | Inspected; hosted GitHub settings were not certified by this file alone. |
| R-API | Repository file | `docs/api.md` at reviewed commit | Route surface, stream identity, bundle format, response headers | Inspected. |
| R-SECURITY-MODEL | Repository file | `docs/security-model.md` at reviewed commit | Current security controls and known gaps | Inspected. |
| R-THREAT-MODEL | Repository file | `docs/threat-model.md` at reviewed commit | Assets, trust boundaries, controls, and limitations | Inspected. |
| R-ENCRYPTION | Repository file | `docs/encryption.md` at reviewed commit | Simulator envelope and key-file behavior | Inspected; current implementation remains simulator/test oriented. |
| R-DEPLOYMENT | Repository file | `docs/deployment.md` at reviewed commit | Private/public exposure warnings, Docker, proxy, HSTS, rate limiting | Inspected; examples are not deployed infrastructure. |
| R-KEY-CUSTODY | Repository file | `docs/key-custody.md` at reviewed commit | Future key custody planning | Inspected as planning-only evidence. |
| R-BROWSER-DECRYPTION | Repository file | `docs/browser-decryption.md` at reviewed commit | Future browser decryption planning | Inspected as planning-only evidence. |
| R-BREAK-GLASS | Repository file | `docs/break-glass-key-access.md` at reviewed commit | Future break-glass/dead-man-switch planning | Inspected as planning-only evidence. |
| R-IOS-PROTOTYPE | Repository file | `docs/ios-local-recorder-prototype.md` at reviewed commit | Future iOS prototype planning | Inspected as planning-only evidence; no iOS implementation files were present. |
| R-DB | Repository file | `server/internal/db/db.go` at reviewed commit | SQLite open, PRAGMAs, migrations | Inspected; supports SR-TR-001. |
| R-STORAGE | Repository file | `server/internal/storage/storage.go` at reviewed commit | Immutable chunk storage and safe paths | Inspected. |
| R-ROUTES | Repository file | `server/internal/httpapi/routes.go` at reviewed commit | Private/public mux separation | Inspected. |
| R-MIDDLEWARE | Repository file | `server/internal/httpapi/middleware.go` at reviewed commit | Request logging and token-path redaction | Inspected. |
| R-RESPONSE | Repository file | `server/internal/httpapi/response.go` at reviewed commit | Browser and JSON response headers | Inspected. |
| R-EMERGENCY | Repository file | `server/internal/httpapi/emergency.go` at reviewed commit | Emergency token handling and viewer response privacy | Inspected. |
| R-BUNDLES | Repository file | `server/internal/httpapi/bundles.go` at reviewed commit | Stream and incident bundle routes | Inspected. |
| R-BUNDLE-ZIP | Repository file | `server/internal/httpapi/bundle_zip.go` at reviewed commit | ZIP entry names and download headers | Inspected. |
| R-BUNDLE-MANIFEST | Repository file | `server/internal/httpapi/bundle_manifest.go` at reviewed commit | Bundle manifest encryption hints | Inspected. |
| R-ENVELOPE-IMPL | Repository file | `server/internal/envelope/envelope.go` at reviewed commit | Simulator/test encryption envelope | Inspected. |
| R-CI | Repository file | `.github/workflows/ci.yml` at reviewed commit | CI jobs, action pinning, publish permissions, absence of attestation steps | Inspected; hosted run results are separate validation evidence. |
| R-DEPENDABOT | Repository file | `.github/dependabot.yml` at reviewed commit | Dependency update coverage | Inspected. |
| R-DOCKERFILE | Repository file | `server/Dockerfile` at reviewed commit | Base image pinning and runtime shape | Inspected. |
| R-DOCKERIGNORE | Repository file | `server/.dockerignore` at reviewed commit | Build-context hygiene | Inspected. |
| R-GOMOD | Repository file | `server/go.mod` at reviewed commit | Direct Go dependency surface | Inspected. |

### External Authoritative Sources Consulted

| Key | Source type | Location | Purpose | Status and limitations |
|---|---|---|---|---|
| S-SQLITE-WAL | External authoritative source | SQLite WAL documentation | WAL activation, returned journal mode, same-host constraints | Consulted; supports SR-TR-001. |
| S-GITHUB-ACTIONS-SECURE | External authoritative source | GitHub Actions secure-use reference | Immutable action references and workflow hardening context | Consulted. |
| S-GITHUB-ATTESTATIONS | External authoritative source | GitHub artifact attestation documentation | Provenance permissions and attestation workflow guidance | Consulted; supports SR-TR-002. |
| S-DOCKER-DIGESTS | External authoritative source | Docker image digest documentation | Tag-versus-digest semantics | Consulted; supports SR-TR-003. |
| S-GOVULNCHECK | External authoritative source | Go govulncheck documentation | Interpreting dependency vulnerability scan evidence | Consulted. |
| S-GO-SQLITE3 | External authoritative source | `pkg.go.dev` page for `github.com/mattn/go-sqlite3@v1.14.44` | Direct dependency metadata | Consulted; not a vulnerability database by itself. |

### Validation And Execution Evidence

| Key | Evidence | Date | Purpose | Status and limitations |
|---|---|---|---|---|
| V-CI-RUN | GitHub Actions run `26398156129` for reviewed SHA `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289` | 2026-05-25 | Confirms CI completed successfully for the reviewed commit: Go tests, Build Linux binary, Build Docker image | Supplied through GitHub CLI during Phase 2; Docker publish was skipped because the run was on the release branch, not `main` or a `v*` tag. |
| V-PHASE2-COMMANDS | Local Codex Phase 2 commands on current checkout | 2026-05-26 | Confirmed `govulncheck` reported `No vulnerabilities found`; report safety/citation checks were run after editing | Current checkout had moved past the reviewed SHA with prompt/ignore changes. This is supporting evidence, not a substitute for commit-pinned CI. |

### Sources, Checks, And Commands Not Available Or Not Executed

No iOS or Swift implementation files existed, so Apple-platform claims were treated as planning-only and no Apple implementation source was required. No live deployment, reverse proxy, firewall, DNS, log sink, or host backup configuration was inspected. No GitHub repository settings were modified. No GitHub issues were created because the requested issue handling mode was `drafts_only`.

### Generated Artifacts And Report Outputs

| Artifact | Purpose | Status |
|---|---|---|
| `docs/reports/2026-05-25-safety-recorder-v0.5.0-rc.1-technical-review.md` | Cleaned public report | Generated. |
| `.backlog-drafts/2026-05-25/release-v0.5.0-prep/` | Branch-scoped local follow-up issue drafts | Generated; ignored local drafts, not public GitHub issues. |
| `docs/reports/README.md` | Reports index | Updated to list this report. |

## Current Implementation Vs Future Planning

The current backend does not implement an iOS recorder, production key custody, browser-side decryption, break-glass key access, server-side decryption, raw server-held media keys, push notifications, SMS/Messenger integration, OAuth, JWT, user accounts, or a public admin dashboard. The reviewed documentation presents these as future design or out-of-scope items, not as shipped behavior. [R-README] [R-DOCS-README] [R-KEY-CUSTODY] [R-BROWSER-DECRYPTION] [R-BREAK-GLASS] [R-IOS-PROTOTYPE]

The implemented server surface is consistent with the documented security boundary. Private `/v1` routes are registered on the private mux, public emergency routes are registered on the public mux, emergency viewer routes are read-only, and bundle routes generate server-controlled ZIP entries from database/storage metadata rather than accepting client-provided stored paths. [R-ROUTES] [R-BUNDLES] [R-BUNDLE-ZIP] [R-BUNDLE-MANIFEST]

The browser-facing posture also matches the documented model: public emergency responses apply browser security headers, token-protected emergency pages/data/errors/downloads use `Cache-Control: no-store`, and HSTS is intentionally left to HTTPS deployment edges rather than being sent by the local HTTP app. [R-RESPONSE] [R-EMERGENCY] [R-SECURITY-MODEL] [R-DEPLOYMENT]

## Findings

| ID | Severity | Classification | Finding | Suggested issue title |
|---|---|---|---|---|
| SR-TR-001 | Low | `follow-up-after-merge` | Verify SQLite WAL activation result instead of assuming the requested mode took effect. | Verify SQLite WAL mode at startup and fail closed when WAL cannot be enabled |
| SR-TR-002 | Low | `follow-up-after-merge` | Add provenance/artifact attestations for published release artifacts and container images. | Add GitHub artifact attestations for release binaries and GHCR images |
| SR-TR-003 | Low | `follow-up-after-merge` | Refresh Docker digest-review documentation so it tracks the actual Alpine tag family used by `server/Dockerfile`. | Align Docker digest refresh documentation with current runtime base image tag family |

### SR-TR-001: Verify SQLite WAL Activation Result

**Severity:** Low.

**Finding.** Startup executes `PRAGMA journal_mode = WAL` with `ExecContext`, but it does not read the returned journal mode. SQLite documents that `PRAGMA journal_mode=WAL` returns the resulting mode; on success the returned string is `wal`, and if conversion cannot be completed, the mode remains unchanged and the prior mode is returned. [R-DB] [S-SQLITE-WAL]

**Why it matters.** The project documents WAL as part of its local SQLite operating model. If a deployment uses storage or VFS behavior where WAL cannot be enabled, the current code may proceed without noticing that the database is not in the expected mode. This is not a confidentiality break and is not a release blocker, but it is a small correctness hardening opportunity. [R-SECURITY-MODEL] [R-THREAT-MODEL] [S-SQLITE-WAL]

**Minimal fix.** Use `QueryRowContext` for the journal-mode pragma, scan the returned string, and fail closed or emit a prominent startup error if the result is not `wal`. Add a unit test or integration-style test around the helper that verifies non-`wal` results are handled explicitly.

### SR-TR-002: Add Artifact Attestations For Release Outputs

**Severity:** Low.

**Finding.** The reviewed CI workflow pins actions to full 40-character SHAs, keeps workflow-level permissions read-only, and grants `packages: write` only to the Docker publish job. That is good baseline hygiene. The workflow does not, however, generate artifact attestations for release binaries or GHCR images. GitHub documents artifact attestations as a way to establish build provenance for binaries and container images, with `id-token: write` and `attestations: write` permissions for attestation jobs. [R-CI] [S-GITHUB-ACTIONS-SECURE] [S-GITHUB-ATTESTATIONS]

**Why it matters.** This project already publishes build artifacts and container images through GitHub Actions. Attestations would improve provenance for release consumers without changing application behavior. This is a supply-chain improvement, not a missing runtime control. [R-CI] [R-DEVELOPMENT] [S-GITHUB-ATTESTATIONS]

**Minimal fix.** Add attestation steps only to the jobs that produce release binaries and published images. Keep default workflow permissions read-only, grant `id-token: write` and `attestations: write` only where needed, and document verification steps in release guidance.

### SR-TR-003: Align Docker Digest Refresh Documentation

**Severity:** Low.

**Finding.** The reviewed Dockerfile pins the runtime image as `alpine:3.23@sha256:...`, but `docs/development.md` still shows a digest-refresh command for `docker.io/library/alpine:3.22`. Docker documents that tags are mutable while digests identify exact image content, so humans refreshing digests should inspect the tag family actually used by the Dockerfile. [R-DEVELOPMENT] [R-DOCKERFILE] [S-DOCKER-DIGESTS]

**Why it matters.** This is documentation/process drift. It does not alter the built runtime image, and the Dockerfile itself is pinned by digest, but stale guidance can lead maintainers to inspect or refresh the wrong runtime image family.

**Minimal fix.** Update the development guide's digest-refresh example from `alpine:3.22` to `alpine:3.23`, and consider keeping Dockerfile tag-family changes and digest-refresh documentation in the same pull request going forward.

## Non-Findings And Limitations

The report does not treat missing production-hardening features as undisclosed defects when the repository already marks them out of scope. In particular, the absence of a production iOS app, public `/v1` authentication, user accounts, built-in TLS, app-level rate limiting, production key custody, browser decryption, break-glass access, retention automation, or playable media export is documented and should not be framed as a hidden bug in this release candidate. [R-README] [R-SECURITY-MODEL] [R-THREAT-MODEL]

The direct Go dependency surface remains small: `server/go.mod` lists `github.com/mattn/go-sqlite3 v1.14.44`. Phase 2 ran `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` and received `No vulnerabilities found`. That is useful vulnerability-scan evidence, but it is not a guarantee that future advisory databases, transitive environment details, or container-image scans will remain clean. [R-GOMOD] [S-GOVULNCHECK] [S-GO-SQLITE3]

This report did not inspect live deployment infrastructure, firewall rules, reverse-proxy logs, DNS, TLS certificates, backups, restore processes, host disk encryption, or real user workflows. It also did not modify GitHub repository settings or create public GitHub issues.

## Branch-Scoped Draft Issue Mapping

Because the issue handling mode was `drafts_only`, Phase 2 created local branch-scoped draft issues only. These drafts are scoped to:

- Current branch: `release/v0.5.0-prep`
- Current HEAD during Phase 2: `1d31f19817fc846dd1e9ac80fdbbf0d4bf178142`
- Reviewed branch/ref: `release/v0.5.0-prep`
- Reviewed commit SHA: `5b5a57354d6fcdbdc1ef1f440372c04b8bba2289`
- Target release/version: `v0.5.0-rc.1`

Draft directory:

```text
.backlog-drafts/2026-05-25/release-v0.5.0-prep/
```

No public GitHub issues were created. Revalidate each draft against the eventual target branch before creating or closing public issues if the branch has moved.

## Release Verdict

For the reviewed commit, this report does not identify a static-analysis reason to block `v0.5.0-rc.1`. The recommended outcome is **release candidate acceptable with follow-up actions**. The WAL verification improvement is the highest-value code follow-up, the artifact attestation work is a release-engineering improvement, and the Docker digest-refresh wording is a small documentation cleanup. [R-CHANGELOG] [R-DEVELOPMENT]

## Reference Definitions

[R-README]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/README.md
[R-SECURITY]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/SECURITY.md
[R-CHANGELOG]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/CHANGELOG.md
[R-DOCS-README]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/README.md
[R-DEVELOPMENT]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/development.md
[R-API]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/api.md
[R-SECURITY-MODEL]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/security-model.md
[R-THREAT-MODEL]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/threat-model.md
[R-ENCRYPTION]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/encryption.md
[R-DEPLOYMENT]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/deployment.md
[R-KEY-CUSTODY]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/key-custody.md
[R-BROWSER-DECRYPTION]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/browser-decryption.md
[R-BREAK-GLASS]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/break-glass-key-access.md
[R-IOS-PROTOTYPE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/docs/ios-local-recorder-prototype.md
[R-DB]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/db/db.go
[R-STORAGE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/storage/storage.go
[R-ROUTES]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/routes.go
[R-MIDDLEWARE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/middleware.go
[R-RESPONSE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/response.go
[R-EMERGENCY]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/emergency.go
[R-BUNDLES]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/bundles.go
[R-BUNDLE-ZIP]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/bundle_zip.go
[R-BUNDLE-MANIFEST]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/httpapi/bundle_manifest.go
[R-ENVELOPE-IMPL]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/internal/envelope/envelope.go
[R-CI]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/.github/workflows/ci.yml
[R-DEPENDABOT]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/.github/dependabot.yml
[R-DOCKERFILE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/Dockerfile
[R-DOCKERIGNORE]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/.dockerignore
[R-GOMOD]: https://github.com/TheSilkky/safety-recorder/blob/5b5a57354d6fcdbdc1ef1f440372c04b8bba2289/server/go.mod
[S-SQLITE-WAL]: https://www.sqlite.org/wal.html
[S-GITHUB-ACTIONS-SECURE]: https://docs.github.com/en/actions/reference/security/secure-use
[S-GITHUB-ATTESTATIONS]: https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations
[S-DOCKER-DIGESTS]: https://docs.docker.com/dhi/core-concepts/digests/
[S-GOVULNCHECK]: https://go.dev/doc/tutorial/govulncheck
[S-GO-SQLITE3]: https://pkg.go.dev/github.com/mattn/go-sqlite3@v1.14.44
