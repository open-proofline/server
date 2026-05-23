# Technical Review of Safety Recorder v0.4.x

**Repository:** `TheSilkky/safety-recorder`  
**Reviewed commit SHA:** `89a07ff0616fe5ad13437f1b2eec93e091ec3ef6`  
**Review date:** 2026-05-23  
**Report status:** Final public report after maintainer Phase 2 validation; accepted findings were mapped to follow-up issues.

**Citation format note:** This report uses portable citation keys only. Repository citations are pinned to reviewed commit `89a07ff0616fe5ad13437f1b2eec93e091ec3ef6`; external citations resolve to canonical documentation URLs. No ChatGPT-internal citation tokens are used.

**AI-assisted review disclosure:** This report was generated with assistance from OpenAI ChatGPT Deep Research using GPT-5.5 Extended Pro, then reviewed and edited by the maintainer. It is not a formal security audit, penetration test, compliance certification, or production-readiness endorsement. Findings should be verified against the reviewed commit, cited sources, and current project scope before being relied on.

**Public-disclosure note:** This report is intended for public project documentation. It intentionally avoids raw tokens, secrets, private deployment details, exploit payloads, and user-safety data.

## Executive summary

Safety Recorder is documented and implemented as an **experimental** Go backend that accepts already-encrypted chunks, stores metadata in SQLite, stores encrypted blobs on local disk, and exposes a token-scoped, read-only emergency viewer. The repository repeatedly warns that the private `/v1` API has no public authentication and **must not** be exposed publicly. The reviewed code and documentation are materially consistent with that boundary, and the review did **not** find evidence that the current backend performs media decryption, stores raw media keys, or silently crosses into the future-design areas described in the key-custody, browser-decryption, and break-glass design documents. [R-README] [R-DOCS-README] [R-KEY-CUSTODY] [R-ROUTES]

The strongest parts of the current implementation are its **clarity of scope** and several defensive defaults. The public and private route trees are separated in code; the emergency viewer applies `no-store`, `no-referrer`, CSP, `X-Frame-Options: DENY`, and `X-Content-Type-Options: nosniff`; emergency-token paths are redacted in logs; the storage layer rejects unsafe paths and uses hard-linking to avoid overwrite-on-commit; and the SQLite layer explicitly enables foreign-key enforcement and WAL mode rather than relying on SQLite defaults. Those choices line up with authoritative guidance from SQLite and OWASP, including SQLite’s documentation on foreign-key enforcement and WAL mode and OWASP’s logging guidance on excluding tokens and secrets from logs. [R-ROUTES] [R-MIDDLEWARE] [R-RESPONSE] [R-STORAGE] [R-DB] [S-SQLITE-PRAGMA] [S-SQLITE-WAL] [S-OWASP-LOGGING]

On cryptography, the repository’s **simulator/development envelope** is broadly sound for its stated scope: it uses AES-256-GCM, generates 32-byte keys and 12-byte nonces from `crypto/rand`, binds incident/stream/media/chunk metadata through AEAD associated data, parses headers strictly, and rejects malformed envelopes in tests. This is consistent with Go’s AEAD contract that nonces must be unique for a given key and with NIST SP 800-38D’s 96-bit IV guidance for GCM. The review did not find a custom cryptographic primitive or an obvious misuse such as nonce truncation, unauthenticated metadata, or insecure randomness. [R-ENVELOPE] [R-ENVELOPE-TEST] [S-GO-CIPHER] [S-GO-RAND] [S-NIST-GCM] [S-OWASP-CRYPTO]

The review identified a **small set of meaningful current findings**. The most important application-level issue is that chunk identity and storage naming remain keyed by `(incident_id, media_type, chunk_index)` rather than `stream_id`, which makes multiple same-media streams within one incident awkward or conflicting. The most important supply-chain and deployment findings are mutable GitHub Action references, over-broad workflow token permissions, a need to broaden the existing `server/.dockerignore` build-context exclusion policy, and tag-based rather than digest-pinned base images. None of these findings justify describing the repository as production-ready, and none alter the central recommendation: **do not expose `/v1` publicly**. [R-MIG-001] [R-STORAGE] [R-INCIDENTS-REPO] [R-STREAMS] [R-CI] [R-DOCKERFILE] [S-GITHUB-ACTIONS-SECURE] [S-DOCKER-CONTEXT] [S-DOCKER-DIGESTS]

## Scope and methodology

