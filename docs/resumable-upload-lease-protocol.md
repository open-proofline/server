# Resumable Upload And Upload Lease Protocol

This document plans whether Proofline Server needs explicit resumable uploads
or upload leases for partially sent encrypted chunks.

It is a planning document only. It does not implement resumable uploads, upload
leases, upload operations, idempotency keys, PostgreSQL, S3-compatible object
storage, Valkey/Redis-compatible coordination, public `/v1` authentication,
account management, browser decryption, backend decryption, key custody, or
playable media export.

## Decision

Do not add a resumable-upload or upload-lease protocol before a local desktop
recorder simulator client is added.

A local desktop recorder simulator client should keep the current
complete-chunk upload contract: record short media intervals, encrypt each
completed interval locally, stage the encrypted envelope durably on the client,
and retry the whole encrypted chunk when an upload is interrupted or ambiguous.

This keeps the next recorder milestone inside the server repository's simulator
and reference-flow boundary. It also avoids making partially uploaded bytes
visible as evidence before the project has production cluster storage, durable
upload operation state, or public `/v1` access control.

Revisit explicit resumability after a desktop recorder simulator has measured
real local capture chunk sizes, retry cost, interruption behavior, and local
storage pressure. Revalidate the decision again later before native iOS or
Android recorder work.

## Current Behavior

The current backend accepts one complete encrypted chunk in
`POST /v1/incidents/{incident_id}/chunks`.

The handler:

1. Streams one multipart `file` part into local temporary storage.
2. Computes SHA-256 while writing the temporary file.
3. Enforces `SAFE_MAX_UPLOAD_BYTES` on the uploaded file bytes.
4. Compares the computed ciphertext hash with `sha256_hex`.
5. Validates incident and stream state.
6. Commits the verified temp file to a server-controlled immutable blob path.
7. Inserts chunk metadata in SQLite.

An accepted chunk is an immutable ciphertext blob. Streamed chunk identity is
`(incident_id, stream_id, chunk_index)`. Legacy unstreamed chunk identity is
`(incident_id, media_type, chunk_index)`.

The current server does not expose partially uploaded bytes. An interrupted or
invalid request must not create a chunk row, final blob, bundle entry, or
viewer-visible evidence item.

## Desktop Recorder Simulator Guidance

For a local desktop recorder simulator client, use the existing complete-chunk
contract.

Client behavior:

- keep chunks short, with the current audio-first default around 5 seconds
- encrypt each finalized chunk before upload
- compute `sha256_hex` over the complete encrypted envelope bytes
- persist encrypted staged chunks and immutable upload metadata locally
- expose adjustable poor-network simulation controls for latency, jitter,
  request timeouts, bandwidth ceilings, intermittent offline windows, upload
  failure rates, and process restart or resume drills
- retry failed uploads by resending the complete encrypted chunk
- keep retries ordered by stream index unless later testing justifies
  out-of-order upload behavior
- complete a stream only after chunks `1..expected_chunk_count` are locally
  marked uploaded
- fail the stream if the client cannot produce a contiguous sequence
- be ready for near-term account-aware flows once an account and access-control
  model exists; until then, account identity should remain local test metadata
  only

Server behavior stays unchanged for this simulator client:

- no resumable upload routes
- no upload lease routes
- no server-visible client queue summary endpoint
- no partial upload commit state
- no public `/v1` exposure
- no account-management routes, OAuth, JWT, or user account model added only for
  simulator scaffolding
- no backend decryption or server-held media keys

This should be enough to test the core recorder loop: local capture,
encryption, durable staging, complete encrypted chunk upload, retry, stream
completion, bundle verification, poor-network recovery, and future account-flow
shape without changing the current backend contract.

## When To Reconsider

Explicit resumable uploads or upload leases become worth reconsidering when
one or more of these are true:

- desktop recorder simulator measurements show complete encrypted chunk
  retries are too expensive for normal local recording interruptions
- desktop sleep, process restart, network loss, or other local interruptions
  regularly interrupt chunks before the complete encrypted envelope can be sent
- recorded audio or video chunks become too large to retry whole under
  practical network conditions
- later iOS or Android lifecycle testing shows mobile foreground, background,
  or upload limits regularly interrupt complete encrypted chunk uploads
