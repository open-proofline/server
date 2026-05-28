# Production Cluster Scope

This document records the planned production-cluster expansion path for Proofline Server.

It is a planning and scope document only. It does not implement PostgreSQL, Valkey/Redis-compatible coordination, S3-compatible object storage, public `/v1` authentication, account management, cloud deployment automation, or production hardening.

## Current Local-First Scope

The current backend remains local-first and experimental:

- SQLite metadata remains supported.
- Local filesystem encrypted blob storage remains supported.
- The simulator and local development flow remain supported.
- The private `/v1` API remains private and unauthenticated.
- The public incident viewer remains token-gated and read-only.
- The backend stores ciphertext only and does not decrypt chunks.

SQLite plus local filesystem storage should remain the default development and small self-hosted deployment shape unless a future release deliberately changes defaults.

## Planned Cluster Scope

The planned production-cluster path adds optional backend support for deployments where more than one API node may handle requests.

Planned optional cluster backends:

| Capability | Local/default backend | Planned cluster backend |
|---|---|---|
| Metadata | SQLite | PostgreSQL |
| Committed encrypted chunks | Local filesystem | S3-compatible object storage |
| Short-lived coordination | None | Valkey/Redis-compatible coordination |

These backends should be additive. They must not remove or weaken SQLite and local filesystem support.

## Cluster-Safety Principles

Cluster-aware behavior means duplicate attempts may happen, but duplicate side effects must not happen.

Production-cluster support should rely on:

- stable operation identities
- idempotency keys for retryable client operations
- database uniqueness constraints for metadata
- object-storage conditional writes for immutable encrypted chunks
- retry-safe upload state transitions
- explicit cleanup for abandoned staging state

Valkey or another Redis-compatible service may reduce duplicate work, hold short-lived leases, and support retry coordination, but it must not be the permanent source of truth for incident metadata or committed encrypted chunks.

## PostgreSQL Scope

PostgreSQL support is planned as the production metadata backend.

PostgreSQL should store:

- incidents
- media streams
- chunk metadata
- checkins
- viewer-token metadata
- future retention/deletion state
- future account and access-control metadata, after that design exists
- upload operation and idempotency state when cluster uploads are implemented

PostgreSQL support should include:

- a separate PostgreSQL migration path
- schema constraints equivalent to or stronger than the SQLite schema
- uniqueness constraints for stream-scoped and legacy chunk identities
- transaction boundaries for chunk metadata insertion and stream completion
- restore and migration documentation before production use

SQLite should remain supported for local development, simulator workflows, and small deployments.

## S3-Compatible Object Storage Scope

S3-compatible object storage is planned as the production blob backend for committed encrypted chunks.

The object store should hold opaque encrypted bytes only. It must not require server-side decryption or raw media keys.

Object-storage support should include:

- server-controlled object keys
- staging keys for in-progress uploads, if needed
- final immutable object keys for committed encrypted chunks
- conditional write or equivalent no-overwrite behavior for final objects
- lifecycle cleanup guidance for abandoned staging objects
- backup and restore guidance that keeps metadata and blobs consistent

The local filesystem backend should remain supported and should continue to use relative server-controlled stored paths.

## Valkey / Redis-Compatible Coordination Scope

Valkey or another Redis-compatible service is planned as optional production coordination, not durable storage.

It may be used for:

- upload leases
- idempotency result caching
- short-lived in-progress state
- retry coordination
- rate-limit counters, if application-level rate limiting is later implemented
- cleanup coordination for abandoned staging uploads

It must not be used as the final source of truth for:

- incident metadata
- chunk metadata
- committed encrypted chunk bytes
- viewer-token metadata
- retention or deletion decisions

If Valkey is unavailable, the system should fail closed or return retryable errors for affected operations. PostgreSQL constraints and object-storage no-overwrite behavior must still protect committed state from duplicates.

## Upload Operation Semantics

Future cluster upload handling should move toward explicit upload operations.

A safe cluster upload flow should be designed around these steps:

1. Reserve or identify the upload operation using stable incident, stream, chunk index, media type, and idempotency metadata.
2. Stage encrypted bytes while computing SHA-256 over the uploaded ciphertext.
3. Verify the computed hash against the client-provided hash.
4. Commit encrypted bytes to the final immutable blob location.
5. Insert or confirm chunk metadata in PostgreSQL.
6. Return an idempotent success response when an equivalent chunk already exists.
7. Return a conflict when the same chunk identity is attempted with different ciphertext or metadata.
8. Clean up abandoned staging state conservatively.

A successful chunk upload should mean encrypted bytes are durably committed outside the staging backend and metadata has been written or confirmed. Loss of pre-commit staging state must be recoverable by client retry.

## Boundaries And Non-Goals

This scope expansion does not by itself add:

- public exposure of the current private `/v1` API
- public account management
- OAuth, JWT, sessions, or user accounts
- web, iOS, Android, or shared protocol implementation in this repository
- backend decryption
- raw server-held media keys
- server-side playable media export
- trusted-contact accounts
- push, SMS, Messenger, or emergency-services integrations
- Docker Compose, Kubernetes, Nomad jobs, Terraform, or provider-specific deployment code

Any future deployment automation must preserve private/public listener separation and must not claim production readiness until the access-control, retention, backup, restore, observability, and abuse-control work exists.

## Implementation Order

Preferred implementation sequence:

1. Add configuration scaffolding for backend selection while preserving current defaults.
2. Introduce metadata and blob-store interfaces around the current SQLite and filesystem implementations.
3. Add S3-compatible blob storage as an optional backend.
4. Add PostgreSQL metadata support as an optional backend.
5. Add explicit idempotency and upload-operation semantics for cluster-safe retries.
6. Add optional Valkey/Redis-compatible coordination.
7. Update deployment, backup, restore, security, and threat-model docs before recommending any production cluster deployment.

Each step should be small, reviewable, and tested against the existing SQLite/filesystem path before adding new backend-specific behavior.

## Documentation Updates Required Before Implementation

Implementation PRs for this scope should update source-of-truth docs together, as applicable:

- `README.md`
- `AGENTS.md`
- `SECURITY.md`
- `docs/architecture.md`
- `docs/configuration.md`
- `docs/api.md`
- `docs/deployment.md`
- `docs/security-model.md`
- `docs/threat-model.md`
- `docs/retention-backup-deletion.md`
- `docs/code-map.md`
- release and Deep Research report prompts when review scope changes

Backlog issues should be created or updated for each backend and cluster-safety milestone before implementation work starts.
