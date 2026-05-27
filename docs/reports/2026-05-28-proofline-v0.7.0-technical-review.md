# Technical Review of Proofline v0.7.0

**Repository:** `open-proofline/server`
**Reviewed branch/ref:** `main`
**Reviewed commit SHA:** `12e97543953ff1ba938c128a6afec73e9643acce`
**Target release/version:** `v0.7.0`
**Review date:** 2026-05-28
**Phase 2 validation date:** 2026-05-28
**Report status:** Final public report after Codex Phase 2 validation. Follow-up items were mapped to local branch-scoped draft issues only.

**Citation format note:** This report uses portable citation keys only. Repository citations are pinned to reviewed commit `12e97543953ff1ba938c128a6afec73e9643acce`; external citations resolve to canonical documentation URLs. No ChatGPT-internal citation tokens are used.

**AI-assisted review disclosure:** This report began as an OpenAI ChatGPT Deep Research draft using GPT-5.5 Thinking-Heavy, then was validated, corrected, and public-hardened with Codex. This report is not a formal security audit, penetration test, compliance certification, legal review, App Store review, Play Store review, or production-readiness endorsement.

**Public-disclosure note:** This report is intended for public project documentation. It intentionally avoids raw tokens, secrets, private deployment details, exploit payloads, raw keys, plaintext media, and user-safety data.

## Executive Summary

Proofline `v0.7.0` is a repository and release-layout milestone for the Go server backend. The reviewed commit moves the module and backend source tree to the repository root as `github.com/open-proofline/server`, normalizes release binary and GHCR naming around `open-proofline/server`, and preserves documented compatibility identifiers where protocol or data-layout migrations have not been explicitly designed. [R-README] [R-CHANGELOG] [R-DEPLOYMENT] [R-ENCRYPTION]

The reviewed backend remains deliberately scoped: it receives already-encrypted recording chunks, stores metadata in SQLite, stores encrypted blobs on local disk, groups uploaded chunks into media streams, and exposes a token-scoped read-only public incident viewer on a listener group separate from private `/v1` write/admin routes. The private `/v1` API has no public user authentication and must stay behind localhost, LAN, WireGuard, a firewall, or a strict reverse proxy. [R-README] [R-SECURITY] [R-API] [R-SECURITY-MODEL] [R-ROUTES]

Phase 2 validation did not identify a critical or high-severity static blocker in the reviewed release commit. The core security boundary remains intact: the backend treats uploaded bytes as opaque ciphertext, does not store raw media keys, does not decrypt chunks, and emits encrypted ZIP evidence bundles rather than decrypted or playable media exports. Future incident modes, key custody, browser decryption, break-glass access, and iOS recorder planning are documented as future design, not implemented behavior. [R-ENCRYPTION] [R-KEY-CUSTODY] [R-BROWSER-DECRYPTION] [R-BREAK-GLASS] [R-IOS-PROTOTYPE]

The main caveat is operational. Public internet exposure still requires careful deployment controls, especially keeping `/v1` private, applying TLS and edge rate limiting, and redacting token-bearing viewer paths in reverse-proxy logs. Those are documented constraints, not hidden implementation claims. The narrow code-level follow-up from this review is to sanitize internal filesystem error logging so local paths and evidence-layout details are not written to server logs when storage or bundle operations fail. [R-DEPLOYMENT] [R-SECURITY-MODEL] [R-THREAT-MODEL] [R-RESPONSE] [R-BUNDLES]

Recent GitHub Actions evidence for the reviewed commit shows successful Go tests, `go vet`, `govulncheck`, Linux binary build, Docker image build, and coverage artifact upload. The same run skipped tag-only publish and attestation jobs because it was a `main` branch run, not a tag workflow run. This report did not download artifacts, replay tests, run the simulator, or inspect a live deployment. [V-RUN] [V-JOBS] [V-ARTIFACTS]

## Source Registry

### Repository sources inspected

All repository entries were inspected at reviewed commit `12e97543953ff1ba938c128a6afec73e9643acce`.

