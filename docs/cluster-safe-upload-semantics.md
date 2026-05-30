# Cluster-Safe Upload Operation Semantics

This document designs future cluster-safe upload operation semantics for
Proofline Server.

It is a planning document only. It does not implement idempotency, upload
leases, resumable uploads, operation-level use of Valkey/Redis-compatible
coordination, changes to the current local account/session model, public `/v1`
exposure, browser decryption, backend decryption, key custody, or playable
media export.

## Current Behavior

The current backend accepts one complete encrypted chunk in a multipart upload,
streams it to local temporary storage while computing SHA-256, verifies the
client-provided ciphertext hash, commits the file to an immutable
server-controlled path, and inserts chunk metadata into the configured metadata
backend.

Accepted chunks are immutable. Duplicate chunk identities currently return
`409 duplicate_chunk`, and the storage layer refuses to overwrite an existing
committed blob.

That local-first behavior remains supported. Cluster-safe upload semantics are
future additive work for optional PostgreSQL metadata and S3-compatible object
storage backends.

## Goals

- Allow clients and API nodes to retry ambiguous chunk uploads safely.
- Prevent duplicate committed evidence for the same chunk identity.
- Distinguish equivalent retries from conflicting duplicate attempts.
- Keep committed encrypted chunks immutable and never overwrite evidence.
- Keep PostgreSQL metadata and object-storage blobs consistent enough for
  bundle reconstruction.
- Keep Valkey/Redis-compatible coordination optional and non-durable.
- Preserve the current ciphertext-only backend posture.

## Non-Goals

- No implementation in this design task.
- No resumable upload protocol or partial committed chunks.
- No duplicate-chunk reconciliation API implementation. The client-facing
  reconciliation design is documented in [api.md](api.md).
- No changes to optional PostgreSQL metadata, no changes to the optional
  S3-compatible storage backend, and no operation-level Valkey coordination
  behavior.
- No public `/v1` exposure, public account workflows, or changes to the current
  local account/session model.
- No client repository, protocol repository, or mobile implementation.
- No backend decryption, raw server-held keys, key escrow, key sharing, or
  playable media export.

## Terminology

Chunk identity is the final evidence identity enforced by metadata uniqueness
constraints.

Upload operation identity is the retryable write operation identity used while
one client attempt is being staged, committed, confirmed, or replayed.

Request fingerprint is the immutable metadata and ciphertext summary that must
match for a retry to be considered equivalent.

Equivalent retry success means the same intended chunk is already committed and
its stored metadata matches the retry request, so the server can return success
without writing a second blob or row.

Conflict means the same chunk identity or idempotency key is being used with
different ciphertext or immutable metadata.

## Chunk Identity

Cluster implementations must preserve the existing chunk identities.

Streamed chunks:

```text
incident_id + stream_id + chunk_index
```

Legacy unstreamed chunks:

```text
incident_id + media_type + chunk_index
```

For streamed chunks, `media_type` is still part of validation and the request
fingerprint because the request media type must match the stream media type. It
is not the uniqueness key for streamed chunks because streams already provide a
media-specific namespace.

For legacy unstreamed chunks, `stream_id` is absent and `chunk_index = 0`
remains valid for compatibility.

PostgreSQL uniqueness constraints should be equivalent to or stronger than the
current SQLite behavior:

- one unique constraint or partial index for `(incident_id, stream_id,
  chunk_index)` where `stream_id IS NOT NULL`
- one unique constraint or partial index for `(incident_id, media_type,
  chunk_index)` where `stream_id IS NULL`

These constraints are the final duplicate guard. Idempotency state helps
produce retry-safe responses, but it must not replace durable chunk uniqueness.

## Upload Operation Identity

Future clients should send a stable idempotency key for each intended chunk
upload. The preferred API shape is an `Idempotency-Key` header on
`POST /v1/incidents/{incident_id}/chunks`, though a future protocol design may
choose an equivalent multipart field if that is easier for mobile clients.

The server should bind the idempotency key to:

- route operation, currently `upload_chunk`
- incident ID
- streamed chunk identity or legacy unstreamed chunk identity
- request fingerprint

Idempotency keys should be treated as token-like request identifiers: do not
log raw values, do not include raw values in public errors, and store a stable
hash instead of the raw key when durable lookup is needed.
Future reverse-proxy, tracing, metrics, and error-reporting guidance must also
avoid recording raw `Idempotency-Key` values or using them as labels,
attributes, object keys, or log fields.

Clients without an idempotency key remain supported through the existing chunk
identity constraints. Without an idempotency key, the server can still return
equivalent success when the committed chunk already matches the request, but it
cannot safely identify or resume an in-progress operation before the final
chunk row exists.

