# Incident Deletion And Retention Enforcement Design

This document designs future incident deletion and retention enforcement for
Proofline Server.

It is a planning document only. It does not implement incident deletion APIs,
CLI commands, background jobs, retention configuration, data deletion,
changes to the current local account/session model, public `/v1` exposure,
public account workflows, backend decryption, key custody, key escrow, or
playable media export.

## Current Behavior

The current backend preserves accepted evidence by default:

- incidents remain until explicit operator action outside the application
- closing an incident stops later uploads but does not delete evidence
- open, complete, and failed media streams remain with the incident
- checkins remain with the incident
- incident viewer token metadata remains as token hashes only, including
  expired and revoked tokens
- encrypted chunk blobs are stored on local disk by default or in optional
  S3-compatible object storage when explicitly configured
- encrypted ZIP bundles are generated on demand and are not persisted by the
  server
- temporary upload files are normally removed after upload success or failure,
  but crash-orphaned temp files need future cleanup

The backend has no incident deletion API and no automatic retention job today.
Manual deletion is an operator action and must keep metadata and encrypted
blobs consistent.

## Goals

- Preserve uploaded evidence unless there is an explicit deletion decision.
- Keep public incident viewer routes read-only.
- Keep deletion entry points private/admin-only.
- Delete encrypted blobs only through server-controlled stored paths from
  metadata, never through client-provided filesystem paths, object keys, or
  object-store URLs.
- Make deletion and retention enforcement idempotent so interrupted work can be
  retried safely.
- Keep metadata, local blobs, and S3-compatible blobs consistent enough that a
  partially failed deletion is visible, retryable, and fails closed.
- Treat open and failed streams as possible evidence unless a deletion decision
  explicitly covers them.
- Define how retention windows interact with closed incidents, token metadata,
  backups, and future incident modes.
- Record non-sensitive audit information without logging raw viewer tokens,
  request bodies, uploaded bytes, plaintext, raw keys, or sensitive evidence
  metadata.

## Non-Goals

- No implementation in this design task.
- No deletion routes, CLI commands, repository methods, migrations, workers, or
  background scheduler in this issue.
- No deletion of local test data, database rows, committed blobs, generated
  bundles, or backups as part of this design task.
- No public admin routes, public `/v1` exposure, OAuth, JWT, public account
  workflows, cloud service automation, Docker Compose, Kubernetes, or public
  dashboard.
- No promise of unrecoverable secure erasure from normal file, object, or
  database row deletion.
- No backend decryption, raw server-held media keys, key escrow, key sharing,
  browser decryption, or key-custody change.
- No attempt to delete copies already downloaded by browsers, trusted contacts,
  operators, reverse proxies, backup systems, or endpoint devices.

## Deletion Decisions

Every incident deletion must begin with an explicit deletion decision. A
decision may be created by a future private/admin route, an authenticated
private admin API, or a local operator CLI. It must not be created by public
incident viewer routes.

A deletion decision should record:

- target incident ID
- requested action, such as `delete_incident`
- decision source, such as `manual_admin`, `retention_policy`, or
  `operator_cli`
- reason code or policy ID, not free-form sensitive evidence details
- whether deleting an open incident is explicitly allowed
- requested time and the non-secret actor or process identifier
- current deletion state
- safe counts or summaries needed for review

Do not record raw viewer tokens, Authorization headers, request bodies,
uploaded bytes, plaintext, raw keys, precise stored paths, original filenames,
location values, notes, private deployment details, or object-store URLs in
operator-facing audit logs.

Open incidents should not be auto-deleted by retention policy. A manual
deletion decision may delete an open incident only when it explicitly says that
open evidence is covered. This prevents accidental loss while an upload session
is still active or ambiguous.

## Tombstone And Hard-Delete Model

Deletion should use a two-phase tombstone and hard-delete model.

The metadata backend should create durable deletion state before any blob is
removed. That state should snapshot the server-controlled stored paths that
need deletion and mark the incident as deletion-pending so normal readers fail
closed while deletion is in progress.

Recommended states:

| State | Meaning |
|---|---|
| `active` | Incident is not being deleted. |
| `deletion_pending` | A deletion decision exists and deletion items have been prepared. |
| `deleting` | A worker or CLI is deleting encrypted blobs and metadata. |
| `deletion_failed` | Deletion stopped with retryable failures. |
| `deleted` | Encrypted blobs and sensitive child metadata have been removed or confirmed absent. |

The current `open` and `closed` incident statuses should keep their existing
meaning. Deletion state should be separate so closing an incident never implies
deletion and deletion does not overload upload state.

After deletion finishes, the backend may retain a minimal tombstone for
idempotency and audit. The tombstone should contain only non-sensitive fields,
such as incident ID, deletion timestamps, state, decision source, reason code,
safe item counts, and backend type summaries. It should not retain checkin
location values, notes, original filenames, raw token data, token hashes, stored
paths, object keys, or evidence-specific metadata longer than needed to finish
the deletion job.