| Key | Source type | Location | Commit/ref/date | Purpose | Status | Limitations | Related findings/sections |
|---|---|---|---|---|---|---|---|
| R-README | Repository file | `README.md` | Reviewed commit | Project scope, warning language, current features, planned repository split | Reviewed | Documentation, not runtime proof | Executive Summary; Non-Findings |
| R-SECURITY | Repository file | `SECURITY.md` | Reviewed commit | Supported versions, vulnerability-reporting rules, public-safety restrictions | Reviewed | Process guidance only | Non-Findings |
| R-CHANGELOG | Repository file | `CHANGELOG.md` | Reviewed commit | `v0.7.0` release delta | Reviewed | Maintainer-authored release notes | Executive Summary |
| R-AGENTS | Repository file | `AGENTS.md` | Reviewed commit | Project rules for route separation, logging, encryption, and backlog handling | Reviewed | Development guardrails only | Scope And Method |
| R-DOCS-README | Repository file | `docs/README.md` | Reviewed commit | Documentation index and current/future scope boundary | Reviewed | Documentation, not runtime proof | Current Implementation Summary |
| R-API | Repository file | `docs/api.md` | Reviewed commit | Current HTTP routes, upload semantics, token routes, bundle format | Reviewed | Does not prove live deployment | Current Implementation Summary; Findings |
| R-DEPLOYMENT | Repository file | `docs/deployment.md` | Reviewed commit | Bind-address guidance, public viewer exposure, Traefik and rate-limit examples | Reviewed | Examples are illustrative, not deployed configs | F-002 |
| R-SECURITY-MODEL | Repository file | `docs/security-model.md` | Reviewed commit | Listener boundary, token handling, storage controls, headers, known gaps | Reviewed | Threat-oriented documentation only | Executive Summary; F-002 |
| R-THREAT-MODEL | Repository file | `docs/threat-model.md` | Reviewed commit | Assets, trust boundaries, current controls, known limitations, next steps | Reviewed | No live attacker testing | F-002; Non-Findings |
| R-ENCRYPTION | Repository file | `docs/encryption.md` | Reviewed commit | Simulator envelope and ciphertext-only backend posture | Reviewed | Does not execute crypto code | Executive Summary; Non-Findings |
| R-INCIDENT-MODES | Repository file | `docs/incident-modes.md` | Reviewed commit | Future incident-mode planning and implemented-status boundary | Reviewed | Planning-only source for future product direction | Non-Findings |
| R-KEY-CUSTODY | Repository file | `docs/key-custody.md` | Reviewed commit | Future hybrid key-custody design | Reviewed | Planning-only; no implementation | Non-Findings; F-006 |
| R-BROWSER-DECRYPTION | Repository file | `docs/browser-decryption.md` | Reviewed commit | Future browser/client-side decryption design | Reviewed | Planning-only; no implementation | Non-Findings |
| R-BREAK-GLASS | Repository file | `docs/break-glass-key-access.md` | Reviewed commit | Future break-glass and dead-man-switch key-access design | Reviewed | Planning-only; no implementation | Non-Findings |
| R-IOS-PROTOTYPE | Repository file | `docs/ios-local-recorder-prototype.md` | Reviewed commit | Future iOS prototype planning and API mapping | Reviewed | Planning-only; no iOS code exists | Non-Findings |
| R-REPORTS-README | Repository file | `docs/reports/README.md` | Reviewed commit | Existing report workflow and historical-report naming rules | Reviewed | Report index only | Scope And Method |
| R-CODEMAP | Repository file | `docs/code-map.md`; `docs/architecture.md`; `docs/development.md` | Reviewed commit | Package layout, architecture, release and development workflow | Reviewed | Descriptive docs | Current Implementation Summary |
| R-CI | Repository file | `.github/workflows/ci.yml`; `.github/dependabot.yml` | Reviewed commit | CI jobs, workflow permissions, action pinning, Dependabot scope | Reviewed | Workflow definitions only; live run evidence is separate | F-003 |
| R-DOCKER | Repository file | `Dockerfile`; `.dockerignore` | Reviewed commit | Container build and build-context hygiene | Reviewed | Image was not executed by Phase 2 | F-003 |
| R-GOMOD | Repository file | `go.mod`; `go.sum` | Reviewed commit | Module path, Go toolchain, dependency surface | Reviewed | Not a vulnerability database query | Current Implementation Summary |
| R-CONFIG | Repository files | `cmd/api/main.go`; `cmd/api/servers.go`; `internal/config/*.go` | Reviewed commit | Server startup, private/public listener groups, config parsing | Reviewed | No local server execution | Current Implementation Summary |
| R-DB | Repository files | `internal/db/*.go`; `migrations/*.sql` | Reviewed commit | SQLite open path, WAL, foreign keys, migrations, schema invariants | Reviewed | No live DB replay | F-005 |
| R-STORAGE | Repository files | `internal/storage/*.go` | Reviewed commit | Temp uploads, immutable blob commits, path validation, open/remove errors | Reviewed | Filesystem behavior not replayed locally | F-001 |
| R-INCIDENTS | Repository files | `internal/incidents/*.go` | Reviewed commit | Incident, stream, chunk, token, and sentinel-error persistence | Reviewed | Static inspection only | Current Implementation Summary |
| R-ROUTES | Repository file | `internal/httpapi/routes.go` | Reviewed commit | Separate private and public mux route registration | Reviewed | Static inspection only | Current Implementation Summary |
| R-MIDDLEWARE | Repository file | `internal/httpapi/middleware.go` | Reviewed commit | Access logging, redacted token paths, recovery middleware | Reviewed | Does not cover reverse-proxy logs | F-001; F-002 |
| R-RESPONSE | Repository file | `internal/httpapi/response.go` | Reviewed commit | JSON responses, internal error logging, security headers | Reviewed | Static inspection only | F-001 |
| R-UPLOAD | Repository files | `internal/httpapi/upload.go`; `internal/httpapi/chunk_handlers.go` | Reviewed commit | Upload parsing, hash checks, final storage, chunk reads | Reviewed | No HTTP requests replayed | Current Implementation Summary; F-001 |
| R-BUNDLES | Repository files | `internal/httpapi/bundles.go`; `internal/httpapi/bundle_zip.go`; `internal/httpapi/bundle_manifest.go` | Reviewed commit | ZIP generation, fail-closed incident bundles, manifest contents | Reviewed | No bundle generated by Phase 2 | F-001; F-004 |
| R-VIEWER | Repository files | `internal/httpapi/incident_viewer.go`; `internal/httpapi/web/*` | Reviewed commit | Public viewer summaries, token lookup, viewer metadata exposure | Reviewed | Browser behavior not manually replayed | F-004 |
| R-TESTS | Repository files | `internal/httpapi/*_test.go`; `internal/db/*_test.go`; `internal/storage/*_test.go`; `internal/incidents/*_test.go` | Reviewed commit | Existing behavioral test coverage for route split, bundles, uploads, DB, storage | Reviewed selectively | Tests were not run by Phase 2 | Current Implementation Summary; F-003 |

