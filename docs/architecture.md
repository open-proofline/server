# Architecture

Proofline Server is currently a single Go backend binary with separate private
and public HTTP listener groups. It stores incident metadata in SQLite by
default or optional PostgreSQL, encrypted uploaded chunks on local disk by
default with optional S3-compatible object storage for committed encrypted
chunks, and optional Valkey/Redis-compatible short-lived coordination when
explicitly configured.

This repository is the server/backend component only. In the planned multi-repo layout it corresponds to `open-proofline/server`. Web, iOS, Android, and shared protocol work are expected to live in separate future repositories.

The long-term product direction is broader than emergency-only recording. Future clients may support emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. The current backend still stores generic incidents and now has local username/password accounts with opaque server-side sessions for the private `/v1` API. First-class incident modes, capture profiles, escalation policies, sharing state, trusted-contact accounts, notification delivery, and mobile/web clients are not implemented yet. Planned mode, escalation, migration, and viewer-wording boundaries are documented in [incident-modes.md](incident-modes.md), and current local session behavior plus future public product API, separately bound private admin API, role, and grant boundaries are documented in [v1-access-control.md](v1-access-control.md).

The repository does not contain an iOS app, Android app, web client, protocol package, recording implementation, production client key storage, key sharing, browser/client-side decryption, server-assisted break-glass key access, notification system, trusted-contact account model, future public product API, future separately bound private admin API, OAuth/JWT identity integration, or playable media export. The Go simulator can produce the documented v1 client-side encryption envelope for development and test flows. Future key custody and emergency access design is documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md).

## High-Level System

```mermaid
flowchart LR
    FutureClients["Future clients<br/>separate repos"] -->|"future encrypted chunks"| PrivateAPI["Private /v1 API<br/>local session auth"]
    Simulator["Simulator CLI<br/>implemented here"] -->|"login + upload"| PrivateAPI
    PrivateAPI --> Repo["Incident repository"]
    Repo --> DB[(SQLite or PostgreSQL metadata)]
    PrivateAPI --> Store["Blob storage"]
    Store --> Files[(Encrypted chunk files)]
    PrivateAPI --> Coord["Optional coordination<br/>startup-checked Valkey/Redis"]
    PrivateAPI --> Token["Viewer token creation"]
    Contact["Trusted contact"] --> Viewer["Public incident viewer<br/>/i/{token}"]
    Viewer --> Repo
    Viewer --> Store
    Viewer --> Bundle["Encrypted ZIP evidence bundles"]
```

## Planned Open Proofline Repository Layout

The intended organisation is `open-proofline`.

Planned repositories:

```text
open-proofline/server
open-proofline/web-client
open-proofline/ios-client
open-proofline/android-client
open-proofline/protocol
```

Responsibilities:

| Repository | Responsibility |
|---|---|
| `server` | Go backend, private API, public incident viewer, SQLite migrations, encrypted blob storage, deployment docs, and server release workflow. |
| `web-client` | Account portal, authorised incident review, trusted-contact access, and eventual replacement for the current token-only viewer. |
| `ios-client` | iOS incident capture, encrypted staging, upload, local account flows, and platform-specific recording behavior. |
| `android-client` | Android incident capture, encrypted staging, upload, local account flows, and platform-specific recording behavior. |
| `protocol` | Shared API specs, encryption envelope specs, bundle manifests, compatibility matrix, and conformance tests. |

The Go module path is `github.com/open-proofline/server`, release binaries use `proofline-server-*` names, and the published GHCR image is `ghcr.io/open-proofline/server`. Compatibility identifiers such as the v1 simulator encryption envelope and default SQLite filename may still use earlier `safety-recorder` names until separate protocol or data-layout migrations are explicitly performed.

## Server Boundary

This repository should remain scoped to backend server responsibilities:

- HTTP API implementation
- SQLite migrations and metadata repository code
- encrypted blob storage
- current token-scoped incident viewer
- deployment and operations docs
- simulator/reference backend flow
- backend security, retention, and threat-model docs

