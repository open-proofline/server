# Documentation

This directory contains the detailed documentation for Proofline Server, the Go backend component of the planned Proofline project. The top-level [README](../README.md) is a concise server overview; these docs keep operational, API, deployment, incident-capture, and development details in one place.

## Contents

| Document | Purpose |
|---|---|
| [Getting started](getting-started.md) | Run the backend locally and exercise the simulator flow. |
| [Architecture](architecture.md) | System diagrams, listener boundaries, repository split, and server data flow. |
| [Configuration](configuration.md) | Environment variables, backend selectors, bind addresses, upload limits, and data layout. |
| [Production cluster scope](production-cluster-scope.md) | Additive path for optional PostgreSQL metadata, optional S3-compatible object storage, and optional Valkey/Redis-compatible coordination. |
| [Cluster backup, restore, and failure runbook](cluster-backup-restore-runbook.md) | Operational guidance for optional PostgreSQL metadata, S3-compatible encrypted blobs, configuration, coordination, restore validation, and cluster failure modes. |
| [PostgreSQL metadata migration path](postgresql-metadata-migration.md) | PostgreSQL metadata backend schema parity, migrations, transaction boundaries, tests, migration limits, and restore expectations. |
| [Cluster-safe upload operation semantics](cluster-safe-upload-semantics.md) | Planning design for future upload operation identity, idempotency state, commit ordering, retry success, conflict handling, and cleanup across metadata and blob backends. |
| [Resumable upload and upload lease protocol](resumable-upload-lease-protocol.md) | Planning decision to defer resumable uploads and upload leases for a local desktop recorder simulator client while preserving complete encrypted chunk retry semantics, poor-network simulation, and future account-flow shape. |
| [Incident capture modes](incident-modes.md) | Planned emergency, interaction-record, safety-check, and evidence-note modes, plus future capture-profile, escalation-policy, sharing-state, and migration boundaries. |
| [/v1 access control](v1-access-control.md) | Current local account/session boundary plus future role, grant, public product API, private admin API listener, audit, and migration boundaries for account-owner, trusted-contact, public-link, admin/operator, and optional escrow access. |
| [Encryption](encryption.md) | Client-side chunk envelope, simulator key file, and local bundle verification. |
| [iOS local recorder prototype](ios-local-recorder-prototype.md) | Future native incident-capture scope, chunking, encrypted staging, retry, and API mapping. |
| [Key custody and emergency access](key-custody.md) | Future production key custody, trusted-contact access, and break-glass design. |
| [Contact-wrapped key metadata simulator prototype](contact-wrapped-key-metadata-simulator.md) | Simulator-only design for modeling trusted-contact public keys, non-secret key IDs, wrapped stream media keys, and safe development metadata without production key custody. |
| [Browser-side decryption](browser-decryption.md) | Future incident viewer decryption options, risks, and phased direction. |
| [Live partial stream access boundary](live-partial-stream-access-boundary.md) | Future live or partial stream access roles, stream-state exposure, partial manifests, caching, and key-custody dependencies. |
| [Break-glass key access](break-glass-key-access.md) | Future optional server-assisted emergency key access and dead-man-switch design. |
| [API](api.md) | Current private `/v1` API routes including health/readiness checks, private `/admin` web routes, request examples, response examples, and bundle formats. |
| [Deployment](deployment.md) | Local, Docker, SQLite WAL operations, reverse proxy, TLS, and public exposure notes. |
| [Retention, backup, and deletion](retention-backup-deletion.md) | Operational policy for evidence lifecycle, backups, restores, and deletion limits. |
| [Incident deletion and retention enforcement design](incident-deletion-retention-enforcement.md) | Future design for private/admin deletion decisions, retention jobs, tombstones, blob deletion retry, and safe audit boundaries. |
| [Security model](security-model.md) | Current controls, browser headers, logging posture, and security assumptions. |
| [Threat model](threat-model.md) | Assets, trust boundaries, controls, limitations, and next security steps. |
| [Simulator](simulator.md) | Simulator commands and test flows. |
| [Development](development.md) | Repository layout, commands, AI assistance note, branch rulesets, checks, and release checklist notes. |
| [Compose smoke tests](../compose/README.md) | Local release-smoke stacks for SQLite/local, PostgreSQL/local, SQLite/S3-compatible MinIO, and full PostgreSQL/MinIO/Valkey combinations. |
| [Codex change control](codex-change-control.md) | Rollback points, scoped Codex tasks, review steps, and issue-first backlog rules. |
| [Code map](code-map.md) | Package layout and main backend request flows. |
| [Reports](reports/README.md) | Public technical review reports and report-generation workflow notes. |