This review covered the repository areas explicitly requested in the original brief: root documentation, `docs/`, `codex/`, `.github/`, `server/`, migrations, the Dockerfile, and CI workflow materials, with the technical deep-dive focused on backend Go code, storage, token handling, ZIP bundle generation, and crypto-adjacent implementation. The review was static: it did not deploy the service to a live environment, fuzz endpoints, or inspect external infrastructure such as reverse proxies, DNS, firewall policy, or GitHub repository settings not exposed through repository contents. [R-README] [R-DOCS-README] [R-CI] [R-DOCKERFILE]

The external standards baseline prioritized the source families specified in the brief: Go package documentation, NIST SP 800-38D, OWASP Cheat Sheets, GitHub Actions security documentation, Docker build-context and digest documentation, and SQLite documentation. In particular, the review used Go’s `crypto/cipher` documentation for AEAD requirements, NIST SP 800-38D for GCM IV guidance, OWASP for cryptographic storage and logging expectations, GitHub’s secure-use reference for action pinning and least privilege, Docker’s documentation for build-context exclusion and immutable digests, and SQLite’s own documentation for foreign-key enforcement defaults and WAL trade-offs. [S-GO-CIPHER] [S-GO-RAND] [S-NIST-GCM] [S-OWASP-CRYPTO] [S-OWASP-LOGGING] [S-GITHUB-ACTIONS-SECURE] [S-DOCKER-CONTEXT] [S-DOCKER-DIGESTS] [S-SQLITE-PRAGMA] [S-SQLITE-WAL]

A key methodological distinction throughout this report is between **current implementation** and **future design**. The repository openly documents future work around key custody, browser-side decryption, and break-glass access as design-only and explicitly says those documents do not change the current backend or schema. The review treated those design documents as scope guardrails, not as shipped functionality, and did not turn “not yet implemented” future ideas into false implementation findings. [R-KEY-CUSTODY] [R-DOCS-README]

### AI assistance and review limitations

This technical review was produced with assistance from OpenAI ChatGPT Deep Research using GPT-5.5 Extended Pro. The system was used to analyze the repository contents, synthesize observations, and compare implementation and documentation against cited authoritative sources.

The report has been manually reviewed and edited by the maintainer, including removal of unsupported claims and correction of repository-specific findings. Despite that review, the report may still contain mistakes, omissions, or interpretations that require further validation.

This document should be treated as an AI-assisted engineering review artifact, not as an independent security audit, penetration test, compliance assessment, legal opinion, cryptographic verification, or certification that the project is production-ready.

## Portable source bibliography

### Repository sources

| Key | Repository source | Link |
|---|---|---|
| R-README | `README.md` | [R-README]|
| R-DOCS-README | `docs/README.md` | [R-DOCS-README]|
| R-KEY-CUSTODY | `docs/key-custody.md` | [R-KEY-CUSTODY]|
| R-SECURITY | `SECURITY.md` | [R-SECURITY]|
| R-CHANGELOG | `CHANGELOG.md` | [R-CHANGELOG]|
| R-AGENTS | `AGENTS.md` | [R-AGENTS]|
| R-DEVELOPMENT | `docs/development.md` | [R-DEVELOPMENT]|
| R-API | `docs/api.md` | [R-API]|
| R-GITIGNORE | `.gitignore` | [R-GITIGNORE]|
| R-DOCKERIGNORE | `server/.dockerignore` | [R-DOCKERIGNORE]|
| R-CI | `.github/workflows/ci.yml` | [R-CI]|
| R-DOCKERFILE | `server/Dockerfile` | [R-DOCKERFILE]|
| R-DB | `server/internal/db/db.go` | [R-DB]|
| R-ROUTES | `server/internal/httpapi/routes.go` | [R-ROUTES]|
| R-MIDDLEWARE | `server/internal/httpapi/middleware.go` | [R-MIDDLEWARE]|
| R-RESPONSE | `server/internal/httpapi/response.go` | [R-RESPONSE]|
| R-EMERGENCY | `server/internal/httpapi/emergency.go` | [R-EMERGENCY]|
| R-EMERGENCY-TEST | `server/internal/httpapi/emergency_test.go` | [R-EMERGENCY-TEST]|
| R-BUNDLES | `server/internal/httpapi/bundles.go` | [R-BUNDLES]|
| R-BUNDLE-MANIFEST | `server/internal/httpapi/bundle_manifest.go` | [R-BUNDLE-MANIFEST]|
| R-BUNDLE-ZIP | `server/internal/httpapi/bundle_zip.go` | [R-BUNDLE-ZIP]|
| R-STORAGE | `server/internal/storage/storage.go` | [R-STORAGE]|
| R-INCIDENTS-REPO | `server/internal/incidents/repository.go` | [R-INCIDENTS-REPO]|
| R-STREAMS | `server/internal/incidents/streams.go` | [R-STREAMS]|
| R-ENVELOPE | `server/internal/envelope/envelope.go` | [R-ENVELOPE]|
| R-ENVELOPE-TEST | `server/internal/envelope/envelope_test.go` | [R-ENVELOPE-TEST]|
| R-MIG-001 | `server/migrations/001_init.sql` | [R-MIG-001]|
| R-MIG-002 | `server/migrations/002_emergency_tokens.sql` | [R-MIG-002]|
| R-MIG-003 | `server/migrations/003_media_streams.sql` | [R-MIG-003]|