- video chunks become too large to retry whole under practical mobile network
  conditions
- future production clients need server-visible upload progress for operator
  support or protocol conformance
- optional S3-compatible object storage needs server-owned multipart staging
  and lifecycle cleanup
- PostgreSQL upload operation state exists and can durably track in-progress
  upload attempts
- Valkey/Redis-compatible coordination is available for short-lived leases,
  while PostgreSQL remains the durable source of truth

Do not add resumability only because an upload can fail. Whole-chunk retry is
the simpler recovery path while chunks are short and clients keep encrypted
staging locally.

## Possible Future Protocol

If future measurements justify resumability, prefer a narrow private `/v1`
protocol that stages encrypted bytes under a server-generated upload session.

A future resumable protocol should keep these properties:

- the final committed chunk identity remains the existing streamed or legacy
  chunk identity
- the client declares immutable chunk metadata before or during session
  creation
- the server returns an opaque upload session ID or lease ID
- raw lease IDs and idempotency keys are treated as token-like values and are
  not logged
- uploaded byte ranges are staged under server-controlled temporary locations
- the server computes or verifies SHA-256 over the complete ciphertext before
  final commit
- the server commits final encrypted bytes only with no-overwrite behavior
- chunk metadata is inserted or confirmed only after final encrypted bytes are
  committed
- incomplete sessions expire without creating evidence
- cleanup never deletes committed chunk rows or final blobs referenced by
  metadata

Possible private route shape, subject to later design:

```text
POST /v1/incidents/{incident_id}/uploads
PATCH /v1/incidents/{incident_id}/uploads/{upload_session_id}
POST /v1/incidents/{incident_id}/uploads/{upload_session_id}/commit
DELETE /v1/incidents/{incident_id}/uploads/{upload_session_id}
```

This route shape is intentionally not part of the current API. If implemented,
it must be documented in `docs/api.md` with exact request and response
contracts, error codes, cleanup behavior, and logging rules.

## Upload Leases

Upload leases should be treated as short-lived coordination for in-progress
staging, not as authentication or authorization.

A lease may eventually help one API node, client, or worker decide who is
actively staging a specific upload attempt. A lease must not grant access to an
incident, expose `/v1` publicly, authorize a trusted contact, or replace future
account-level access control.

Lease state may be useful in Valkey/Redis-compatible coordination, but durable
upload decisions belong in the metadata backend. Expiring a lease may make an
in-progress staging attempt eligible for cleanup, but it must never delete
committed chunk metadata or committed encrypted blobs.

## Client Queue Summaries

Do not add a server-visible client queue summary endpoint for the local desktop
recorder simulator client.

The simulator client should keep queue state local: pending chunk identities,
ciphertext hashes, byte sizes, attempt counts, retry categories, and local
status. That local state is enough to resume after process restart and decide
when to retry complete encrypted chunk uploads.

A future private queue-summary endpoint may be useful for telemetry or support,
but it should be designed separately from resumable upload commit semantics.
Such an endpoint should be metadata-only, should not accept local file paths,
should not expose uploaded bytes or plaintext, and should not become public
`/v1` authentication or a trusted-contact access model.

## Size Limits And Expiry

The current complete-chunk API applies `SAFE_MAX_UPLOAD_BYTES` to the uploaded
file bytes. A future resumable protocol should preserve a final ciphertext size
limit at least as strict as the current complete upload limit unless an
explicit configuration change is designed and documented.

For a future resumable session:

- the declared or accumulated total ciphertext size must stay within the
  configured upload limit
- range or part uploads may have additional per-request limits, but those
  limits must not allow a final oversized chunk to be committed
- expiry should apply to incomplete upload sessions and short-lived leases
- expiry should make only temporary staging eligible for cleanup
- expiring a session or lease must not remove a committed chunk row, final
  blob, or evidence bundle content

## Cleanup Rules

Cleanup must distinguish temporary staging from committed evidence.

Current local cleanup:

- request-path errors remove the current temporary upload when the handler can
  do so safely
- hash mismatches do not commit final files
- duplicate uploads do not overwrite existing committed chunks
- normal handler cleanup removes uncommitted temp files
- a process crash can still leave an unreferenced local temp file under the
  server temp directory

