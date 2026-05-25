# Safety Recorder

[![CI](https://github.com/TheSilkky/safety-recorder/actions/workflows/ci.yml/badge.svg)](https://github.com/TheSilkky/safety-recorder/actions/workflows/ci.yml)
[![Latest Tag](https://img.shields.io/github/v/tag/TheSilkky/safety-recorder?sort=semver)](https://github.com/TheSilkky/safety-recorder/tags)
[![License: AGPL-3.0-only](https://img.shields.io/badge/license-AGPL--3.0--only-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/TheSilkky/safety-recorder?filename=server%2Fgo.mod)](server/go.mod)
[![Status: Experimental](https://img.shields.io/badge/status-experimental-orange.svg)](#security-warning)
[![Security Policy](https://img.shields.io/badge/security-policy-blue.svg)](SECURITY.md)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fthesilkky%2Fsafety--recorder-blue?logo=github)](https://github.com/TheSilkky/safety-recorder/pkgs/container/safety-recorder)

Safety Recorder is an experimental Go backend for a private personal-safety recording system. It receives already-encrypted recording chunks, stores metadata in SQLite, keeps encrypted blobs on local disk, and exposes a token-scoped emergency viewer for read-only incident review.

## Security Warning

> This project is not production-ready public infrastructure. The private `/v1` API has no public user authentication and must stay behind localhost, LAN, WireGuard, a firewall, or a strict reverse proxy. Separate bind addresses are a deployment boundary, not a complete security model.

## What It Is

This repository currently contains the backend only. The intended future client is an iOS app that records audio/video in short chunks, encrypts them locally, and uploads them continuously so already-uploaded evidence is retained if a phone is lost, damaged, powered off, or taken.

Evidence bundles are ZIP files containing encrypted chunks and JSON manifests. They are not decrypted, playable, or merged media exports.

The simulator encrypts fake chunks by default with the documented v1 AES-256-GCM envelope and verifies downloaded bundles locally. Keys remain client-side and are not uploaded to the backend. Future production key custody is expected to use a hybrid trusted-contact model; see [docs/key-custody.md](docs/key-custody.md).

## What Works Today

- Private `/v1` write/admin API listener group
- Public read-only emergency viewer listener group
- SQLite metadata and local disk encrypted blob storage
- Immutable chunk uploads with SHA-256 verification
- Documented client-side chunk encryption envelope
- Media streams with `open`, `complete`, and `failed` states
- Completed encrypted stream and incident ZIP evidence bundle downloads
- Scoped emergency viewer tokens with a default 24-hour expiry
- Simulator CLI for encrypted upload, check-in, stream completion, and bundle download/decrypt-verification flows
- Docker image build and GitHub Actions / GHCR publishing

## What It Is Not Yet

- No iOS app
- No recording implementation
- No production client-side encryption implementation
- No backend/browser decryption, key sharing, server escrow, break-glass key access, or playable media export
- No push notifications, SMS, or Messenger integration
- No user accounts, OAuth, JWT, or public admin dashboard
- No built-in TLS, app-level rate limiting, automated retention/deletion enforcement, or production deployment hardening

## Architecture

Safety Recorder runs separate private and public HTTP listener groups from the same Go binary. Private `/v1` routes handle writes and admin-style operations. Public emergency viewer routes are token-gated and read-only.

```mermaid
flowchart LR
    FutureClient["Planned iOS client<br/>not in this repo"] --> Private["Private /v1 API<br/>localhost/LAN/WireGuard"]
    Private --> DB[(SQLite metadata)]
    Private --> Blobs[(Local encrypted blobs)]
    Private --> Tokens["Emergency token creation"]
    Contact["Trusted contact"] --> Public["Public emergency viewer<br/>/e/{token}"]
    Public --> DB
    Public --> Blobs
    Public --> Bundles["Encrypted ZIP bundles<br/>completed streams only"]
```

For more diagrams and package-level details, see [docs/architecture.md](docs/architecture.md) and [docs/code-map.md](docs/code-map.md).

## Quick Start

Requirements:

- Go 1.26.3
- SQLite via the bundled Go SQLite driver dependency
- Local disk storage for encrypted uploaded blobs

Run the backend:

```bash
cd server
go run ./cmd/api
```

By default this starts:

| Listener | Address |
|---|---|
| Private API | `127.0.0.1:8080` |
| Public emergency viewer | `127.0.0.1:8081` |

In another terminal, run the simulator:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

The simulator creates an incident, creates an emergency token, encrypts and uploads test chunks into a media stream, sends checkins, completes the stream, downloads the encrypted bundle, and verifies local decryption.

## Docker

Build from the repository root:

```bash
docker build -t safety-recorder-backend ./server
```

Run with local-only port publishing and a named data volume:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  -v safety-recorder-data:/data \
  safety-recorder-backend
```

Container defaults bind to `0.0.0.0` inside the container. Restrict host exposure with port publishing, firewall rules, WireGuard, or a reverse proxy. See [docs/deployment.md](docs/deployment.md).

## Documentation

- [Docs index](docs/README.md)
- [Getting started](docs/getting-started.md)
- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [Encryption](docs/encryption.md)
- [Key custody and emergency access](docs/key-custody.md)
- [Browser-side decryption design](docs/browser-decryption.md)
- [Break-glass key access design](docs/break-glass-key-access.md)
- [API reference](docs/api.md)
- [Deployment notes](docs/deployment.md)
- [Retention, backup, and deletion](docs/retention-backup-deletion.md)
- [Security model](docs/security-model.md)
- [Threat model](docs/threat-model.md)
- [Simulator](docs/simulator.md)
- [Development](docs/development.md)
- [Code map](docs/code-map.md)
- [Technical review reports](docs/reports/README.md)

## AI-Assisted Development

This project has been developed with substantial assistance from OpenAI Codex.

Codex has been used to draft, refactor, test, document, and review parts of the Go backend and Markdown documentation. All accepted changes are reviewed, tested, and committed by the maintainer.

AI assistance does not replace human responsibility. The maintainer remains responsible for:

- code correctness
- security review
- licensing decisions
- release decisions
- deployment choices
- any real-world use of the software

Use of Codex does not imply endorsement, audit, certification, or maintenance by OpenAI.

## Backlog workflow

Use `80-backlog-scan-issue-drafts.md` to generate reviewed issue drafts under `.backlog-drafts/`.

Review those drafts manually before creating GitHub issues.

Only after review, use `85-create-github-issues-from-drafts.md` to generate a script that creates GitHub issues with `gh issue create`.

Do not let Codex create GitHub issues directly during the initial scan.

## Security

Emergency viewer links are bearer-token URLs and should be treated as secrets. Public deployment still needs TLS, rate limiting, log review, proxy hardening, operational testing, and deployment-specific retention, backup, and deletion enforcement. Do not expose `/v1` publicly as-is.

Please see [SECURITY.md](SECURITY.md) for supported versions and vulnerability reporting guidance. Do not report security vulnerabilities through public GitHub issues.

## Roadmap

- WireGuard-only bind/firewall deployment guidance
- iOS client
- Client-side recording and encryption
- Production key custody, trusted-contact access, and browser/client-side decryption
- Optional break-glass/dead-man-switch key access
- Playable media export
- Reverse-proxy/TLS hardening for emergency viewer exposure
- Explicit `/v1` access-control story before any public control-plane deployment

## License

Safety Recorder is licensed under the GNU Affero General Public License v3.0 only (`AGPL-3.0-only`). See [LICENSE](LICENSE).
