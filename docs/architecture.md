# Architecture

Safety Recorder is a single Go backend binary with separate private and public HTTP listener groups. It stores incident metadata in SQLite and encrypted uploaded chunks on local disk.

The repository does not contain an iOS app, recording implementation, production client key storage, key sharing, browser/client-side decryption, server-assisted break-glass key access, or playable media export. The Go simulator can produce the documented v1 client-side encryption envelope for development and test flows. Future key custody and emergency access design is documented in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md).

## High-Level System

```mermaid
flowchart LR
    FutureClient["Planned iOS app<br/>not implemented"] -->|"future encrypted chunks"| PrivateAPI["Private /v1 API<br/>write/admin routes"]
    Simulator["Simulator CLI<br/>implemented"] --> PrivateAPI
    PrivateAPI --> Repo["Incident repository"]
    Repo --> DB[(SQLite metadata)]
    PrivateAPI --> Store["Blob storage"]
    Store --> Files[(Encrypted chunk files)]
    PrivateAPI --> Token["Emergency token creation"]
    Contact["Trusted contact"] --> Viewer["Public emergency viewer<br/>/e/{token}"]
    Viewer --> Repo
    Viewer --> Store
    Viewer --> Bundle["Encrypted ZIP evidence bundles"]
```

## Example Network Topology

```mermaid
flowchart TB
    subgraph PrivateNetwork["Private boundary"]
        FuturePhone["Planned iOS app<br/>future"] --> WireGuard["WireGuard / LAN / firewall"]
        Simulator["Simulator CLI"] --> PrivateListener["Private API listener<br/>SAFE_PRIVATE_BIND_ADDRS"]
        WireGuard --> PrivateListener
        PrivateListener --> Storage["SQLite + local encrypted blobs"]
    end

    subgraph PublicEdge["Public emergency viewer exposure"]
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
    participant Public as Public emergency viewer
    participant Contact as Trusted contact

    Client->>Private: POST /v1/incidents
    Private->>DB: Create incident metadata
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

## Private/Public Server Boundary

```mermaid
flowchart LR
    subgraph PrivateMux["Private mux"]
        V1["/v1 routes<br/>create incidents, upload chunks,<br/>create/revoke tokens, complete streams"]
    end

    subgraph PublicMux["Public mux"]
        Emergency["/e/{token} routes<br/>read-only page, JSON,<br/>completed bundle downloads"]
        Static["/static assets<br/>token-neutral"]
    end

    PrivateMux --> PrivateBind["SAFE_PRIVATE_BIND_ADDRS"]
    PublicMux --> PublicBind["SAFE_PUBLIC_BIND_ADDRS"]

    Warning["Do not mount /v1 on public viewer listeners"]
```

## Evidence Bundles

Completed stream and incident downloads are ZIP files generated on demand. ZIP entry names are controlled by the server and manifests are generated from trusted database metadata. Bundles contain encrypted chunks and JSON manifests only.

They are not decrypted, playable, or merged media exports.
