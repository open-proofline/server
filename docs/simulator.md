# Simulator

The simulator CLI lives at `cmd/simclient`. It exercises the current Proofline ingest flow that a future recording client is expected to use. It logs in to the private `/v1` API with a local account session, then encrypts fake chunk plaintext with the v1 client-side envelope before upload by default. Each intended chunk upload includes a stable `Idempotency-Key`, and the simulator verifies one equivalent replay without printing the raw key.

The simulator covers generic incidents only. It does not implement planned incident modes such as emergency incidents, interaction records, safety checks, or evidence notes.

## Future Desktop Recorder Simulator

A future local desktop recorder simulator client may be added in this
repository as a backend reference flow. It should remain a simulator, not a
production desktop app or a replacement for planned mobile clients.

That simulator should use the current complete encrypted chunk upload contract:
capture short local test intervals, encrypt each completed chunk, stage
encrypted chunks locally, retry failed uploads by resending complete chunks, and
complete or fail streams through the existing private `/v1` routes.

It should include adjustable poor-network simulation rather than one fixed
failure mode. Useful controls include latency, jitter, request timeouts,
bandwidth ceilings, intermittent offline windows, upload failure rates, and
process restart or resume drills. Those controls should exercise local staging,
retry scheduling, and stream completion behavior without requiring partially
uploaded bytes to become server-visible evidence.

The desktop simulator should continue using account-aware flows without
turning this repository into a production desktop app. Simulator credentials
are local development credentials only. Future simulator work must not
incidentally add OAuth, JWT, public `/v1` exposure, browser decryption, mobile
client behavior, or a public account portal.

The simulator may also prototype contact-wrapped key metadata in local
development artifacts. That design is documented in
[contact-wrapped-key-metadata-simulator.md](contact-wrapped-key-metadata-simulator.md)
and must keep raw media keys, contact private keys, plaintext, and decryption
capabilities out of server storage, logs, and bundle manifests unless a later
explicit production key-custody task changes the boundary.

Do not add resumable uploads, upload leases, or server-visible queue summary
routes just to support that simulator. The resumable-upload decision is planned
separately in
[resumable-upload-lease-protocol.md](resumable-upload-lease-protocol.md).

## Basic Flow

Start the backend first:

```bash
SAFE_AUTH_BOOTSTRAP_SECRET='replace-with-local-bootstrap-secret' \
go run ./cmd/api
```

For a new local database, create an admin account through
`POST /v1/bootstrap/admin`, then remove `SAFE_AUTH_BOOTSTRAP_SECRET` and
restart the server. See [deployment](deployment.md) for the bootstrap flow.

Then run:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --chunks 12 --interval 5s
```

The simulator prints an incident viewer URL. Open it to watch incident metadata update.

## Bundle Download Flow

To test encrypted bundle download through the incident viewer:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

This creates a media stream, uploads encrypted chunks with `stream_id`, completes the stream, downloads the completed encrypted ZIP bundle through the incident viewer, and verifies local decryption.

## Encryption

Encryption is enabled by default:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

The simulator prints a non-secret `key_id`, but it never prints the raw key or decrypted plaintext.

To reuse a simulator key across runs:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle --key-file /tmp/proofline-sim.key.json
```

If the key file exists, the simulator loads it. If it does not exist, the simulator creates it with restrictive permissions where practical. Do not upload or commit simulator key files.

Older examples may use `/tmp/safety-recorder-sim.key.json`; the file name is not part of the encryption protocol.

To preserve the old raw fake chunk behavior:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --encrypt=false
```

This is only for development compatibility. See [encryption.md](encryption.md) for the envelope and key file format.

## Failure And Retry Flow

To test hash failure and retry behavior:

```bash
PROOFLINE_SIM_USERNAME=admin \
PROOFLINE_SIM_PASSWORD='replace-with-a-long-local-password' \
go run ./cmd/simclient --chunks 12 --interval 2s --simulate-failure-every 4
```

Every fourth chunk intentionally fails SHA-256 verification before being
retried. Hash-mismatch attempts do not reserve idempotency state because the
server has not accepted the immutable fingerprint. The first successfully
uploaded chunk is then resent with the same `Idempotency-Key` to verify
equivalent retry success.

The simulator does not yet call the duplicate chunk reconciliation route. Future
ambiguous-network and process-restart drills can use
`POST /v1/incidents/{incident_id}/chunks/reconcile` to compare a local expected
chunk fingerprint with accepted server metadata without re-uploading ciphertext.
Broader simulator drills remain future work planned in
[cluster-safe-upload-semantics.md](cluster-safe-upload-semantics.md).

## Useful Flags

| Flag | Purpose |
|---|---|
| `--api` | Private API base URL. |
| `--viewer` | Incident viewer base URL. |
| `--username` | Proofline account username. Defaults to `PROOFLINE_SIM_USERNAME`. |
| `--password` | Proofline account password. Defaults to `PROOFLINE_SIM_PASSWORD`. |
| `--chunks` | Number of chunks to upload. |
| `--interval` | Delay between chunk uploads. |
| `--chunk-size` | Size of each fake plaintext chunk before optional encryption. |
| `--media-type` | Media type to upload. |
| `--complete-stream` | Mark the uploaded media stream complete. |
| `--download-bundle` | Download the completed stream bundle through the incident viewer. |
| `--encrypt` | Encrypt simulated chunk bytes before upload. Defaults to `true`. |
| `--key-file` | Optional local simulator key file. |
| `--verify-bundle-decryption` | Locally decrypt downloaded bundles when encryption is enabled. |
| `--simulate-failure-every` | Intentionally fail every Nth chunk hash before retrying. |
| `--close` | Close the incident when complete. |