### External authoritative sources consulted

| Key | Source type | Location | Commit/ref/date | Purpose | Status | Limitations | Related findings/sections |
|---|---|---|---|---|---|---|---|
| S-SQLITE-WAL | External authoritative source | SQLite Write-Ahead Logging documentation | Accessed 2026-05-28 | WAL concurrency, same-host constraints, checkpointing and WAL growth considerations | Reviewed | General SQLite documentation, not proof of this deployment | F-005 |
| S-SQLITE-FK | External authoritative source | SQLite Foreign Key Support documentation | Accessed 2026-05-28 | Per-connection foreign-key enablement expectations | Reviewed | General SQLite documentation, not proof of runtime connection state | Current Implementation Summary |
| S-GITHUB-ACTIONS-SECURE | External authoritative source | GitHub Actions secure use reference | Accessed 2026-05-28 | Full-length action SHA pinning and workflow hardening context | Reviewed | General GitHub guidance, not proof of repository settings | F-003 |
| S-OWASP-PATH-TRAVERSAL | External authoritative source | OWASP Path Traversal page | Accessed 2026-05-28 | Path-handling review criteria for filesystem and ZIP paths | Reviewed | General application-security guidance | Current Implementation Summary |

Required external-source categories not consulted: Apple/iOS, Android, recording-law, App Store, Play Store, browser-decryption implementation, and production key-custody implementation sources were not required for implemented-code findings because this reviewed tree contains planning documents only for those areas. Docker-specific external sources were not consulted because this Phase 2 pass did not make Docker-runtime findings beyond repository workflow/build-context observations. Claims depending on unconsulted external categories are avoided or marked as future planning.

### Validation and execution evidence

