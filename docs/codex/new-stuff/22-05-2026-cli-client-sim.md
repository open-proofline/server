# Codex Prompt: Add Incident Simulator CLI

Add a small Go CLI tool that simulates the future iOS client.

Do not change backend behaviour unless required to fix bugs found by the simulator.

## Goal

Create:

```text
server/cmd/simclient/main.go
```

The simulator should create an incident, create an emergency viewer token, print the viewer URL, then upload fake encrypted chunks and checkins over time.

This is intended to test the full backend flow before building the real iOS client.

## Requirements

### CLI flags

Add these flags:

| Flag | Default | Description |
|---|---:|---|
| `--api` | `http://localhost:8080` | Private API base URL |
| `--viewer` | `http://localhost:8081` | Emergency viewer base URL |
| `--chunks` | `12` | Number of chunks to upload |
| `--interval` | `5s` | Delay between chunk uploads |
| `--media-type` | `audio` | Media type to upload |
| `--chunk-size` | `64KiB` | Size of each fake encrypted chunk |
| `--close` | `false` | Close the incident when complete |
| `--simulate-failure-every` | `0` | Every Nth chunk should intentionally fail hash verification before retrying |

## Flow

1. Create an incident using:

```http
POST /v1/incidents
```

2. Create an emergency token using:

```http
POST /v1/incidents/{incident_id}/emergency-tokens
```

3. Print:

- incident ID
- emergency viewer URL

4. For each chunk:

- generate random bytes
- compute SHA-256
- upload using:

```http
POST /v1/incidents/{incident_id}/chunks
```

- send a checkin every few chunks
- wait for `--interval`

5. If `--simulate-failure-every` is greater than `0`:

- intentionally send a bad hash for every Nth chunk
- confirm the server rejects it
- retry the same chunk with the correct hash

6. If `--close` is true:

- close the incident at the end using:

```http
POST /v1/incidents/{incident_id}/close
```

## Constraints

- Use Go standard library where practical.
- Do not add frontend frameworks.
- Do not add new auth.
- Do not add Docker Compose.
- Do not add Kubernetes.
- Do not add unrelated deployment files.
- Keep output readable.
- Do not change public JSON field names.
- Do not change existing endpoint behaviour.
- Do not add unrelated features.

## Output style

The simulator should print readable progress logs, for example:

```text
Creating incident...
Incident: inc_abc123
Emergency viewer: http://localhost:8081/e/token_here

Uploading audio chunk 1/12...
Sending checkin...
Uploading audio chunk 2/12...
Uploading audio chunk 3/12...
Done.
```

If simulated failures are enabled, output should make the failure/retry clear:

```text
Uploading audio chunk 4/12 with intentionally bad hash...
Server rejected chunk as expected.
Retrying audio chunk 4/12 with correct hash...
Retry succeeded.
```

## README update

Update `server/README.md` with a section:

~~~md
## Simulate an incident

Run the backend, then in another terminal:

```bash
go run ./cmd/simclient --chunks 12 --interval 5s
```

Open the printed emergency viewer URL to watch incident metadata update.

To test upload failure/retry behaviour:

```bash
go run ./cmd/simclient --chunks 12 --interval 2s --simulate-failure-every 4
```
~~~

## Tests

Add tests where practical for reusable simulator helpers.

Do not overbuild tests for terminal output unless it is simple.

Existing backend tests must continue to pass.

## Validation

After implementing:

```bash
go test ./...
```

Then manually verify:

```bash
go run ./cmd/api
```

In another terminal:

```bash
go run ./cmd/simclient --chunks 5 --interval 1s
```

Open the printed emergency viewer URL and confirm it shows updated chunk/checkin data.

## Summary after implementation

After making changes, summarize:

- files changed
- CLI flags added
- how to run the simulator
- whether any backend bugs were found/fixed
- whether tests pass