## Request Fingerprint

The request fingerprint should include all client-controlled fields that become
immutable chunk metadata or verify the ciphertext:

- normalized chunk identity
- `media_type`
- `started_at`
- `ended_at`
- normalized `original_filename`, including empty value
- uploaded ciphertext byte size
- `sha256_hex` over the complete uploaded ciphertext bytes

The fingerprint must not include server-generated fields such as chunk ID,
stored path, creation time, operation row ID, or object-storage staging key.

A retry is equivalent only when the chunk identity and request fingerprint
match. A different ciphertext hash, byte size, time range, media type, or
original filename is a conflict, even if the chunk identity is the same.

## Durable Idempotency State

Durable upload operation state belongs in the metadata backend. For the planned
cluster backend, PostgreSQL should be the source of truth for idempotency and
upload operation records.

Valkey/Redis-compatible coordination may hold leases, in-progress hints, or
cached results, but it must not be the durable source of truth for committed
chunk metadata, committed chunk bytes, or completed idempotency decisions.

A future logical `upload_operations` table should track at least:

- operation ID generated by the server
- hashed idempotency key, when supplied
- operation type, currently `upload_chunk`
- normalized chunk identity fields
- request fingerprint fields or a stable fingerprint hash
- state, such as `reserved`, `staging`, `blob_committed`,
  `metadata_committed`, `failed_retryable`, or `failed_conflict`
- final chunk ID when metadata has been committed or confirmed
- final stored path or object key when known
- object-storage version, ETag, or equivalent ownership proof when available
- expiry or cleanup eligibility timestamp for abandoned staging state
- timestamps for creation and last update

Completed chunk metadata remains the durable evidence index. Idempotency rows
may support a retry window or operational audit trail, but expiring an
idempotency row must not remove committed chunk rows or committed blobs.

## Commit Flow

A future cluster-safe upload should use this ordering.

1. Validate route parameters and multipart metadata enough to determine the
   normalized chunk identity and request fingerprint inputs.
2. Reserve or find the upload operation in PostgreSQL when an idempotency key
   is supplied.
3. If the same idempotency key already completed with the same fingerprint,
   return equivalent success using the committed chunk metadata.
4. If the same idempotency key exists with a different chunk identity or
   request fingerprint, return `409 idempotency_conflict`.
5. Stage uploaded ciphertext under an operation-specific staging key while
   computing SHA-256 and enforcing the upload byte limit.
6. Compare the computed SHA-256 with `sha256_hex`. On mismatch, remove staging
   bytes and return `400 hash_mismatch`.
7. Recheck incident and stream state in the metadata backend before final blob
   commit. Streamed uploads must still require an open stream with matching
   media type.
8. Commit the staged object to the final server-controlled immutable object key
   using conditional no-overwrite behavior.
9. Insert chunk metadata in PostgreSQL, or confirm the existing chunk row when
   a race already inserted an equivalent row.
10. Mark the upload operation `metadata_committed` with the final chunk ID.
11. Remove operation-specific staging state.
12. Return success only after final encrypted bytes exist outside staging and
   chunk metadata has been inserted or confirmed.

The final object key should remain server-controlled and derived from the
normalized chunk identity. Clients must never provide stored paths or final
object keys.

## Equivalent Success

Equivalent success is allowed when all of the following are true:

- the normalized chunk identity matches
- the existing chunk row belongs to the same incident and, for streamed chunks,
  the same stream
- the existing chunk row has the same request fingerprint
- the committed blob exists and matches the stored byte size and `sha256_hex`
  when the backend can verify this cheaply or as part of the operation

The recommended future HTTP behavior is:

- `201 Created` when this request created the chunk row
- `200 OK` when a retry or racing request confirms an already committed
  equivalent chunk
- the same chunk metadata shape for both responses
- an optional response header, such as `Idempotency-Replayed: true`, for
  clients that need to distinguish replayed success without changing the JSON
  body

The exact response contract should be documented in `docs/api.md` when the
feature is implemented.

## Conflicts

The server must return a conflict and must not overwrite evidence when:

- the same idempotency key is reused for a different chunk identity
- the same idempotency key is reused with a different request fingerprint
- the same chunk identity already exists with a different ciphertext hash
- the same chunk identity already exists with different immutable metadata
- a final object already exists but cannot be proven equivalent to the request
  and committed metadata

Recommended future error codes:

- `409 idempotency_conflict` for idempotency-key reuse with different inputs
- `409 duplicate_chunk` or a future duplicate-reconciliation-specific code for
  same chunk identity but different ciphertext or metadata
