# Documentation

This directory contains the detailed project documentation for Safety Recorder. The top-level [README](../README.md) is a concise project overview; these docs keep operational, API, deployment, and development details in one place.

## Contents

| Document | Purpose |
|---|---|
| [Getting started](getting-started.md) | Run the backend locally and exercise the simulator flow. |
| [Architecture](architecture.md) | System diagrams, listener boundaries, and data flow. |
| [Configuration](configuration.md) | Environment variables, bind addresses, upload limits, and data layout. |
| [API](api.md) | Current HTTP routes, request examples, response examples, and bundle formats. |
| [Deployment](deployment.md) | Local, Docker, reverse proxy, TLS, and public exposure notes. |
| [Security model](security-model.md) | Current controls, browser headers, logging posture, and security assumptions. |
| [Threat model](threat-model.md) | Assets, trust boundaries, controls, limitations, and next security steps. |
| [Simulator](simulator.md) | Simulator commands and test flows. |
| [Development](development.md) | Repository layout, commands, checks, and release checklist notes. |
| [Code map](code-map.md) | Package layout and main backend request flows. |

## Current Scope

Safety Recorder currently contains the Go backend only. It receives already-encrypted chunks, stores metadata in SQLite, stores encrypted blobs on local disk, groups chunks into media streams, and exposes a token-scoped read-only emergency viewer.

Evidence bundles are encrypted chunk bundles with JSON manifests. They are not decrypted, playable, or merged media exports.

## Security Reminder

The private `/v1` API has no public user authentication. Keep it behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. Separate private/public bind addresses reduce accidental exposure, but they are not a complete security model.