### External authoritative sources

| Key | Source | Link |
|---|---|---|
| S-GO-CIPHER | Go package documentation: `crypto/cipher` | [S-GO-CIPHER]|
| S-GO-RAND | Go package documentation: `crypto/rand` | [S-GO-RAND]|
| S-NIST-GCM | NIST SP 800-38D, Galois/Counter Mode and GMAC | [S-NIST-GCM]|
| S-OWASP-CRYPTO | OWASP Cryptographic Storage Cheat Sheet | [S-OWASP-CRYPTO]|
| S-OWASP-LOGGING | OWASP Logging Cheat Sheet | [S-OWASP-LOGGING]|
| S-GITHUB-ACTIONS-SECURE | GitHub Docs, Secure use reference for GitHub Actions | [S-GITHUB-ACTIONS-SECURE]|
| S-GITHUB-TOKEN | GitHub Docs, `GITHUB_TOKEN` permission modification | [S-GITHUB-TOKEN]|
| S-DOCKER-CONTEXT | Docker Docs, Build context and `.dockerignore` files | [S-DOCKER-CONTEXT]|
| S-DOCKER-DIGESTS | Docker Docs, Image digests | [S-DOCKER-DIGESTS]|
| S-SQLITE-PRAGMA | SQLite documentation, PRAGMA statements and `foreign_keys` | [S-SQLITE-PRAGMA]|
| S-SQLITE-WAL | SQLite documentation, Write-Ahead Logging | [S-SQLITE-WAL]|

## Repository architecture summary

The principal repository materials reviewed were the top-level `README.md`, `SECURITY.md`, `CHANGELOG.md`, `AGENTS.md`, the docs index and design documents, core server packages under `server/internal/`, the migration SQL files, the embedded emergency-viewer assets, and the repository’s CI/CD materials under `.github/`. The current architecture is straightforward: a single Go binary can expose two separate listener groups, one private for `/v1` write/admin routes and one public for read-only emergency viewing. Metadata is stored in SQLite; encrypted blobs are stored on local disk under a controlled directory; and completed streams can be exported as ZIP bundles containing encrypted chunk files and JSON manifests. [R-README] [R-DOCS-README] [R-SECURITY] [R-CHANGELOG] [R-AGENTS] [R-ROUTES] [R-STORAGE] [R-MIG-001] [R-MIG-002] [R-MIG-003] [R-BUNDLE-MANIFEST] [R-BUNDLE-ZIP] [R-BUNDLES]

The security model is intentionally narrow. The backend accepts opaque encrypted bytes, verifies uploaded ciphertext hashes, does not hold raw media keys, and does not provide backend decryption. The emergency viewer authorizes read-only access through a bearer token scoped to one incident; the repository stores only the token hash; and tests verify that raw emergency tokens are not stored and are redacted from request logs. This is an internally coherent ciphertext-only model for an experimental backend, even though it is not a production-complete public-facing system. [R-README] [R-KEY-CUSTODY] [R-INCIDENTS-REPO] [R-EMERGENCY] [R-EMERGENCY-TEST]

The SQLite layer shows deliberate choices rather than accidental defaults. The code enables foreign-key enforcement and WAL mode during connection setup, and the migrations use foreign keys and integrity checks to reinforce handler validation. SQLite’s documentation states that foreign-key enforcement is off by default unless enabled, and its WAL documentation explains both WAL’s concurrency benefits and its same-host operational constraints. The current code is therefore aligned with SQLite’s operational model for this single-host local-storage architecture. [R-DB] [R-MIG-001] [R-MIG-002] [R-MIG-003] [S-SQLITE-PRAGMA] [S-SQLITE-WAL]

