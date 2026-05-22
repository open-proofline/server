# Codex Prompt: Add Completed Media Streams and Downloadable Evidence Bundles

Add support for completed uploaded media streams and downloadable incident evidence bundles.

Do not add playable media merging, decryption, React, Node, npm, Docker Compose, Kubernetes, SMS, Messenger, push notifications, OAuth, JWT, or user accounts.

## Goal

The backend currently stores uploaded encrypted chunks and shows incident/chunk/checkin metadata in the emergency viewer.

Add a layer above chunks so a sequence of uploaded chunks can be marked as a completed media stream and downloaded as an evidence bundle.

The first version should produce downloadable encrypted bundles, not decrypted or playable media.

## Design concept

A chunk is an immutable uploaded blob.

A media stream groups chunks that belong to the same recording stream.

Examples:

```text
Incident
  -> audio stream #1
      -> audio chunk 1
      -> audio chunk 2
      -> audio chunk 3

  -> video stream #1
      -> video chunk 1
      -> video chunk 2

  -> location stream #1
      -> location chunk 1
```

A media stream can be:

```text
open
complete
failed
```

An incident may still be open while one stream is complete.

## Database changes

Add a `media_streams` table.

Suggested fields:

- `id`
- `incident_id`
- `media_type`: `audio`, `video`, `location`, or `metadata`
- `label`: nullable string
- `status`: `open`, `complete`, or `failed`
- `expected_chunk_count`: nullable integer
- `created_at`
- `updated_at`
- `completed_at`: nullable timestamp
- `failed_at`: nullable timestamp
- `failure_reason`: nullable string

Update chunks so each chunk may belong to a stream:

- add `stream_id` nullable or required depending on migration strategy

Prefer a migration path that does not break existing tests or existing chunks.

If existing chunks do not have `stream_id`, handle them safely and document the compatibility behaviour.

## API additions

Private API routes:

```http
POST /v1/incidents/{incident_id}/streams
GET  /v1/incidents/{incident_id}/streams
GET  /v1/incidents/{incident_id}/streams/{stream_id}
POST /v1/incidents/{incident_id}/streams/{stream_id}/complete
POST /v1/incidents/{incident_id}/streams/{stream_id}/fail
GET  /v1/incidents/{incident_id}/streams/{stream_id}/download
```

Emergency viewer routes:

```http
GET /e/{token}/streams/{stream_id}/download
GET /e/{token}/incident/download
```

The emergency viewer should show download buttons only for completed streams.

## Create stream

Route:

```http
POST /v1/incidents/{incident_id}/streams
```

Request JSON:

```json
{
  "media_type": "audio",
  "label": "main audio recording"
}
```

Response `201`:

```json
{
  "stream": {
    "id": "str_...",
    "incident_id": "inc_...",
    "media_type": "audio",
    "label": "main audio recording",
    "status": "open",
    "created_at": "..."
  }
}
```

Rules:

- reject if incident does not exist
- reject invalid media type
- do not require incident to be closed
- streams start as `open`

## Upload chunk changes

Update chunk upload so a chunk can be associated with a stream.

For multipart upload, accept an optional or required field:

```text
stream_id
```

Recommended behaviour:

- If `stream_id` is provided:
  - verify the stream exists
  - verify the stream belongs to the same incident
  - verify the stream is open
  - verify the stream media type matches the uploaded chunk media type
  - reject uploads to complete or failed streams
- If `stream_id` is not provided:
  - keep existing behaviour for backwards compatibility
  - document that new clients should provide `stream_id`

Do not change the existing hash verification, temp file, immutable storage, duplicate chunk, or upload size behaviour.

## Complete stream

Route:

```http
POST /v1/incidents/{incident_id}/streams/{stream_id}/complete
```

Request JSON:

```json
{
  "expected_chunk_count": 12
}
```

Rules:

- reject if incident does not exist
- reject if stream does not exist
- reject if stream does not belong to incident
- reject if stream is already complete
- reject if stream is failed
- verify all chunks from 1..expected_chunk_count exist for that stream
- verify chunk indexes are contiguous
- verify each chunk already has a valid stored file
- set stream status to `complete`
- set `completed_at`
- store `expected_chunk_count`
- do not modify chunk files

Response `200`:

```json
{
  "stream": {
    "id": "str_...",
    "status": "complete",
    "expected_chunk_count": 12,
    "completed_at": "..."
  }
}
```

## Fail stream

Route:

```http
POST /v1/incidents/{incident_id}/streams/{stream_id}/fail
```

Request JSON:

```json
{
  "failure_reason": "client stopped recording unexpectedly"
}
```

Rules:

- mark stream as `failed`
- set `failed_at`
- preserve uploaded chunks
- failed streams should not show normal completed download buttons
- failed streams may still be visible in metadata

## Download completed stream bundle

Route:

```http
GET /v1/incidents/{incident_id}/streams/{stream_id}/download
```

Emergency viewer route:

```http
GET /e/{token}/streams/{stream_id}/download
```

Rules:

- only allow download if stream status is `complete`
- emergency token route must verify token access to that incident
- do not allow path traversal
- do not require raw file paths from the client
- stream the bundle response; do not load all chunks into memory
- do not store generated ZIP bundles initially unless there is already a clear caching design
- set safe headers

Response headers:

```http
Content-Type: application/zip
Content-Disposition: attachment; filename="incident_<incident_id>_<media_type>_<stream_id>.zip"
X-Content-Type-Options: nosniff
Cache-Control: no-store
Referrer-Policy: no-referrer
```

ZIP contents:

```text
manifest.json
chunks/audio_000001.enc
chunks/audio_000002.enc
chunks/audio_000003.enc
```

`manifest.json` should include:

```json
{
  "incident_id": "inc_...",
  "stream_id": "str_...",
  "media_type": "audio",
  "status": "complete",
  "created_at": "...",
  "completed_at": "...",
  "chunk_count": 12,
  "total_bytes": 786432,
  "chunks": [
    {
      "chunk_index": 1,
      "media_type": "audio",
      "byte_size": 65536,
      "sha256_hex": "...",
      "started_at": "...",
      "ended_at": "...",
      "original_filename": "audio_000001.enc"
    }
  ]
}
```

The manifest should be deterministic and generated from trusted database metadata.

## Download whole incident bundle

Route:

```http
GET /e/{token}/incident/download
```

Optional private route:

```http
GET /v1/incidents/{incident_id}/download
```

This should produce a ZIP containing:

```text
manifest.json
streams/<stream_id>/manifest.json
streams/<stream_id>/chunks/audio_000001.enc
streams/<stream_id>/chunks/audio_000002.enc
```

Only include completed streams in the initial implementation unless there is a clear reason to include failed/open streams.

`manifest.json` should summarize:

- incident metadata
- latest checkin
- all completed streams
- chunk counts
- total bytes
- generated_at

## Emergency viewer updates

Update the emergency viewer page so completed streams show download buttons.

Example display:

```text
Completed recordings

Audio recording
12 chunks · 786 KB · completed 3 minutes ago
[Download encrypted bundle]
```

Do not add a frontend build system.

Use existing server-rendered HTML/CSS.

No React.

## Simulator updates

Update `cmd/simclient` so it can test streams.

Simulator flow should become:

1. create incident
2. create emergency token
3. create media stream
4. upload chunks with `stream_id`
5. send checkins
6. complete stream
7. print emergency viewer URL
8. optionally test stream download endpoint

Add flags if useful:

- `--complete-stream`, default `true`
- `--download-bundle`, default `false`

Keep backwards-compatible simulator behaviour where practical.

## Security requirements

- emergency token downloads must be read-only
- raw emergency tokens must not be logged
- bundle download must not expose server filesystem paths
- never accept a client-provided stored path for download
- generated ZIP entry names must be sanitized and controlled by the server
- set `Content-Disposition: attachment`
- set `X-Content-Type-Options: nosniff`
- set `Cache-Control: no-store`
- set `Referrer-Policy: no-referrer`
- keep private API and public emergency viewer on separate server/mux boundaries
- do not mount private write/admin routes on the public viewer server

## Tests

Add tests covering:

- create media stream
- reject invalid media type
- upload chunk with valid `stream_id`
- reject chunk upload where stream belongs to another incident
- reject chunk upload where media type does not match stream media type
- reject chunk upload to completed stream
- complete stream with contiguous chunks
- reject stream completion with missing chunk
- reject duplicate completion
- fail stream
- reject download of open stream
- reject download of failed stream
- download completed stream bundle
- ZIP contains `manifest.json`
- ZIP contains expected chunk files
- ZIP manifest hashes match chunk metadata
- emergency token can download completed stream
- invalid/expired/revoked emergency token cannot download
- public viewer shows download button only for completed streams
- private routes are not mounted on public server

Existing tests must continue to pass.

## Documentation

Update:

- `server/README.md`
- `docs/code-map.md`
- `CHANGELOG.md` if appropriate
- any API documentation if present

Document:

- media streams
- stream completion
- encrypted bundle downloads
- emergency viewer download buttons
- current limitation: bundles are encrypted chunk bundles, not playable/decrypted media
- future work: client-side decryption, playable export, key sharing

## Constraints

Do not add:

- playable media merging
- ffmpeg
- decryption
- browser decryption
- React
- Node
- npm
- OAuth
- JWT
- user accounts
- SMS
- Messenger
- push notifications
- Docker Compose
- Kubernetes
- cloud deployment integrations

Keep the implementation small, boring, local/private-first, and testable.

## Validation

After implementation:

```bash
gofmt -w .
go test ./...
```

Also manually verify:

```bash
go run ./cmd/api
```

In another terminal:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

Open the emergency viewer URL and confirm completed stream download buttons appear.

## Summary after implementation

Summarize:

- files changed
- database changes
- API routes added
- simulator changes
- emergency viewer changes
- security considerations
- tests added
- whether all tests pass