| Key | Evidence type | Location | Commit/ref/date | Purpose | Status | Limitations | Related findings/sections |
|---|---|---|---|---|---|---|---|
| V-PR-71 | GitHub PR metadata | Pull request `#71`, `Release v0.7.0` | Closed 2026-05-27 | Release context and maintainer validation checklist | Reviewed | Maintainer-supplied validation summary; not replayed by Phase 2 | Executive Summary |
| V-ISSUE-49 | GitHub issue metadata | Issue `#49`, `Add CI Vulnerability And Coverage Signals` | Closed 2026-05-27 | Confirms prior vulnerability/coverage CI follow-up was completed before this release line | Reviewed | Historical issue; current workflow remains source of truth | F-003 |
| V-RUN | GitHub Actions run metadata | Run `26528978662`, CI run number `350` | 2026-05-27, head SHA `12e97543953ff1ba938c128a6afec73e9643acce` | Commit-associated CI status | Reviewed | Metadata only; no local replay | Executive Summary; F-003 |
| V-JOBS | GitHub Actions job metadata | Run `26528978662` jobs | 2026-05-27 | Successful `Go tests`, `Go vulnerability scan`, `Build Linux binary`, and `Build Docker image`; tag-only release jobs skipped | Reviewed | Full logs were not exhaustively inspected | F-003 |
| V-ARTIFACTS | GitHub Actions artifact metadata | Run `26528978662` artifacts | 2026-05-27 | `go-coverage`, Docker build record, and Linux binary artifacts existed and were not expired | Reviewed | Artifacts were not downloaded or executed | F-003 |
| V-LABELS | GitHub label list | `gh label list --repo open-proofline/server --limit 100` | 2026-05-28 | Verified existing labels before writing local issue drafts | Reviewed | Labels can change later | Follow-Up Recommendations |
| V-PHASE2 | Local Codex Phase 2 checks | Current branch | 2026-05-28 | Report cleanup, citation review, public-safety review, `git diff --check` | Completed after edits | Report-validation only, not runtime proof | Generated artifacts |

### Sources, checks, and commands not available or not executed

Phase 2 did not run `go test ./...`, `go vet ./...`, `gofmt`, `govulncheck`, `docker build`, the API server, the simulator smoke test, release artifact download, attestation verification, or live HTTP requests. It relied on the reviewed repository tree, GitHub Actions metadata, PR validation summaries, and source review.

No live deployment, reverse proxy, firewall, DNS, TLS certificate, host logs, backup/restore workflow, disk encryption setup, real incident data, real viewer token, user safety data, or production storage volume was inspected.

No iOS, Android, web-client, protocol, account-system, notification, browser-decryption, production key-custody, break-glass, or public `/v1` authentication implementation existed in the reviewed tree. Those areas are not treated as implemented behavior.

### Generated artifacts and report outputs

| Artifact | Purpose | Status |
|---|---|---|
| `docs/reports/2026-05-28-proofline-v0.7.0-technical-review.md` | Cleaned public technical review report | Generated by Phase 2. |
| `docs/reports/README.md` | Reports index | Updated to list this report. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/README.md` | Branch-scoped draft index | Generated locally only. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/001-sanitize-internal-filesystem-error-logging.md` | Follow-up draft for internal path-safe error logging | Generated locally only; no GitHub issue created. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/002-add-public-viewer-deployment-checklist.md` | Follow-up draft for deployment checklist wording | Generated locally only; no GitHub issue created. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/003-add-runtime-smoke-test-for-built-artifacts.md` | Follow-up draft for CI runtime smoke testing | Generated locally only; no GitHub issue created. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/004-decide-original-filename-metadata-policy.md` | Follow-up draft for metadata exposure policy | Generated locally only; no GitHub issue created. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/005-add-sqlite-wal-operational-guidance.md` | Follow-up draft for SQLite WAL operational guidance | Generated locally only; no GitHub issue created. |
| `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/006-define-v1-access-control-design.md` | Follow-up draft for future `/v1` access-control design | Generated locally only; no GitHub issue created. |

## Scope And Method

This report validates a Deep Research draft for `open-proofline/server` at reviewed commit `12e97543953ff1ba938c128a6afec73e9643acce`, target release `v0.7.0`. The Phase 2 pass checked repository facts against the current checked-out branch and the reviewed commit context, removed non-portable citation tokens, replaced informal draft language, separated implemented behavior from future planning, and converted follow-up recommendations into local branch-scoped issue drafts because issue handling mode was `drafts_only`.

The current branch used for publication work was `add-technical-report-for-v7.0.0` at `7c07b7842d065d312d29022d16f08d6519f5a064` before these report edits. At the start of Phase 2, the branch differed from the reviewed release commit only in `codex/prompts/95-validate-deep-research-report.md`, so repository facts in this report remain pinned to the reviewed release commit. [V-PHASE2]