## Current Repository Scope

This repository is the Go server backend only. In the planned multi-repo layout it corresponds to:

```text
open-proofline/server
```

Future companion repositories are expected to be separate projects:

```text
open-proofline/web-client
open-proofline/ios-client
open-proofline/android-client
open-proofline/protocol
```

Those repositories do not exist in this repository and should not be implemented here by accident. This server repository may keep planning notes for client and protocol work only while the split is being designed.

## Current Backend Scope

Proofline Server receives already-encrypted chunks, stores metadata in SQLite by default or optional PostgreSQL, stores encrypted blobs on local disk by default or in optional S3-compatible object storage, exposes private coarse liveness/readiness checks, performs a startup check against optional Valkey/Redis-compatible coordination when explicitly configured, groups chunks into media streams, serves a private admin web surface under `/admin`, and exposes a token-scoped read-only incident viewer. The Go simulator can produce the documented v1 client-side encryption envelope for development and test flows.

The planned production-cluster scope is additive: SQLite and local filesystem
storage remain supported, optional PostgreSQL metadata can store incident
metadata, optional S3-compatible object storage can store committed encrypted
chunks, and optional Valkey/Redis-compatible coordination can be configured for
startup-checked short-lived coordination. Current backend selector
configuration accepts only implemented values. See
[production-cluster-scope.md](production-cluster-scope.md).

The PostgreSQL metadata path is implemented for explicit opt-in configuration;
see [postgresql-metadata-migration.md](postgresql-metadata-migration.md). It
does not change the current SQLite default or perform automatic
SQLite-to-PostgreSQL data migration.

The future cluster-safe upload operation path is a planning design only; see
[cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md). It does
not implement idempotency keys, upload operations, resumable uploads, or
operation-level use of Valkey/Redis-compatible coordination.
The resumable upload and upload lease path is also planning-only; see
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md). It
defers resumable uploads and leases for a local desktop recorder simulator
	client and keeps the current complete encrypted chunk upload contract. The
	future desktop simulator should include adjustable poor-network simulation and
	use the current local account/session flow unless a later client protocol
	replaces it.

The long-term Proofline product direction is broader than emergency-only recording. Future clients should support emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes while keeping capture, escalation, sharing, and legal/export actions separate. The planned incident-mode schema, capture-profile, escalation-policy, sharing-state, and migration boundaries are documented in [incident-modes.md](incident-modes.md). Current local account/session behavior and future account-owner, trusted-contact, public-link, admin/operator, and optional escrow access boundaries are documented in [v1-access-control.md](v1-access-control.md).

The future iOS incident-capture prototype is planned in [ios-local-recorder-prototype.md](ios-local-recorder-prototype.md). Future production key custody is documented in [key-custody.md](key-custody.md), with a simulator-only contact-wrapped key metadata prototype in [contact-wrapped-key-metadata-simulator.md](contact-wrapped-key-metadata-simulator.md), browser decryption and break-glass follow-up designs in [browser-decryption.md](browser-decryption.md) and [break-glass-key-access.md](break-glass-key-access.md), and live or partial stream access boundaries in [live-partial-stream-access-boundary.md](live-partial-stream-access-boundary.md). None of those future designs make the current private `/v1` API or `/admin` surface safe for broad public exposure.

Evidence bundles are encrypted chunk bundles with JSON manifests. They are not decrypted, playable, or merged media exports.

Retention, backup, and deletion policy is documented in
[retention-backup-deletion.md](retention-backup-deletion.md), with the future
incident deletion and retention enforcement design in
[incident-deletion-retention-enforcement.md](incident-deletion-retention-enforcement.md).
The backend does not yet implement automatic expiration or incident deletion
APIs.

Cluster-style backup, restore, and failure handling for optional PostgreSQL,
S3-compatible blob storage, and Valkey/Redis-compatible coordination is
documented in
[cluster-backup-restore-runbook.md](cluster-backup-restore-runbook.md). That
runbook is operational guidance only and does not make Proofline
production-ready public infrastructure.

## Security Reminder

The private `/v1` API and `/admin` web surface use local account sessions but are still not public product surfaces. Keep them behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. Separate private/public bind addresses reduce accidental exposure, but they are not a complete security model.
