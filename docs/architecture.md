# Architecture

Proofline is currently a single Go backend binary with separate private and public HTTP listener groups. It stores incident metadata in SQLite and encrypted uploaded chunks on local disk.

The long-term product direction is broader than emergency-only recording. Future clients may support emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. The current backend still stores generic incidents; first-class incident types, escalation policies, account access, trusted-contact accounts, notification delivery, and mobile/web clients are not implemented yet. Planned modes are documented in [incident-modes.md](incident-modes.md).

The repository does not contain an iOS app, Android app, web client, recording implementation, production client key storage, key sharing, browser/client-side decryption, server-assisted break-glass key access, notification system, user account model, or playable media export. The Go simulator can produce the documented v1 client-side encryption envelope for development and test flows. Future key custody and emergency access design is documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md).

## High-Level System

```mermaid
flowchart LR
    FutureClients["Planned clients<br/>web / iOS / Android<br/>not implemented"] -->|"future encrypted chunks"| PrivateAPI["Private /v1 API<br/>write/admin routes"]
    Simulator["Simulator CLI<br/>implemented"] --> PrivateAPI
    PrivateAPI --> Repo["Incident repository"]
    Repo --> DB[(SQLite metadata)]
    PrivateAPI --> Store["Blob storage"]
    Store --> Files[(Encrypted chunk files)]
    PrivateAPI --> Token["Viewer token creation"]
    Contact["Trusted contact"] --> Viewer["Public incident viewer<br/>/e/{token}"]
    Viewer --> Repo
    Viewer --> Store
    Viewer --> Bundle["Encrypted ZIP evidence bundles"]
```

## Planned Product Shape

The current backend is expected to become the server component in a future multi-client product. A future organisation/repository split may separate:

```text
server        Go backend, migrations, deployment docs, and admin/operational UI
web-client    account portal, authorised incident review, and eventual viewer replacement
ios-client    iOS incident capture, encrypted staging, upload, and account management
android-client Android incident capture, encrypted staging, upload, and account management
protocol      shared API specs, envelope specs, bundle manifests, and conformance tests
```

This repository has not been split yet. Documentation now uses the product name Proofline, while repository URLs, module paths, Docker image names, and GHCR package names may still use `safety-recorder` until a separate migration is performed.

## Example Network Topology

```mermaid
flowchart TB
    subgraph PrivateNetwork["Private boundary"]
        FuturePhone["Planned mobile client<br/>future"] --> WireGuard["WireGuard / LAN / firewall"]
        Simulator["Simulator CLI"] --> PrivateListener["Private API listener<br/>SAFE_PRIVATE_BIND_ADDRS"]
        WireGuard --> PrivateListener
        PrivateListener --> Storage["SQLite + local encrypted blobs"]
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
    participant Blob as Local blob storage
    participant Public as Public incident viewer
    participant Contact as Trusted contact

    Client->>Private: POST /v1/incidents
    Private->>DB: Create generic incident metadata
    Client->>Private: POST /v1/incidents/{id}/emergency-tokens
    Private->>DB: Store token hash only
    Client->>Private: POST /v1/incidents/{id}/streams
    Private->>DB: Create open stream
    Client->>Private: POST encrypted chunks + ciphertext SHA-256
    Private->>Blob: Stage, hash, commit immutable chunk
    Private->>DB: Recheck state and store chunk metadata
    Client->>Private: POST /streams/{stream_id}/complete
    Private->>DB: Verify chunks and mark complete transactionally
    Contact->>Public: GET /e/{token}
    Public->>DB: Validate token and read summary
    Public->>Blob: Stream completed encrypted bundle
```

Future clients may classify the same generic backend incident as an emergency incident, interaction record, safety check, or evidence note in client/protocol metadata after that design exists. The current API does not yet store a first-class incident type.

## Private/Public Server Boundary

```mermaid
flowchart LR
    subgraph PrivateMux["Private mux"]
        V1["/v1 routes<br/>create incidents, upload chunks,<br/>create/revoke tokens, complete streams"]
    end

    subgraph PublicMux["Public mux"]
        Viewer["/e/{token} routes<br/>read-only page, JSON,<br/>completed bundle downloads"]
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

Proofline does not currently contact emergency services. Future dead-man switch or safety-check designs should rely on trusted contacts to review the context and decide whether to call emergency services unless a future jurisdiction-specific emergency-services integration is explicitly designed, implemented, and documented.
