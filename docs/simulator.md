# Simulator

The simulator CLI lives at `server/cmd/simclient`. It exercises the current Proofline ingest flow that a future recording client is expected to use. By default it encrypts fake chunk plaintext with the v1 client-side envelope before upload.

The simulator covers generic incidents only. It does not implement planned incident modes such as emergency incidents, interaction records, safety checks, or evidence notes.

## Basic Flow

Start the backend first:

```bash
cd server
go run ./cmd/api
```

Then run:

```bash
go run ./cmd/simclient --chunks 12 --interval 5s
```

The simulator prints an incident viewer URL. Open it to watch incident metadata update.

## Bundle Download Flow

To test encrypted bundle download through the incident viewer:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

This creates a media stream, uploads encrypted chunks with `stream_id`, completes the stream, downloads the completed encrypted ZIP bundle through the incident viewer, and verifies local decryption.

## Encryption

Encryption is enabled by default:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

The simulator prints a non-secret `key_id`, but it never prints the raw key or decrypted plaintext.

To reuse a simulator key across runs:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle --key-file /tmp/proofline-sim.key.json
```

If the key file exists, the simulator loads it. If it does not exist, the simulator creates it with restrictive permissions where practical. Do not upload or commit simulator key files.

Older examples may use `/tmp/safety-recorder-sim.key.json`; the file name is not part of the encryption protocol.

To preserve the old raw fake chunk behavior:

```bash
go run ./cmd/simclient --encrypt=false
```

This is only for development compatibility. See [encryption.md](encryption.md) for the envelope and key file format.

## Failure And Retry Flow

To test hash failure and retry behavior:

```bash
go run ./cmd/simclient --chunks 12 --interval 2s --simulate-failure-every 4
```

Every fourth chunk intentionally fails SHA-256 verification before being retried.

## Useful Flags

| Flag | Purpose |
|---|---|
| `--api` | Private API base URL. |
| `--viewer` | Incident viewer base URL. |
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