This is a static technical review plus validation of supplied CI evidence. It is not a formal audit, penetration test, legal review, compliance review, production-readiness certification, or live deployment review.

## Current Implementation Summary

The backend structure is small and conventional. `cmd/api` wires configuration, SQLite, blob storage, incident persistence, and HTTP handlers. `internal/config` owns environment parsing and timeout defaults. `internal/db` opens SQLite, enables foreign keys and WAL mode, and runs migrations. `internal/incidents` owns incident, stream, chunk, checkin, and token persistence. `internal/storage` owns temporary uploads and committed blob storage. `internal/httpapi` owns private routes, public viewer routes, upload parsing, ZIP bundles, and viewer summaries. [R-CONFIG] [R-DB] [R-INCIDENTS] [R-STORAGE] [R-ROUTES]

The private/public listener boundary is represented in code and documentation. Private `/v1` routes handle incident creation, stream creation/completion/failure, chunk upload/list/read, checkins, incident-token creation/revocation, and private bundle downloads. Public incident viewer routes are read-only and token-gated under `/i/{token}`, with pre-rename `/e/{token}` compatibility aliases. [R-API] [R-SECURITY-MODEL] [R-THREAT-MODEL] [R-ROUTES] [R-VIEWER]

Token handling is coherent for the current scope. Incident viewer tokens are generated from random bytes, returned raw only at creation time, stored as SHA-256 hashes, scoped to one incident, and revocable. Omitted expiries default to the configured incident-token TTL, which defaults to 24 hours unless explicitly changed. Public invalid, expired, and revoked token states collapse to the same error. Token-bearing application request logs use route patterns rather than raw token paths. [R-API] [R-INCIDENTS] [R-MIDDLEWARE] [R-VIEWER]

Upload and storage behavior preserves the evidence-store boundary. Uploads stream to temporary storage while SHA-256 is computed, commit final blobs only after hash verification, use server-generated stored paths, and avoid overwriting stored chunks. Streamed chunk identity is scoped by incident, stream, and positive stream-local chunk index. Legacy unstreamed chunks remain compatible but are not included in completed-stream bundle downloads. [R-API] [R-UPLOAD] [R-STORAGE] [R-INCIDENTS]

Completed stream and incident bundles are generated as ZIP responses with server-controlled entry names and generated JSON manifests. The server does not accept client-provided stored paths for download. Incident bundle generation fails closed if a completed stream cannot be reconstructed. The path-handling design aligns with OWASP guidance to avoid letting user input determine filesystem paths. [R-BUNDLES] [R-STORAGE] [S-OWASP-PATH-TRAVERSAL]

SQLite is used as local metadata storage. The code enables foreign-key enforcement on its connection and enables WAL mode at startup. SQLite documents that foreign-key enforcement must be enabled per connection, and its WAL documentation describes the same-host/shared-memory and checkpointing considerations that matter for operational deployments. [R-DB] [S-SQLITE-FK] [S-SQLITE-WAL]

The encryption boundary remains current and clear. The simulator can upload chunks using the documented v1 AES-256-GCM envelope with compatibility names still using `safety-recorder` / `SafetyRecorderChunk`. The backend validates and stores ciphertext bytes only, does not store raw keys, does not decrypt media, and does not expose browser or backend decryption routes. [R-ENCRYPTION] [R-KEY-CUSTODY]

## Findings

### F-001: Internal filesystem errors can include local path detail in server logs

**Severity:** Low.
**Confidence:** High.
**Classification:** Current implementation follow-up.
**Affected files:** `internal/httpapi/response.go`, `internal/httpapi/bundles.go`, `internal/httpapi/bundle_zip.go`, `internal/httpapi/chunk_handlers.go`, `internal/storage/*.go`.

The application correctly avoids exposing server filesystem paths in public client responses and bundle manifests. However, internal error logging still records raw Go error strings for storage and bundle failures. Errors returned by `os.Open`, `os.Remove`, and wrapped bundle-write paths can include local filesystem paths or storage-layout details. This is not a client-visible leak, but it can put sensitive operational path information into application logs. [R-RESPONSE] [R-BUNDLES] [R-UPLOAD] [R-STORAGE]

**Why it matters.** The project explicitly treats raw tokens, request bodies, uploaded bytes, plaintext, raw keys, and private deployment details as public-safety-sensitive. Local storage paths are less sensitive than those prohibited values, but they can still reveal deployment layout and incident-storage context in logs.