Hard-delete should remove or prune sensitive child rows, including streams,
chunks, checkins, and incident viewer token rows, after the deletion process no
longer needs those rows for retry. If a future deployment needs to keep
tombstones for legal or operational reasons, that retention must be explicit
and documented.

## Consistency And Retry Model

Database transactions cannot atomically delete local filesystem blobs or
S3-compatible objects. Future deletion should therefore use a durable
metadata-backed work queue, sometimes called an outbox pattern.

Recommended incident deletion flow:

1. Validate the private/admin deletion request or retention decision.
2. In one metadata transaction, confirm the incident exists, check whether open
   incident deletion is allowed, create or find the deletion decision, create
   deletion item rows from existing chunk stored paths, and mark the incident
   `deletion_pending`.
3. Return or continue only after durable deletion state exists.
4. Delete each encrypted blob through the storage backend using the
   server-controlled stored path from metadata.
5. Treat a missing blob as idempotent success only for a deletion item that was
   already created from server metadata.
6. Mark each deletion item deleted, or record a retryable error class without
   logging sensitive paths or private deployment details.
7. After all blob deletion items are complete, delete or prune sensitive child
   metadata in one metadata transaction.
8. Mark the tombstone `deleted`.

If the process fails after deletion state is created, retries should resume
from the deletion item table. A repeated request for the same incident should
return the existing deletion state instead of creating a competing deletion
operation. A repeated request for an already deleted incident should be a safe
success or a clear `410 Gone`-style private/admin response, depending on the
future API contract.

While an incident is `deletion_pending`, `deleting`, or `deleted`, public
viewer token lookups should fail closed with the same generic public failure
shape used for invalid, expired, or revoked tokens. Public routes must not
reveal whether a deletion exists.

## Blob Deletion Rules

Blob deletion must be driven by metadata. The future deletion worker, CLI, or
private/admin route must not accept final stored paths from clients.

For local storage:

- stored paths must pass the existing safe relative path checks
- deletion should call the storage boundary with stored paths from metadata
- directory cleanup, if any, should remove only empty server-controlled
  directories below the configured data root
- path traversal, absolute paths, backslashes, and unsafe path segments must
  continue to be rejected

For S3-compatible storage:

- object keys must be derived from stored paths and the configured safe prefix
- deletion must not expose bucket URLs or raw object keys in public responses
  or logs
- object versioning, provider replication, lifecycle snapshots, and backup
  copies mean object deletion is not guaranteed secure erasure
- deployment runbooks must define backup and object lifecycle expiry separately

Blob deletion failures should keep the deletion job retryable. Metadata must
not be hard-deleted before the system has either deleted or explicitly
accounted for every blob deletion item.

## Retention Windows

Retention enforcement should be policy-driven and conservative.

Initial future policy settings should cover:

- closed incident retention window
- whether failed streams inherit the incident retention window
- token metadata audit retention window for expired or revoked token rows
- deletion tombstone retention window
- backup generation retention after an incident deletion
- orphaned temporary upload cleanup age

Open incidents should not be eligible for automatic retention expiry. Closing
an incident may start the retention window, but it must not itself delete
evidence.

Failed streams should be retained with their incident by default. They may
contain useful encrypted chunks even when they are not downloadable as completed
stream bundles. A first implementation should avoid stream-only retention
deletion unless a later issue designs stream-level deletion semantics.

Expired and revoked incident viewer token rows may be pruned after an audit
window. Token pruning must remove only stored token-hash metadata and labels;
raw tokens are not stored. Token pruning must not delete incidents, streams,
chunks, checkins, or blobs.

Backups must be handled as a separate lifecycle. Deleting live metadata and
blobs does not remove older SQLite backups, PostgreSQL backups, S3 object
versions, filesystem snapshots, volume snapshots, reverse-proxy caches, or
downloaded bundles. Operators must configure backup expiry so deleted
incidents eventually age out of backup sets.

Future incident modes may require different defaults:

| Future mode | Retention implication |
|---|---|
| Emergency incident | May need a longer evidence and backup retention window. |
| Interaction record | May need a shorter default and stronger user-controlled deletion policy. |
| Safety check | May need retention tied to check completion, missed check-in policy, and escalation state. |
| Evidence note | May need separate retention for lightweight notes and attached encrypted media. |

No incident-mode-specific retention should be implemented until first-class
incident-mode, capture-profile, escalation-policy, and sharing-state fields
exist and the retention policy is updated before or alongside that
implementation.

## Public And Private Entry Points

The first implementation should prefer a local operator CLI or a private/admin
entry point that is clearly separated from public incident viewer routes.

