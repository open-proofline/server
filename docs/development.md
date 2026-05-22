# Development

Safety Recorder is a Go backend only. Keep changes small, boring, and testable.

## AI Assistance

This repository uses OpenAI Codex as an AI-assisted development tool. Codex may generate or modify code and documentation, but changes are accepted only after maintainer review and testing.

The maintainer remains responsible for correctness, security, licensing, releases, deployment decisions, and real-world use. Use of Codex does not imply endorsement, audit, certification, or maintenance by OpenAI.

## Repository Layout

```text
server/
  cmd/api          API server entry point
  cmd/simclient    simulator CLI
  internal/config  environment configuration and HTTP timeout parsing
  internal/db      SQLite setup, schema_migrations, and compatibility migrations
  internal/envelope client-side chunk encryption envelope helpers
  internal/httpapi HTTP handlers, muxes, middleware, bundles, web assets
  internal/incidents incident, stream, chunk, checkin, and token repository code
  internal/storage local immutable blob storage
  migrations       embedded SQLite schema
docs/              project documentation
```

See [code-map.md](code-map.md) for a package-level walkthrough.

## Commands

From `server/`:

```bash
gofmt -w .
go test ./...
```

Use `go vet ./...` when reviewing larger changes:

```bash
go vet ./...
```

## Documentation Checks

When editing docs, keep these claims aligned:

- current version and scope in [../README.md](../README.md)
- route details in [api.md](api.md)
- security assumptions in [security-model.md](security-model.md) and [threat-model.md](threat-model.md)
- package layout in [code-map.md](code-map.md)
- release notes in [../CHANGELOG.md](../CHANGELOG.md)

Do not claim production readiness unless deployment hardening has actually been implemented.

## Release Checklist

Before tagging:

- run `gofmt -w .`
- run `go test ./...`
- run `go vet ./...` when practical
- verify README badges and links
- verify `LICENSE` and `SECURITY.md`
- verify Docker/GHCR notes
- verify private/public route separation is documented
- verify raw tokens, request bodies, uploaded bytes, and Authorization headers are not logged
- verify no local DBs, uploaded blobs, generated binaries, `.env` files, or temporary files are committed
