# AGENTS.md

## Project rules

- Keep the backend small, boring, and testable.
- Do not add React, Node, npm, Docker, Kubernetes, OAuth, JWT, or user accounts unless explicitly requested.
- Prefer Go standard library where practical.
- Do not expose write/admin APIs publicly.
- Treat uploaded chunks as immutable.
- Never log raw tokens, request bodies, uploaded file bytes, or Authorization headers.

## Commands

From `server/`:

```bash
go test ./...
gofmt -w .