Deletion entry points must:

- run only on a private/admin surface
- require future admin authentication and authorization when the access-control
  model exists
- never be mounted on the public incident viewer listener
- never accept client-provided stored paths, filesystem paths, object keys, or
  object-store URLs
- support dry-run or preview output before destructive action where practical
- return idempotent status for repeated deletion attempts
- avoid logging raw tokens, request bodies, uploaded bytes, plaintext, raw keys,
  private deployment details, or sensitive evidence metadata

Public incident viewer routes must remain read-only. They should never expose
deletion controls, deletion job status, tombstone details, retention policy, or
blob deletion errors.

## Audit Fields

Future audit records should be useful for operational review without becoming a
new source of secrets or sensitive evidence.

Safe audit fields may include:

- audit event ID
- incident ID
- deletion decision ID
- action name, such as `deletion_requested`, `deletion_started`,
  `blob_delete_failed`, `metadata_pruned`, or `deletion_completed`
- actor type, such as `operator_cli`, `private_admin_api`, or
  `retention_policy`
- reason code or policy ID
- deletion state
- non-sensitive item counts
- metadata backend type and blob backend type
- timestamps

Audit records should avoid:

- raw viewer tokens or future token-like values
- token hashes unless a specific internal audit design requires them
- request bodies
- uploaded bytes
- plaintext
- raw keys
- Authorization headers
- checkin location values
- notes
- original filenames
- stored paths, object keys, bucket names, object URLs, and private endpoints
- user safety narrative or exploit details

Detailed retry state that needs stored paths should remain in internal
deletion item tables, not in public issue drafts, public logs, dashboards, or
operator-facing audit summaries.

## Backup And Restore Interaction

Deletion and retention enforcement must be documented with backup and restore
behavior before real-world use.

Deletion should not claim completion across the full deployment until the
operator understands backup retention. Live deletion can mark the active
metadata and blob backends deleted while older backups still contain recoverable
encrypted evidence.

Restore drills should verify both sides of the lifecycle:

- a restored active incident can still reconstruct completed encrypted bundles
- a restored deleted incident remains deleted or is clearly marked as a
  tombstone
- restoring from an older backup may resurrect data that was deleted after the
  backup was taken, unless backup expiry or key retirement prevents it
- public viewer routes still fail closed for deleted incidents after restore

If a restore reintroduces an incident that was deleted in live state, the
operator must have a documented reconciliation process. That process is
deployment-specific and should not rely on public routes.

## Future Implementation Tasks

Implementing this design should be split into separate issues.

Repository and metadata tasks:

- add metadata schema for deletion decisions, deletion items, deletion state,
  tombstones, and optional retention policy settings
- add repository methods to create or find deletion decisions idempotently
- add repository methods to snapshot deletion items from chunk metadata
- add repository methods to mark deletion items complete or retryable
- add repository methods to prune sensitive child metadata after blob deletion
- add repository methods to select retention-eligible closed incidents and
  token rows
- preserve SQLite support and optional PostgreSQL support

Storage tasks:

- add deletion helpers that operate only on server-stored relative paths
- define idempotent handling for already-missing local files and S3-compatible
  objects
- keep local and S3 path validation consistent with existing storage safety
  checks
- avoid exposing object-store keys, bucket URLs, private endpoints, or local
  filesystem paths in logs and responses

Private/admin or CLI tasks:

- add a local operator CLI or private/admin endpoint to request incident
  deletion
- add dry-run output for retention candidates and deletion previews
- add status output for deletion jobs without exposing sensitive evidence
  metadata
- keep all deletion controls off the public incident viewer listener

Retention tasks:

- add explicit retention settings for closed incidents, token metadata,
  tombstones, and orphaned temporary upload files
- add a manual or scheduled retention runner only after dry-run behavior and
  operator review are documented
- defer incident-mode-specific retention until first-class incident-mode,
  capture-profile, escalation-policy, and sharing-state fields exist

Test tasks:

- test that public incident viewer routes remain read-only
- test that public token failures do not reveal deletion state
- test idempotent retry after failures before blob deletion, after partial blob
  deletion, and after metadata pruning
- test local and S3-compatible stored path safety
- test open incident deletion guard behavior
- test failed stream retention and deletion with the parent incident
- test token metadata pruning without incident deletion
- test backup and restore documentation examples where practical

Documentation tasks:

- update `docs/api.md` when any private/admin route or CLI behavior exists
- update `docs/security-model.md` and `docs/threat-model.md` for implemented
  deletion behavior
- update `docs/retention-backup-deletion.md` with concrete configuration and
  operator steps
- update deployment and backup runbooks for live deletion, backup expiry, and
  restore reconciliation
- document that normal deletion is not guaranteed secure erasure

These tasks are intentionally future work and are not implemented by this
planning issue.