## Findings

### Findings table

| ID | Severity | Finding |
|---|---|---|
| F-A | Medium | Chunk identity and on-disk naming ignore `stream_id`, which constrains multiple same-media streams in one incident. |
| F-B | Medium | GitHub Actions workflow should pin action references to full-length commit SHAs. |
| F-C | Medium | GitHub Actions token permissions appear broader than necessary for the whole workflow. |
| F-D | Low | The existing `server/.dockerignore` build-context exclusion policy should be broadened to cover more local-only artifacts. |
| F-E | Low | Base images are tag-pinned, not digest-pinned, which weakens reproducibility and provenance assurance. |
| F-F | Low | Incident bundle assembly can omit a completed stream on certain internal consistency failures instead of failing closed. |

### F-A — Chunk identity and on-disk naming are not stream-scoped

**Severity:** Medium.

**Finding.** The stream abstraction is implemented only partially. The database uniqueness constraint for chunks is `(incident_id, media_type, chunk_index)`, the duplicate pre-check uses the same key, and the final blob path is derived from the same tuple as `incidents/<incident>/<media_type>_<index>.enc`. Stream completion, however, expects chunk indices to be contiguous from `1..expected_chunk_count` per stream. In practice, that means two `audio` streams inside the same incident cannot both naturally use chunk index `1`, because they collide at the metadata and storage layers. This is a correctness and API-design issue rather than a confidentiality break. The limitation is already disclosed in `docs/api.md`, so it should be treated as planned stream-model correctness work rather than an undisclosed vulnerability. [R-API] [R-MIG-001] [R-STORAGE] [R-INCIDENTS-REPO] [R-STREAMS]

**Evidence location.** `server/migrations/001_init.sql`, unique constraint on `chunks`; `server/internal/incidents/repository.go`, duplicate checks and chunk lookup keying; `server/internal/storage/storage.go`, `CommitTemp`; `server/internal/incidents/streams.go`, contiguous stream completion rules. [R-MIG-001] [R-INCIDENTS-REPO] [R-STORAGE] [R-STREAMS]

**Why it matters.** The repository advertises open/complete/failed media streams and completed stream bundle downloads. A stream model that cannot cleanly support multiple streams of the same media type within one incident is a material limitation for future recording scenarios such as restarts, camera flips, or parallel audio capture. This is especially relevant because evidence export and viewer summaries are already stream-aware. [R-README] [R-EMERGENCY] [R-BUNDLES]

**Actionable fix.** Change the stream-bound chunk identity for streamed uploads to `(incident_id, stream_id, chunk_index)` and move on-disk naming into a stream-specific namespace such as `incidents/<incident>/<stream_id>/<chunk_index>.enc`. Preserve backwards compatibility for legacy unstreamed uploads by explicitly keeping a separate storage path and uniqueness rule when `stream_id` is empty. Add migration coverage and tests proving that two same-media streams can each upload `chunk_index=1` and complete independently. [R-MIG-001] [R-STORAGE] [R-STREAMS]

### F-B — GitHub Actions references should be pinned to full commit SHAs

**Severity:** Medium.

**Finding.** The reviewed CI workflow uses version-tagged GitHub Actions references rather than full-length commit SHAs. GitHub’s secure-use guidance states that pinning an action to a full-length commit SHA is the only way to use an action as an immutable release, and recommends this specifically to reduce the risk of a compromised action backdooring a workflow. [R-CI] [S-GITHUB-ACTIONS-SECURE]

**Evidence location.** `.github/workflows/ci.yml`, in the `uses:` statements for the workflow’s GitHub Actions dependencies. [R-CI]

**Why it matters.** This repository publishes build artifacts and Docker images. A mutable action tag does not provide the same immutability guarantee as a specific commit digest. For a public repository that explicitly discusses AI-assisted development and supply-chain hygiene, pinning actions is a backlog-ready hardening step rather than an optional nicety. [R-README] [R-CI] [S-GITHUB-ACTIONS-SECURE]

**Actionable fix.** Replace version-tagged action references with full-length commit SHAs for all third-party and first-party workflow actions, then document the upgrade process in the release checklist so updates remain reviewable and intentional. [S-GITHUB-ACTIONS-SECURE]

