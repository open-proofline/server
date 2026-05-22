# Codex Prompt: Simulator Maintenance

Update or review the simulator CLI for the current backend.

Do not add unrelated product features.

## Current simulator purpose

The simulator acts like a fake future iOS client.

It should exercise:

- incident creation
- emergency token creation
- media stream creation
- chunk uploads with `stream_id`
- SHA-256 validated upload flow
- checkins
- stream completion
- optional failure/retry behaviour
- optional encrypted bundle download verification

## Review or update focus

Check that the simulator:

- uses the private API base URL for `/v1` routes
- uses the public viewer base URL for printed emergency links
- creates a media stream before uploading chunks
- uploads chunks with `stream_id`
- keeps chunk indexes unique per `(incident_id, media_type, chunk_index)`
- sends checkins during simulation
- completes the stream by default unless configured otherwise
- can simulate bad-hash failure and retry
- can optionally download a completed encrypted bundle
- prints readable progress output
- does not require public auth
- does not assume decrypted/playable media exists

## Constraints

Do not add:

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
- cloud integrations

## Tests

Add tests for reusable simulator helpers where practical.

Do not overbuild tests around terminal output unless simple.

Existing backend tests must continue to pass.

## Validation

Run:

```bash
gofmt -w .
go test ./...
```

Manual smoke test:

```bash
go run ./cmd/api
```

In another terminal:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

Summarize any changes and whether the manual flow was updated.