**Minimal actionable fix.** Keep client responses generic, and change internal error logging around storage/bundle failures to log stable operation names, safe error categories, and non-secret identifiers only where needed. Preserve enough diagnostic value without logging local absolute paths, stored paths, raw tokens, uploaded bytes, plaintext, or keys.

**Suggested draft issue:** `Sanitize internal filesystem error logging`.

### F-002: Public viewer deployment remains dependent on operator checklist controls

**Severity:** Low.
**Confidence:** High.
**Classification:** Operational follow-up.
**Affected files:** `docs/deployment.md`, `docs/security-model.md`, `docs/threat-model.md`, optionally `docs/development.md`.

The repository clearly documents that public exposure should be limited to the incident viewer listener, and that production-style public viewer exposure still needs TLS, rate limiting, reverse-proxy token-path redaction, retention/backup/deletion enforcement, monitoring, and restore testing. The Go app itself does not include built-in app-level rate limiting and does not make `/v1` safe for public exposure. [R-DEPLOYMENT] [R-SECURITY-MODEL] [R-THREAT-MODEL]

This is accurately documented as a current deployment boundary, not an implementation defect. The remaining improvement is to make public-viewer deployment review easier by turning the scattered requirements into a concise checklist for release and operator review.

**Why it matters.** Viewer URLs are bearer-token URLs. The public viewer can be exposed more safely only when deployment logs, metrics, and edge controls do not undermine token secrecy or availability.

**Minimal actionable fix.** Add or refine a deployment/release checklist covering public viewer TLS, edge rate limiting, proxy log redaction for `/i/{token}` and legacy `/e/{token}` paths, no public `/v1` routing, viewer-token expiry/revocation review, and backup/restore expectations.

**Suggested draft issue:** `Add public incident-viewer deployment checklist`.

### F-003: CI builds artifacts but does not smoke-test built runtime outputs

**Severity:** Informational.
**Confidence:** High.
**Classification:** CI assurance follow-up.
**Affected files:** `.github/workflows/ci.yml`, `Dockerfile`, `.dockerignore`, `docs/development.md`.

The reviewed workflow runs `go vet`, `go test`, `govulncheck`, Linux binary build, Docker image build, and artifact upload. Recent workflow-run metadata for the reviewed commit confirms successful Go tests, vulnerability scan, Linux binary build, Docker image build, and coverage artifact upload. [R-CI] [V-RUN] [V-JOBS] [V-ARTIFACTS]

The workflow builds the binary and Docker image but does not run a minimal startup or runtime smoke test against the produced outputs. This is not a release blocker for `v0.7.0`, but it is a useful next assurance step for a backend that publishes binaries and a container image. GitHub recommends pinning third-party actions to full-length commit SHAs; the current workflow already does this. [R-CI] [S-GITHUB-ACTIONS-SECURE]

**Why it matters.** Build success catches compilation and packaging failures, while a small runtime smoke test can catch startup/configuration regressions in the built artifact path.

**Minimal actionable fix.** Add a narrow CI job or step that starts the built binary and, separately if practical, starts the built Docker image with local-only bindings and verifies a basic health/startup condition without requiring secrets or external services.

**Suggested draft issue:** `Add runtime smoke test for built artifacts`.

### F-004: `original_filename` metadata exposure needs an explicit long-term policy

**Severity:** Informational.
**Confidence:** Medium.
**Classification:** Privacy/design follow-up.
**Affected files:** `internal/httpapi/incident_viewer.go`, `internal/httpapi/bundle_manifest.go`, `docs/api.md`, `docs/security-model.md`.

The backend accepts `original_filename` as optional display metadata, sanitizes it to a basename, and includes it in viewer chunk summaries and bundle manifests when present. This is current behavior rather than an accidental server path leak; stored filesystem paths remain excluded from public viewer summaries and manifests. [R-API] [R-VIEWER] [R-BUNDLES]

The current behavior is acceptable for a development-stage backend, but the long-term product should decide whether user-supplied filenames are useful evidence metadata, unnecessary metadata exposure, or mode-dependent information. This matters more as future interaction records, evidence notes, and account/trusted-contact access are designed.

**Why it matters.** Filenames can contain contextual or personal information even after path stripping. A clear policy helps future clients decide whether to send, suppress, normalize, or separately encrypt filename metadata.

**Minimal actionable fix.** Document the intended `original_filename` policy for current viewer/manifests and future clients. If the policy changes, update API docs and tests.