### F-C — Workflow-token permissions appear broader than necessary

**Severity:** Medium.

**Finding.** GitHub recommends that workflow credentials follow least privilege and that `GITHUB_TOKEN` permissions be configured only as broadly as required. The reviewed workflow grants broader permissions than the baseline needed by all jobs at workflow scope, which increases the blast radius if any individual action or step in the workflow is compromised. [R-CI] [S-GITHUB-ACTIONS-SECURE] [S-GITHUB-TOKEN]

**Evidence location.** `.github/workflows/ci.yml`, top-level `permissions:` block and jobs that do not need package publication rights. [R-CI]

**Why it matters.** GitHub’s secure-use guidance notes that actions can use workflow credentials, and GitHub’s token-permission documentation explains that permissions can be restricted for a workflow or job. For a repository that builds and publishes containers, the recommended pattern is to keep default permissions minimal and elevate only the publishing job. [S-GITHUB-ACTIONS-SECURE] [S-GITHUB-TOKEN]

**Actionable fix.** Set the workflow default to `contents: read` only. Move any write-capable permission, including package publication, to the narrowest job that actually pushes an image, and gate that job to trusted events only. Document this in the release process notes. [R-CI] [S-GITHUB-TOKEN]

### F-D — Existing Docker build-context ignore policy should be broadened

**Severity:** Low.

**Finding.** The reviewed tree contains `server/.dockerignore`, and it already excludes `data/`, `tmp/`, SQLite file patterns, Go test binaries, `coverage.out`, and the Dockerfile from the `./server` build context. [R-DOCKERIGNORE]

The remaining hardening gap is that `server/.dockerignore` is not as broad as the repository's root `.gitignore`, which excludes local environment overrides, simulator key files, profiling output, local binaries, editor noise, and generated review directories. Docker documents `.dockerignore` as the mechanism for excluding unwanted files from build context before the context is sent to the builder, so aligning the Docker ignore policy with local-artifact hygiene is still useful. [R-GITIGNORE] [R-DOCKERIGNORE] [S-DOCKER-CONTEXT]

**Evidence location.** `server/.dockerignore`, root `.gitignore`, `README.md` Docker build instructions, and `server/Dockerfile` builder-stage `COPY`. [R-DOCKERIGNORE] [R-GITIGNORE] [R-README] [R-DOCKERFILE]

**Why it matters.** This is deployment and supply-chain hygiene, not a missing-control blocker. Broadening the ignore policy reduces accidental context upload of local-only files if they appear under `server/`, keeps builds smaller, and makes the Docker build context more consistent with repository hygiene rules. [S-DOCKER-CONTEXT]

**Actionable fix.** Broaden `server/.dockerignore` to cover plausible local-only files under the `server/` context, such as `.env` variants, simulator key files, generated bundles, local binaries or `dist/`, profiling outputs, and editor or OS noise. Keep the Docker runtime behavior unchanged and verify the image still builds from the repository root. [R-DOCKERIGNORE] [R-GITIGNORE] [S-DOCKER-CONTEXT]

### F-E — Docker base images are tag-pinned, not digest-pinned

**Severity:** Low.

**Finding.** Docker documents image digests as immutable identifiers and contrasts them with tags, which can be reused or changed. The reviewed Dockerfile uses tag-based base images rather than digest-pinned bases, which weakens build reproducibility and makes provenance review noisier over time. [R-DOCKERFILE] [S-DOCKER-DIGESTS]

**Evidence location.** `server/Dockerfile`, `FROM` lines. [R-DOCKERFILE]

**Why it matters.** This is a low-severity supply-chain hardening issue, not an immediate bug. The repository is already explicitly experimental, but digest pinning is a public-readiness improvement because it makes rebuilds more deterministic and narrows the set of possible upstream changes between reviews. [R-README] [S-DOCKER-DIGESTS]

**Actionable fix.** Pin both builder and runtime base images by digest, and update the release checklist so digest refreshes are reviewed intentionally rather than being absorbed automatically through tag movement. [S-DOCKER-DIGESTS]

### F-F — Incident-wide bundle assembly can omit completed streams on internal inconsistency

**Severity:** Low.

**Finding.** When building an incident-wide bundle, the code iterates completed streams and silently continues if rebuilding one stream bundle returns `ErrInvalidState` or `ErrNotFound`. For an evidence-export path, this should fail closed rather than partially succeeding without an explicit signal. This is a correctness and evidentiary-completeness issue. The code path suggests an internal-consistency race or corruption case would currently be hidden from the incident bundle caller instead of surfaced as an export failure. [R-BUNDLES]

