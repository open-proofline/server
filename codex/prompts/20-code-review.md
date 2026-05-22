# Codex Prompt: Code Review

Review the current changes for correctness, maintainability, security, and scope control.

Do not add features unless needed to fix a bug.

## Project context

This is a Go backend for a private personal-safety recording system.

The current system includes:

- private `/v1` write/admin API routes
- public read-only emergency viewer routes
- separate private/public listener groups
- SQLite metadata
- local disk encrypted chunk storage
- immutable chunk uploads
- media streams that can be completed or failed
- encrypted ZIP evidence bundle downloads
- simulator CLI
- Docker/GHCR/CI support

## Review focus

Check for:

- upload overwrite risks
- temp file cleanup
- SHA-256 verification correctness
- duplicate chunk rejection
- upload size limits
- SQLite constraints
- stream completion logic
- stream failure logic
- legacy chunks without `stream_id`
- evidence bundle ZIP path traversal
- ZIP manifest correctness
- server-controlled ZIP entry names
- download routes exposing filesystem paths
- private `/v1` routes mounted on public emergency server
- public emergency routes mutating data
- raw token logging
- request body logging
- uploaded file byte logging
- Authorization header logging
- template escaping
- security headers on emergency viewer and downloads
- plural bind address parsing
- tests that do not assert important behaviour

## Output format

Return:

1. Critical issues
2. Important issues
3. Minor issues
4. Suggested minimal fixes

If you make changes:

- keep them small
- do not add unrelated features
- do not change public JSON field names unless required for a bug
- run:

```bash
gofmt -w .
go test ./...
```

Summarize what changed.