**Suggested draft issue:** `Decide original filename metadata policy`.

### F-005: WAL and SQLite operational guidance should be expanded before higher-volume use

**Severity:** Informational.
**Confidence:** Medium.
**Classification:** Operational/database follow-up.
**Affected files:** `docs/deployment.md`, `docs/development.md`, `docs/security-model.md`, optionally `internal/db/*.go`.

SQLite startup enables WAL mode and foreign-key enforcement, which is a sound fit for the current local single-process backend shape. SQLite documents that WAL improves concurrency but requires same-host shared-memory behavior, has checkpointing considerations, and can suffer WAL growth when checkpoints cannot complete. [R-DB] [S-SQLITE-WAL] [S-SQLITE-FK]

The current docs mention local disk storage and deployment expectations, but they do not yet provide much operational guidance for WAL checkpoint behavior, DB backup/restore while WAL files exist, or observability for growing deployments.

**Why it matters.** As incident volume and bundle-viewer traffic grow, operators need to know what to back up, what not to put on network filesystems, and what symptoms indicate SQLite/WAL pressure.

**Minimal actionable fix.** Add deployment/development guidance for SQLite WAL files, checkpoint expectations, backup/restore notes, and any simple operational checks the project wants to support before higher-volume use.

**Suggested draft issue:** `Add SQLite WAL operational guidance`.

### F-006: Public `/v1` access-control design remains a future prerequisite

**Severity:** Informational.
**Confidence:** High.
**Classification:** Future design prerequisite, not a current defect.
**Affected files:** `docs/security-model.md`, `docs/threat-model.md`, `docs/deployment.md`, `docs/incident-modes.md`, `docs/key-custody.md`.

The private `/v1` API remains unauthenticated by application-level public user identity. The repository documents this repeatedly and instructs operators to keep `/v1` behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. This report does not treat missing public `/v1` authentication as a current vulnerability because public exposure is explicitly out of scope. [R-SECURITY] [R-SECURITY-MODEL] [R-THREAT-MODEL] [R-DEPLOYMENT]

However, the future product direction includes account-owner access, trusted-contact access, first-class incident modes, and key-custody workflows. Those cannot safely move toward a public control plane until the `/v1` access-control story is designed.

**Why it matters.** Future account, trusted-contact, notification, key-custody, and public control-plane work all depend on clear role and authorization boundaries.

**Minimal actionable fix.** Create a design/backlog item for the future `/v1` access-control model before any public control-plane exposure or account-enabled client work.

**Suggested draft issue:** `Define future /v1 access-control design`.

## Non-Findings And Confirmed Boundaries

The absence of public `/v1` authentication is not treated as an undisclosed defect in this report because the documentation states that `/v1` is private, unauthenticated, and must not be exposed publicly as-is. [R-SECURITY] [R-DEPLOYMENT] [R-SECURITY-MODEL]

Missing iOS, Android, web-client, protocol, account, OAuth/JWT, push notification, SMS, Messenger, first-class incident-type, escalation-policy, browser-decryption, production key-custody, break-glass, public admin dashboard, automated retention/deletion, and playable media export features are not treated as defects. The repository identifies them as absent or future work. [R-README] [R-DOCS-README] [R-INCIDENT-MODES] [R-KEY-CUSTODY] [R-BROWSER-DECRYPTION] [R-BREAK-GLASS] [R-IOS-PROTOTYPE]

Remaining `safety-recorder` compatibility identifiers are not treated as stale product naming by themselves. The documentation explicitly preserves compatibility names for the v1 simulator encryption envelope, default SQLite filename, legacy `/e/{token}` aliases, and historical migration or compatibility contexts until explicit protocol or data-layout migrations are designed. [R-README] [R-DEPLOYMENT] [R-ENCRYPTION]

The report does not claim that Proofline contacts emergency services, reports crimes, provides legal advice, guarantees legal admissibility, has passed platform-store review, or is production-ready public infrastructure. [R-README] [R-SECURITY] [R-INCIDENT-MODES]

## Follow-Up Recommendations

No public GitHub issues were created. Because issue handling mode was `drafts_only`, follow-ups were written as local branch-scoped Markdown drafts under `.backlog-drafts/2026-05-28/add-technical-report-for-v7.0.0/`.