- `409 upload_in_progress` with `Retry-After` when an equivalent operation is
  still actively staging or committing and no committed chunk row is available

Do not include uploaded bytes, plaintext, raw keys, raw tokens, raw idempotency
keys, request bodies, final stored paths, staging paths, or object-storage
credentials in conflict responses.

## Object Storage Commit Rules

S3-compatible storage should use conditional writes or an equivalent
no-overwrite primitive for final objects. A final object write that would
replace existing evidence must fail.

For object stores that support object versioning, ETags, conditional deletes,
or user metadata, the implementation should record enough server-controlled
ownership proof to decide whether this operation created a final object that is
safe to remove after a later metadata failure.

If a final object exists without a committed chunk row, the system should treat
that state as ambiguous. It may repair or clean up the object only when
operation state and object metadata prove ownership and no committed chunk row
references it. Otherwise, it should fail closed with a retryable operational
error rather than overwrite or delete possible evidence.

## Cleanup

Cleanup must distinguish staging state from committed evidence.

Safe cleanup targets:

- local temporary upload files
- operation-specific object-storage staging keys
- expired upload operation rows that have no committed chunk row
- final objects created by a failed operation only when ownership proof shows
  this operation created the object and no committed chunk row references it

Unsafe cleanup targets:

- committed chunk rows
- committed final blobs referenced by chunk metadata
- final blobs that cannot be tied safely to a failed operation
- any object path or stored path provided by a client

Cleanup should be conservative. Leaving a server-owned staging object for a
later lifecycle rule is preferable to deleting evidence that might have been
committed by another API node.

## Relationship To Other Issues

Issue `#85`, "Design Duplicate Chunk Reconciliation API", chooses a separate
private query workflow for already committed duplicate chunk identities. The
planned route compares a client's expected ciphertext hash and immutable
metadata with an accepted chunk row without re-uploading ciphertext, overwriting
evidence, or exposing bytes, stored paths, raw tokens, plaintext, or keys. This
cluster-safe upload design can still add idempotency-key equivalent success on
the upload route later; duplicate reconciliation remains the fallback for
clients that only know the final chunk identity and expected fingerprint.

Issue `#86`, "Plan Resumable Upload And Upload Lease Protocol", should decide
whether incomplete transfers need explicit leases, resumable multipart upload,
or a simpler complete-chunk retry model. This document does not make partial
uploads visible as committed evidence and does not design a resumable transfer
protocol. That follow-up decision is documented in
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md), which
defers resumable uploads and upload leases for a local desktop recorder
simulator client.

This document focuses on the commit and confirmation semantics once an upload
attempt has enough data to compute the ciphertext hash and decide whether a
final chunk can be committed.

## Required Future Work

API documentation:

- implement and document the duplicate-chunk reconciliation route designed in
  [api.md](api.md)
- document the idempotency key location and allowed format
- document equivalent success status codes and replay headers
- document idempotency conflict and in-progress responses
- document that raw idempotency keys are not logged or returned
- document reverse-proxy, tracing, metrics, and error-reporting redaction for
  raw idempotency keys
- document the relationship between duplicate reconciliation and idempotent
  retry success

Backend tests:

- repository contract tests for streamed and legacy uniqueness
- repository tests for idempotency reservation, replay, conflict, expiry, and
  operation-state transitions
- HTTP tests for equivalent retry success with the same idempotency key
- HTTP tests for idempotency-key reuse with different metadata
- HTTP tests for same chunk identity with different ciphertext
- HTTP tests for duplicate reconciliation matches and conflicts without
  returning uploaded bytes, stored paths, or conflicting stored values
- blob-store tests for conditional no-overwrite final commits
- cleanup tests proving staging cleanup does not delete committed evidence
- race tests for upload versus incident close and stream completion

Simulator changes:

- generate one idempotency key per intended chunk upload
- retry an upload after simulated ambiguous response loss
- verify equivalent success for the same ciphertext and metadata
- verify conflict behavior for same chunk identity with different ciphertext
- keep the existing hash-mismatch and bundle verification flows

Backend-specific work:

- add PostgreSQL upload-operation tables and migrations
- extend the metadata repository boundary for operation reservation,
  completion, replay lookup, and conflict detection
- extend the blob-store boundary for operation-specific staging and conditional
  final object commit
- add S3-compatible object-storage implementation with no-overwrite final keys
- decide how Valkey/Redis-compatible coordination should be used for leases or
  reducing duplicate in-progress work
- update deployment, backup, restore, security, threat-model, retention, and
  code-map docs before recommending production cluster deployment
