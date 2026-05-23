# Encryption

Safety Recorder currently stores opaque encrypted chunk bytes. This document describes the first client-side chunk encryption envelope used by the Go simulator and tests.

This milestone does not add backend decryption. The server still validates SHA-256 over uploaded ciphertext bytes, stores those bytes on local disk, and emits encrypted ZIP evidence bundles.

## Threat Model

The v1 envelope protects chunk plaintext from the backend, SQLite, local blob storage, and evidence bundle readers who do not have the client-held key. It does not protect metadata that is already sent to the backend, such as incident ID, stream ID, media type, chunk index, timestamps, byte size, and ciphertext hashes.

The simulator key handling in this repository is for development and test use only. Future production client key storage, sharing, recovery, and emergency-contact access are out of scope for the current implementation and are designed separately in [key-custody.md](key-custody.md).

## Scheme v1

| Field | Value |
|---|---|
| Scheme | `safety-recorder-chunk-encryption-v1` |
| Algorithm | `AES-256-GCM` |
| Key size | 32 bytes |
| Nonce size | 12 bytes |
| Tag | Included in the AES-GCM ciphertext |
| Associated data | Deterministic UTF-8 metadata string |

The Go implementation uses the standard library packages `crypto/aes`, `crypto/cipher`, `crypto/rand`, `encoding/base64`, `encoding/binary`, and `encoding/json`.

Generate a fresh random nonce for every encrypted chunk. Never reuse a nonce with the same key. A single key must not be used for more than `2^32` chunks/messages. By default, the simulator generates a fresh ephemeral key per run unless `--key-file` is supplied.

## Associated Data

The AEAD associated data is an exact UTF-8 string:

```text
SafetyRecorderChunk:v1
incident_id=<incident_id>
stream_id=<stream_id>
media_type=<media_type>
chunk_index=<chunk_index>
```

There is a trailing newline after the `chunk_index` line. Example:

```text
SafetyRecorderChunk:v1
incident_id=inc_abc
stream_id=str_def
media_type=audio
chunk_index=1
```

Encryption and decryption must use identical associated data. IDs and media type must not contain newlines, and `chunk_index` must be positive. This matches streamed upload semantics; legacy unstreamed `chunk_index = 0` chunks cannot use this v1 associated data. Decryption fails when incident ID, stream ID, media type, or chunk index differs from the original metadata.

## Chunk Envelope

Each uploaded `.enc` file contains:

```text
magic bytes
uint32 big-endian header length
UTF-8 JSON header
AES-GCM ciphertext including authentication tag
```

Magic bytes are exactly:

```text
SRCENC1\n
```

The JSON header is non-secret:

```json
{
  "version": 1,
  "scheme": "safety-recorder-chunk-encryption-v1",
  "algorithm": "AES-256-GCM",
  "key_id": "kid_...",
  "nonce_b64": "base64url-no-padding-12-byte-nonce",
  "aad": "SafetyRecorderChunk:v1\nincident_id=inc_...\nstream_id=str_...\nmedia_type=audio\nchunk_index=1\n"
}
```

The implementation rejects malformed magic, truncated envelopes, oversized headers, unknown versions, unknown algorithms, missing fields, wrong nonce lengths, and associated-data mismatches. Header length is capped at 16 KiB. The header must not contain plaintext, secret keys, or server filesystem paths.

Nonce and key values use URL-safe base64 without padding.

## Simulator Key File

The simulator can load or create a local development key file:

```json
{
  "version": 1,
  "scheme": "safety-recorder-chunk-encryption-v1",
  "algorithm": "AES-256-GCM",
  "key_id": "kid_...",
  "key_b64": "base64url-no-padding-32-byte-key"
}
```

`key_id` is non-secret. `key_b64` is secret. Do not upload this file, add it to evidence bundles, commit it to git, or paste it into logs. The simulator writes key files with `0600` permissions where practical.

## Simulator Usage

Encryption is enabled by default:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

Expected output includes the non-secret key ID, encrypted chunk uploads, bundle download, and local decrypt verification. The simulator does not print raw keys or plaintext.

To persist a simulator key locally:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle --key-file /tmp/safety-recorder-sim.key.json
```

Run it again with the same path to load the existing key:

```bash
go run ./cmd/simclient --chunks 2 --interval 1s --download-bundle --key-file /tmp/safety-recorder-sim.key.json
```

To preserve the old raw fake chunk behavior for development compatibility:

```bash
go run ./cmd/simclient --encrypt=false
```

Bundle decrypt verification defaults on when `--download-bundle` and `--encrypt` are both enabled. It can be disabled with:

```bash
go run ./cmd/simclient --download-bundle --verify-bundle-decryption=false
```

## What The Backend Sees

The backend sees only opaque uploaded bytes and client-provided metadata. It stores ciphertext and validates SHA-256 over the ciphertext envelope. It does not parse keys, store keys in SQLite, upload keys, decrypt chunks, or expose public decryption endpoints.

Evidence bundles remain ZIP files containing encrypted `.enc` chunk files and JSON manifests. Bundle manifests include a non-secret hint that client-side encryption is expected and that the server does not decrypt.

## Future Work

The intended Apple-side equivalent is CryptoKit or Swift Crypto AES-GCM. This repository does not include iOS or Swift code yet.

Future work includes production client key storage, Keychain integration, emergency-contact key access, key sharing, browser/client-side decryption, and playable export. The intended production key custody direction is a hybrid trusted-contact model documented in [key-custody.md](key-custody.md), with browser decryption constraints documented in [browser-decryption.md](browser-decryption.md). Password-derived keys, passphrases, public-key wrapping, key escrow, backend decryption, and browser decryption are not implemented in this milestone.
