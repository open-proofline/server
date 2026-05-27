# Codex Prompt: Simulator Maintenance

Update or review the simulator CLI for the current backend.

Do **not** add unrelated product features.
Do **not** implement iOS code.
Do **not** add backend decryption.
Do **not** change key custody model unless explicitly requested.

## Source of truth

Before making changes, read current source-of-truth files as relevant:

- `README.md`
- `AGENTS.md`
- `CHANGELOG.md`
- `SECURITY.md`
- `docs/README.md`
- relevant files in `docs/`
- relevant source files
- relevant tests
- relevant issue or PR, if this is issue/PR work

Do not rely on stale assumptions from this prompt if the repository has changed.
## Global constraints

- Keep changes scoped to the task.
- Do not add unrelated features.
- Do not weaken security warnings.
- Do not claim production readiness.
- Do not expose `/v1` publicly.
- Do not log raw tokens, request bodies, uploaded bytes, Authorization headers, plaintext, raw keys, or future token-like values.
- Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, Docker Compose, Kubernetes, cloud integrations, or public admin dashboard features unless explicitly requested.
- Prefer Go standard library where practical.
- Preserve private/public listener separation.
- Preserve the current backend ciphertext-only implementation unless the task explicitly concerns key custody, emergency access, or decryption design.
- Do not introduce backend decryption, raw server-held decryption keys, key escrow, browser decryption, or key-sharing behaviour as an incidental implementation detail.
- Any key custody/decryption change must be an explicit security-sensitive task that updates the threat model, security model, encryption docs, tests, and operational guidance before or alongside implementation.
- Future production key custody should assume the iPhone may be unavailable; keys must not exist solely on the client device.
- Server storage of wrapped/encrypted keys may be acceptable if explicitly designed.
- Raw server-side key access or server-side decryption may be acceptable only as a deliberate break-glass/dead-man-switch/emergency-access mode with clear access controls, audit expectations, and deployment warnings.

## Current simulator purpose

The simulator acts like a fake future iOS client.

It should exercise:

- incident creation
- incident token creation
- media stream creation
- chunk uploads with `stream_id`
- stream chunk indexes starting at `1`
- SHA-256 validated upload flow
- client-side encryption envelope for simulated chunks
- local simulator key-file loading/saving, when configured
- checkins
- stream completion
- optional failure/retry behaviour
- optional encrypted bundle download verification
- local decrypt verification of downloaded bundles when encryption is enabled

## Review or update focus

Check that the simulator:

- uses the private API base URL for `/v1` routes
- uses the public viewer base URL for printed incident viewer links
- creates a media stream before uploading chunks
- uploads chunks with `stream_id`
- keeps chunk indexes unique per `(incident_id, media_type, chunk_index)`
- starts stream chunk indexes at `1`
- encrypts fake chunks by default
- supports `--encrypt=false` only for development compatibility
- supports `--key-file` without printing the key
- does not print raw keys or plaintext
- sends checkins during simulation
- completes the stream by default unless configured otherwise
- can simulate bad-hash failure and retry
- can optionally download a completed encrypted bundle
- can decrypt-verify downloaded encrypted chunks locally
- prints readable progress output
- does not require public auth
- does not assume playable media exists

## Tests

Add tests for reusable simulator helpers where practical.

Do not overbuild tests around terminal output unless simple.

Existing backend tests must continue to pass.

## Validation

Run:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

Manual smoke test:

```bash
cd server
go run ./cmd/api
```

In another terminal:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

## Output

Summarize:

1. files changed
2. simulator behaviour updated
3. encryption/key-file impact
4. validation commands run
5. any follow-up work