Future cleanup for resumable uploads or leases should allow:

- removing expired local temp files that are not referenced by committed chunk
  metadata
- removing expired operation-specific object-storage staging keys
- expiring upload session or lease rows that have no committed chunk row
- keeping conservative lifecycle cleanup for staging objects when ownership is
  ambiguous

Future cleanup must not remove:

- committed chunk rows
- final blobs referenced by committed chunk metadata
- final object keys or stored paths provided by a client
- evidence bundle contents
- any object that cannot be proven to be temporary staging

Leaving stale staging bytes is preferable to deleting possible evidence.

## Sensitive Data Rules

Resumable upload and lease designs must not expose or log:

- raw request bodies
- uploaded bytes
- plaintext
- raw media keys
- raw viewer or incident tokens
- raw idempotency keys
- raw upload session or lease IDs
- Authorization headers
- local filesystem paths
- staging paths
- final stored paths
- object-storage credentials

Safe responses may include normalized non-secret chunk identity, accepted byte
counts for the active session, expected total ciphertext size, non-secret
status values, retry timing hints, and server-generated timestamps. They should
not include stored-path internals or existing conflicting ciphertext metadata
beyond safe field names.

## Relationship To Other Upload Designs

Duplicate chunk reconciliation is about already committed chunk identities. It
lets a client compare expected ciphertext hash and immutable metadata after a
`409 duplicate_chunk` or ambiguous retry outcome. It does not resume an
in-progress transfer.

Cluster-safe upload operation semantics are about idempotent commit and retry
success once the server has enough bytes and metadata to decide whether the
final encrypted chunk can be committed or confirmed. They do not require a
byte-range resumable upload protocol.

Resumable upload and upload leases are about in-progress transfer recovery
before the complete ciphertext exists on the server. They should build on the
same immutable commit rules, but they can be deferred until whole-chunk retry
is proven insufficient.

## Required Future Work

If resumable uploads or leases are later implemented, update documentation:

- `docs/api.md` with private route contracts, error codes, headers, and
  response examples
- `docs/simulator.md` with local desktop recorder simulator queue and retry
  mapping
- `docs/ios-local-recorder-prototype.md` with later mobile client queue and
  retry mapping, if mobile work is affected
- `docs/cluster-safe-upload-semantics.md` with the relationship between
  upload operations and byte-range staging
- `docs/production-cluster-scope.md` with object-storage and coordination
  cleanup expectations
- `docs/security-model.md` and `docs/threat-model.md` with lease/session
  logging rules, cleanup boundaries, and remaining gaps
- `docs/code-map.md` after implementation changes the upload flow

Add backend tests for:

- creating upload sessions without exposing filesystem paths
- rejecting invalid immutable metadata before staging bytes
- appending or replacing ranges only according to the chosen protocol
- enforcing total ciphertext size limits
- computing or verifying final SHA-256 over the complete ciphertext
- expiring incomplete sessions without committed evidence
- cleaning temporary staging without deleting committed chunks
- refusing commit after incident close or stream completion
- preserving no-overwrite behavior for duplicate final chunk identities
- avoiding raw session IDs, tokens, request bodies, uploaded bytes, plaintext,
  raw keys, and paths in logs and responses

Add simulator coverage only after the backend feature exists:

- interrupted upload followed by resumable completion
- expired upload session followed by whole-chunk retry
- ambiguous retry that uses idempotency or duplicate reconciliation
- hash mismatch at final commit
- proof that downloaded bundles still contain only committed encrypted chunks

Until then, the simulator should keep using complete encrypted chunk uploads.

## Out Of Scope

- Implementing resumable upload routes.
- Implementing upload leases.
- Adding public `/v1` authentication or exposing `/v1` publicly.
- Adding web, iOS, Android, or protocol repository code.
- Adding PostgreSQL, S3-compatible object storage, Valkey, or background
  workers.
- Adding backend decryption, raw server-held keys, key escrow, key sharing, or
  playable media export.
- Allowing partial files to appear as committed evidence.
- Weakening upload hash validation.
- Deleting committed chunks as part of temporary upload cleanup.