Do not add future web-client, iOS-client, Android-client, or protocol implementation here unless the maintainer explicitly changes the repository strategy.

## Example Network Topology

```mermaid
flowchart TB
    subgraph PrivateNetwork["Private boundary"]
        FuturePhone["Future mobile client<br/>separate repo"] --> WireGuard["WireGuard / LAN / firewall"]
        Simulator["Simulator CLI"] --> PrivateListener["Private API listener<br/>SAFE_PRIVATE_BIND_ADDRS"]
        WireGuard --> PrivateListener
        PrivateListener --> Auth["Local account sessions"]
        Auth --> Storage["SQLite or PostgreSQL + local or S3 encrypted blobs"]
        PrivateListener --> Coordination["Optional Valkey/Redis coordination"]
    end

    subgraph PublicEdge["Public incident viewer exposure"]
        TrustedContact["Trusted contact"] --> HTTPS["HTTPS reverse proxy<br/>future deployment edge"]
        HTTPS --> PublicListener["Public viewer listener<br/>SAFE_PUBLIC_BIND_ADDRS"]
        PublicListener --> ReadOnly["Token-scoped read-only access"]
        ReadOnly --> Storage
    end
```

## Incident Data Flow

```mermaid
sequenceDiagram
    participant Client as Simulator or future client
    participant Private as Private /v1 API
    participant DB as SQLite
    participant Blob as Blob storage
    participant Public as Public incident viewer
    participant Contact as Trusted contact

    Client->>Private: POST /v1/auth/login
    Private->>DB: Validate account and create hashed session record
    Client->>Private: POST /v1/incidents
    Private->>DB: Create generic incident metadata for account
    Client->>Private: POST /v1/incidents/{id}/incident-tokens
    Private->>DB: Store token hash only
    Client->>Private: POST /v1/incidents/{id}/streams
    Private->>DB: Create open stream
    Client->>Private: POST encrypted chunks + ciphertext SHA-256
    Private->>Blob: Stage, hash, commit immutable chunk
    Private->>DB: Recheck state and store chunk metadata
    Client->>Private: POST /streams/{stream_id}/complete
    Private->>DB: Verify chunks and mark complete transactionally
    Contact->>Public: GET /i/{token}
    Public->>DB: Validate token and read summary
    Public->>Blob: Stream completed encrypted bundle
```

Future clients may classify the same generic backend incident as an emergency incident, interaction record, safety check, or evidence note in client/protocol metadata after that design exists. The current API does not yet store a first-class incident mode, capture profile, escalation policy, or sharing state.

## Private/Public Server Boundary

```mermaid
flowchart LR
    subgraph PrivateMux["Private mux"]
        V1["/v1 routes<br/>create incidents, upload chunks,<br/>create/revoke tokens, complete streams"]
    end

    subgraph PublicMux["Public mux"]
        Viewer["/i/{token} routes<br/>read-only page, JSON,<br/>completed bundle downloads"]
        LegacyViewer["/e/{token} aliases<br/>pre-rename compatibility"]
        Static["/static assets<br/>token-neutral"]
    end

    PrivateMux --> PrivateBind["SAFE_PRIVATE_BIND_ADDRS"]
    PublicMux --> PublicBind["SAFE_PUBLIC_BIND_ADDRS"]

    Warning["Do not mount /v1 on public viewer listeners"]
```

## Evidence Bundles

Completed stream and incident downloads are ZIP files generated on demand. ZIP entry names are controlled by the server and manifests are generated from trusted database metadata. Bundles contain encrypted chunks and JSON manifests only.

They are not decrypted, playable, or merged media exports.

## Emergency Services Boundary

Proofline Server does not currently contact emergency services. Future dead-man switch or safety-check designs should rely on trusted contacts to review the context and decide whether to call emergency services unless a future jurisdiction-specific emergency-services integration is explicitly designed, implemented, and documented.