| Draft | Priority | Type | Existing labels used | Related finding |
|---|---|---|---|---|
| `001-sanitize-internal-filesystem-error-logging.md` | P1 | security / logging | `backlog`, `security`, `maintenance`, `go` | F-001 |
| `002-add-public-viewer-deployment-checklist.md` | P1 | deployment / security | `backlog`, `deployment`, `security`, `docs` | F-002 |
| `003-add-runtime-smoke-test-for-built-artifacts.md` | P2 | ci / testing | `backlog`, `ci`, `testing`, `docker` | F-003 |
| `004-decide-original-filename-metadata-policy.md` | P2 | docs / privacy design | `backlog`, `docs`, `security`, `maintenance` | F-004 |
| `005-add-sqlite-wal-operational-guidance.md` | P3 | deployment / database ops | `backlog`, `deployment`, `docs`, `maintenance` | F-005 |
| `006-define-v1-access-control-design.md` | P1 | security / architecture design | `backlog`, `security`, `docs`, `enhancement` | F-006 |

The repository does not currently have labels such as `logging`, `privacy`, `ops`, `database`, `architecture`, or `auth`, so the drafts use the closest existing labels and note the mismatch where relevant. [V-LABELS]

## Conclusion

Proofline `v0.7.0` is internally consistent for the documented server-backend scope. The release should not be described as production-ready public infrastructure, but the reviewed code and docs preserve the important current boundaries: separate private/public route groups, read-only public viewer routes, ciphertext-only evidence storage, server-controlled ZIP paths, no backend decryption, and clear treatment of future incident-mode and key-custody work as planning.

The most actionable short-term improvement is logging hygiene around internal filesystem/storage errors. The next operational improvements are clearer deployment checklist coverage for public viewer exposure and a small runtime smoke test for built artifacts. Broader public `/v1` access control, production key custody, browser decryption, and client work should remain explicit future design tasks.

## Citation References

[R-README]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/README.md
[R-SECURITY]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/SECURITY.md
[R-CHANGELOG]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/CHANGELOG.md
[R-AGENTS]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/AGENTS.md
[R-DOCS-README]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/README.md
[R-API]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/api.md
[R-DEPLOYMENT]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/deployment.md
[R-SECURITY-MODEL]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/security-model.md
[R-THREAT-MODEL]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/threat-model.md
[R-ENCRYPTION]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/encryption.md
[R-INCIDENT-MODES]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/incident-modes.md
[R-KEY-CUSTODY]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/key-custody.md
[R-BROWSER-DECRYPTION]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/browser-decryption.md
[R-BREAK-GLASS]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/break-glass-key-access.md
[R-IOS-PROTOTYPE]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/ios-local-recorder-prototype.md
[R-REPORTS-README]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/docs/reports/README.md
[R-CODEMAP]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/docs
[R-CI]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/.github/workflows/ci.yml
[R-DOCKER]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce
[R-GOMOD]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/go.mod
[R-CONFIG]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/cmd/api
[R-DB]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal/db
[R-STORAGE]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal/storage
[R-INCIDENTS]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal/incidents
[R-ROUTES]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi/routes.go
[R-MIDDLEWARE]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi/middleware.go
[R-RESPONSE]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi/response.go
[R-UPLOAD]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi
[R-BUNDLES]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi
[R-VIEWER]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/internal/httpapi/incident_viewer.go
[R-TESTS]: https://github.com/open-proofline/server/tree/12e97543953ff1ba938c128a6afec73e9643acce/internal
[S-SQLITE-WAL]: https://www.sqlite.org/wal.html
[S-SQLITE-FK]: https://www.sqlite.org/foreignkeys.html
[S-GITHUB-ACTIONS-SECURE]: https://docs.github.com/en/actions/reference/security/secure-use
[S-OWASP-PATH-TRAVERSAL]: https://owasp.org/www-community/attacks/Path_Traversal
[V-PR-71]: https://github.com/open-proofline/server/pull/71
[V-ISSUE-49]: https://github.com/open-proofline/server/issues/49
[V-RUN]: https://github.com/open-proofline/server/actions/runs/26528978662
[V-JOBS]: https://github.com/open-proofline/server/actions/runs/26528978662
[V-ARTIFACTS]: https://github.com/open-proofline/server/actions/runs/26528978662
[V-LABELS]: https://github.com/open-proofline/server/labels
[V-PHASE2]: https://github.com/open-proofline/server/blob/12e97543953ff1ba938c128a6afec73e9643acce/codex/prompts/95-validate-deep-research-report.md
