# Documentation

This directory contains the detailed project documentation for Safety Recorder. The top-level [README](../README.md) is a concise project overview; these docs keep operational, API, deployment, and development details in one place.

## Contents

| Document | Purpose |
|---|---|
| [Getting started](getting-started.md) | Run the backend locally and exercise the simulator flow. |
| [Architecture](architecture.md) | System diagrams, listener boundaries, and data flow. |
| [Configuration](configuration.md) | Environment variables, bind addresses, upload limits, and data layout. |
| [Encryption](encryption.md) | Client-side chunk envelope, simulator key file, and local bundle verification. |
| [Key custody and emergency access](key-custody.md) | Future production key custody, trusted-contact access, and break-glass design. |
| [Browser-side decryption](browser-decryption.md) | Future emergency viewer decryption options, risks, and phased direction. |
| [Break-glass key access](break-glass-key-access.md) | Future optional server-assisted emergency key access and dead-man-switch design. |
| [API](api.md) | Current HTTP routes, request examples, response examples, and bundle formats. |
| [Deployment](deployment.md) | Local, Docker, reverse proxy, TLS, and public exposure notes. |
| [Security model](security-model.md) | Current controls, browser headers, logging posture, and security assumptions. |
| [Threat model](threat-model.md) | Assets, trust boundaries, controls, limitations, and next security steps. |
| [Simulator](simulator.md) | Simulator commands and test flows. |
| [Development](development.md) | Repository layout, commands, AI assistance note, checks, and release checklist notes. |
| [Codex change control](codex-change-control.md) | Rollback points, scoped Codex tasks, review steps, and issue-first backlog rules. |
| [Code map](code-map.md) | Package layout and main backend request flows. |

## Current Scope

Safety Recorder currently contains the Go backend only. It receives already-encrypted chunks, stores metadata in SQLite, stores encrypted blobs on local disk, groups chunks into media streams, and exposes a token-scoped read-only emergency viewer. The Go simulator can produce the documented v1 client-side encryption envelope for development and test flows. Future production key custody is documented in [key-custody.md](key-custody.md).

Evidence bundles are encrypted chunk bundles with JSON manifests. They are not decrypted, playable, or merged media exports.

## Security Reminder

The private `/v1` API has no public user authentication. Keep it behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. Separate private/public bind addresses reduce accidental exposure, but they are not a complete security model.