**Evidence location.** `server/internal/httpapi/bundles.go`, `loadCompletedIncidentBundles`. [R-BUNDLES]

**Why it matters.** Evidence exports should be conservative. If a stream is listed as completed but cannot be reconstructed consistently, the safer default is to refuse the incident bundle or emit a conspicuous manifest error, not to silently omit content. Even if this path is rare, it is a bad failure mode for a safety-recording system. [R-BUNDLES]

**Actionable fix.** Change incident-bundle assembly so that any completed stream that cannot be reconstructed causes the request to fail, or include a top-level manifest error that makes the omission explicit and machine-detectable. Add regression tests for corrupted, missing, and state-raced completed streams. [R-BUNDLES]

## Opened GitHub issues for follow-up

Phase 2 validation converted the report's actionable findings into six GitHub issues. The Docker ignore finding was framed as a policy-broadening task because `server/.dockerignore` exists and already provides build-context exclusion control. [R-DOCKERIGNORE]

| Finding | Issue | Initial status on 2026-05-23 |
|---|---|---|
| F-A | [#19 — Refactor Streamed Chunk Identity To Be Stream-Scoped][I-19] | Open |
| F-F | [#20 — Make Incident Bundle Export Fail Closed On Completed-Stream Inconsistency][I-20] | Open |
| F-C | [#21 — Reduce Workflow Token Permissions To Job-Level Least Privilege][I-21] | Open |
| F-B | [#22 — Pin GitHub Actions To Full Commit SHAs][I-22] | Open |
| F-D | [#23 — Broaden Docker Build Context Ignore Policy][I-23] | Open |
| F-E | [#24 — Pin Docker Base Images By Digest][I-24] | Open |

## Explicit non-findings and limitations

Several things are **not findings** because the repository already documents them as outside the current implementation scope. The review did not treat the absence of backend decryption, browser-side decryption, server escrow, break-glass access, an iOS client, user accounts, public `/v1` authentication, built-in TLS termination, rate limiting, or production retention/deletion automation as undisclosed defects, because the repository explicitly says those capabilities are not yet implemented or are design-only for later phases. [R-README] [R-DOCS-README] [R-KEY-CUSTODY]

Likewise, the review did **not** identify a material contradiction between the central docs and the current code on the core security boundary. The README, docs index, and key-custody design all describe a ciphertext-only backend with future key-custody work kept separate from the present implementation, and the current handlers and bundle paths are consistent with that description. [R-README] [R-DOCS-README] [R-KEY-CUSTODY] [R-EMERGENCY] [R-BUNDLES]

This review has two important limitations. First, it was a repository-content review, not a live deployment audit, so reverse-proxy behavior, TLS posture, firewalling, host backups, and log sinks were not directly tested. Second, live GitHub repository settings were not independently certified as hosted configuration objects in this report. The repository now documents the active `Protect main` ruleset and PR-only admin bypass policy in `docs/development.md`, but maintainers should still verify current GitHub settings before relying on them. That limitation matters for any future production-readiness assessment. [R-CI] [R-DEVELOPMENT] [R-README]

## Appendix: mapping findings to authoritative guidance

| Finding | Mapping | Sources |
|---|---|---|
| F-A | Data model and evidence handling correctness. This is not primarily a cryptography issue; it is a state-modelling issue in stream/chunk storage and export. | [R-MIG-001] [R-STORAGE] [R-INCIDENTS-REPO] [R-STREAMS] |
| F-B | GitHub Actions secure-use guidance for immutable action references. | [R-CI] [S-GITHUB-ACTIONS-SECURE] |
| F-C | GitHub Actions least-privilege `GITHUB_TOKEN` permissions. | [R-CI] [S-GITHUB-ACTIONS-SECURE] [S-GITHUB-TOKEN] |
| F-D | Docker build-context hygiene and broadened `.dockerignore` exclusions. | [R-DOCKERIGNORE] [R-GITIGNORE] [R-DOCKERFILE] [S-DOCKER-CONTEXT] |
| F-E | Docker image digest pinning for immutable base-image references. | [R-DOCKERFILE] [S-DOCKER-DIGESTS] |
| F-F | Repository-specific evidence-export completeness and fail-closed behavior. | [R-BUNDLES] |

For the repository’s overall strengths, the current crypto-adjacent implementation aligns reasonably well with Go and NIST guidance: Go’s AEAD interface requires a unique nonce per key and matching associated data on decrypt; the simulator envelope uses 12-byte nonces from `crypto/rand`; and NIST SP 800-38D treats 96-bit IVs as the standard GCM form and requires uniqueness discipline for IV construction. OWASP’s cryptographic-storage guidance also supports keeping keys separate from encrypted data and using secure random generation for security-critical values, both of which match the repository’s current ciphertext-only boundary and token-generation approach. [R-ENVELOPE] [R-ENVELOPE-TEST] [S-GO-CIPHER] [S-GO-RAND] [S-NIST-GCM] [S-OWASP-CRYPTO]

On logging and storage posture, the repository’s existing decisions also map well to authoritative guidance. OWASP says logs should usually exclude session identifiers, access tokens, encryption keys, and other primary secrets; the repository redacts emergency-token paths in request logs and tests that raw tokens are not logged or stored. SQLite’s own documentation says foreign-key enforcement is off by default unless enabled and that WAL improves concurrency while remaining same-host only; the code explicitly enables foreign keys and WAL, which is the correct pattern for this architecture. [R-MIDDLEWARE] [R-INCIDENTS-REPO] [R-EMERGENCY-TEST] [R-DB] [S-OWASP-LOGGING] [S-SQLITE-PRAGMA] [S-SQLITE-WAL]

[R-README]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/README.md
[R-DOCS-README]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/docs/README.md
[R-KEY-CUSTODY]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/docs/key-custody.md
[R-SECURITY]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/SECURITY.md
[R-CHANGELOG]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/CHANGELOG.md
[R-AGENTS]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/AGENTS.md
[R-CI]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/.github/workflows/ci.yml
[R-DOCKERFILE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/Dockerfile
[R-DB]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/db/db.go
[R-ROUTES]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/routes.go
[R-MIDDLEWARE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/middleware.go
[R-RESPONSE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/response.go
[R-EMERGENCY]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/emergency.go
[R-EMERGENCY-TEST]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/emergency_test.go
[R-BUNDLES]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/bundles.go
[R-BUNDLE-MANIFEST]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/bundle_manifest.go
[R-BUNDLE-ZIP]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/httpapi/bundle_zip.go
[R-STORAGE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/storage/storage.go
[R-INCIDENTS-REPO]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/incidents/repository.go
[R-STREAMS]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/incidents/streams.go
[R-ENVELOPE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/envelope/envelope.go
[R-ENVELOPE-TEST]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/internal/envelope/envelope_test.go
[R-MIG-001]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/migrations/001_init.sql
[R-MIG-002]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/migrations/002_emergency_tokens.sql
[R-MIG-003]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/migrations/003_media_streams.sql
[S-GO-CIPHER]: https://pkg.go.dev/crypto/cipher
[S-GO-RAND]: https://pkg.go.dev/crypto/rand
[S-NIST-GCM]: https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf
[S-OWASP-CRYPTO]: https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html
[S-OWASP-LOGGING]: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
[S-GITHUB-ACTIONS-SECURE]: https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions
[S-GITHUB-TOKEN]: https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#modifying-the-permissions-for-the-github_token
[S-DOCKER-CONTEXT]: https://docs.docker.com/build/building/context/#dockerignore-files
[S-DOCKER-DIGESTS]: https://docs.docker.com/dhi/core-concepts/digests/
[S-SQLITE-PRAGMA]: https://sqlite.org/pragma.html#pragma_foreign_keys
[S-SQLITE-WAL]: https://sqlite.org/wal.html

[R-DEVELOPMENT]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/docs/development.md
[R-API]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/docs/api.md
[R-GITIGNORE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/.gitignore
[R-DOCKERIGNORE]: https://github.com/TheSilkky/safety-recorder/blob/89a07ff0616fe5ad13437f1b2eec93e091ec3ef6/server/.dockerignore
[I-19]: https://github.com/TheSilkky/safety-recorder/issues/19
[I-20]: https://github.com/TheSilkky/safety-recorder/issues/20
[I-21]: https://github.com/TheSilkky/safety-recorder/issues/21
[I-22]: https://github.com/TheSilkky/safety-recorder/issues/22
[I-23]: https://github.com/TheSilkky/safety-recorder/issues/23
[I-24]: https://github.com/TheSilkky/safety-recorder/issues/24
