# Getting Started

This guide starts the backend locally and runs the simulator against it.

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
| Public emergency viewer | `127.0.0.1:8081` |

The backend writes local data under `./data` by default:

```text
data/
  safety.db
  tmp/
  incidents/{incident_id}/{media_type}_{zero_padded_chunk_index}.enc
```

Uploads are staged in `tmp/`, hashed while streaming, and then hard-linked into the final incident path without overwriting existing chunk files.

## Run The Simulator

In another terminal from `server`:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

The simulator:

- creates an incident
- creates an emergency viewer token
- creates an audio media stream
- encrypts fake chunk plaintext and uploads the ciphertext envelope with `stream_id`
- sends periodic checkins
- marks the stream complete
- downloads the completed encrypted stream bundle through the emergency viewer when requested
- verifies local decryption of the downloaded bundle when encryption is enabled

## Useful Next Reads

- [Configuration](configuration.md)
- [Encryption](encryption.md)
- [Simulator](simulator.md)
- [API reference](api.md)
- [Deployment](deployment.md)
