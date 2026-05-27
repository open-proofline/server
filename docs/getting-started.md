# Getting Started

This guide starts the Proofline backend locally and runs the simulator against it.

## Requirements

- Go 1.26.3
- SQLite through the bundled Go SQLite driver dependency
- Local disk storage for encrypted uploaded blobs

## Run The Backend

From the `server` directory:

```bash
go run ./cmd/api
```

Default listeners:

| Listener | Default address |
|---|---|
| Private API | `127.0.0.1:8080` |
| Public incident viewer | `127.0.0.1:8081` |

The backend writes local data under `./data` by default:

```text
data/
  safety.db
  tmp/
  incidents/{incident_id}/streams/{stream_id}/{media_type}_{zero_padded_chunk_index}.enc
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

The database file name still uses `safety.db` until a separate artifact/data-layout migration is performed.

Uploads are staged in `tmp/`, hashed while streaming, and then hard-linked into the final incident path without overwriting existing chunk files. Streamed uploads use the stream-scoped path; the incident-level path remains for legacy unstreamed chunks.

## Run The Simulator

In another terminal from `server`:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

The simulator:

- creates an incident
- creates an incident viewer token using the current emergency-token route
- creates an audio media stream
- encrypts fake chunk plaintext and uploads the ciphertext envelope with `stream_id`
- sends periodic checkins
- marks the stream complete
- downloads the completed encrypted stream bundle through the incident viewer when requested
- verifies local decryption of the downloaded bundle when encryption is enabled

The simulator exercises the current generic incident API. It does not implement planned incident modes such as emergency incidents, interaction records, safety checks, or evidence notes.

## Useful Next Reads

- [Configuration](configuration.md)
- [Incident capture modes](incident-modes.md)
- [Encryption](encryption.md)
- [Simulator](simulator.md)
- [API reference](api.md)
- [Deployment](deployment.md)
