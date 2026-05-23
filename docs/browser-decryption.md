# Browser-Side Emergency Viewer Decryption

This document is a design spike for future browser/client-side emergency viewer
decryption. It does not implement browser decryption, JavaScript cryptography,
new routes, database schema changes, or backend behavior changes.

## Summary

Browser-side decryption could let a trusted contact open an emergency viewer,
download encrypted evidence, and decrypt it locally without exposing raw media
keys to the backend during normal operation. This fits the broader hybrid key
custody direction:

- recording clients encrypt media before upload
- uploaded chunks remain ciphertext on the backend
- keys are not stored solely on the iPhone
- trusted contacts can access emergency evidence when the phone is unavailable
- browser decryption is one possible trusted-contact access path
- server escrow and break-glass access remain separate explicit design paths

The browser path is attractive for emergency access because it can be available
from a normal web link. It is not a complete end-to-end security answer when the
same backend can serve or alter the JavaScript that performs decryption.

## Current Viewer Behaviour

The current emergency viewer is token-scoped and read-only. Public routes are
mounted only on the public emergency viewer listener, not on the private `/v1`
listener.

Current behavior:

- `GET /e/{token}` renders a read-only HTML summary after token validation.
- `GET /e/{token}/data` returns the same summary as JSON for polling.
- Static CSS and JavaScript are served from embedded files under `/static/`.
- The page shows incident status, latest checkin, chunk counts, and completed
  recording download links.
- Completed stream and incident downloads are encrypted ZIP evidence bundles.
- Bundle ZIP entry names are controlled by the server.
- Bundle manifests do not include stored filesystem paths or decryption keys.
- There is no browser decryption today.

The current JavaScript only builds download links and polls the JSON endpoint.
It does not parse bundles, handle keys, decrypt chunks, or display plaintext.

## Candidate Approaches

### 1. URL Fragment Decryption Capability

How it works:

The emergency URL carries a decryption capability in the fragment, for example
`/e/{token}#key=...` or a future structured fragment. The HTTP request sends
only `/e/{token}` to the server. JavaScript running in the browser reads the
fragment and uses it to unwrap or import the media key.

What the backend sees:

- emergency token in the request path
- normal viewer and bundle requests
- incident metadata and encrypted bundle bytes
- no fragment value in normal HTTP requests

What the browser sees:

- emergency token path
- fragment-carried secret or wrapped secret
- encrypted bundles or chunks
- decrypted plaintext after local decryption

UX strengths:

- one link can carry both read access and decryption capability
- no separate file or contact-key setup is required for the simplest version
- useful for early prototypes and manual recovery workflows

Security weaknesses:

- fragment secrets can leak through screenshots, copy/paste, browser history
  behavior, extensions, local compromise, or user error
- JavaScript can read the fragment, including malicious JavaScript served by a
  compromised backend
- if the fragment carries the raw media key, anyone with the full URL can
  decrypt
- fragment delivery does not solve trusted contact revocation or key rotation

Implementation complexity:

Low for a prototype, moderate for a robust production UX with key import,
clearing fragments, error handling, and safe local downloads.

Fit for emergency use:

Potentially useful as a transitional or recovery mechanism, but too weak as the
only long-term trusted-contact model.

### 2. Contact Private Key Stored Or Imported In Browser

How it works:

Each trusted contact has a private key. The emergency viewer downloads
contact-wrapped media keys and asks the browser to import or use the contact
private key to unwrap them. The key may be stored locally by the browser,
imported from a file, or provided through a platform credential mechanism.

What the backend sees:

- emergency token and read requests
- contact-wrapped media keys
- encrypted chunks or bundles
- no raw contact private key if the browser flow is implemented correctly

What the browser sees:

- contact private key or a handle to it
- wrapped media keys
- raw media keys after unwrap
- decrypted media plaintext

UX strengths:

- aligns with the hybrid trusted-contact model
- avoids making the emergency link alone sufficient to decrypt
- can support contact revocation for future incidents by stopping new wrapping

Security weaknesses:

- a compromised browser or device can steal the contact private key
- malicious viewer JavaScript can exfiltrate imported keys or plaintext
- browser private-key storage and backup behavior may be confusing
- contact key loss can make evidence unavailable to that contact

Implementation complexity:

High. It requires contact enrollment, public-key verification, key wrapping,
browser key import/storage UX, and careful failure handling.

Fit for emergency use:

Strong long-term fit if setup happens before an emergency and the contact UX is
simple under stress.

### 3. One-Time Recovery Phrase Or File Import

How it works:

The trusted contact receives a recovery phrase, QR code, or key file through an
out-of-band process. During emergency access, the browser imports that material
and uses it to unwrap or derive the media key.

What the backend sees:

- emergency token and read requests
- encrypted evidence and metadata
- no recovery phrase or file unless the viewer uploads it, which should be
  avoided

What the browser sees:

- recovery phrase or imported file contents
- media keys or unwrapping keys derived from that material
- decrypted plaintext

UX strengths:

- does not require the contact to have preinstalled software
- can work when contact public-key enrollment was not completed
- simple enough for a manual disaster-recovery path

Security weaknesses:

- phrases and files are easy to copy, photograph, lose, or store insecurely
- phishing and fake viewer pages are a significant risk
- malicious viewer JavaScript can steal imported recovery material
- passphrase-derived approaches require careful KDF choices and must not use
  custom cryptography

Implementation complexity:

Moderate to high, depending on recovery material format and verification UX.

Fit for emergency use:

Useful as a fallback or early prototype, but risky as the primary model unless
the recovery material is strongly protected and clearly explained.

### 4. Separate Trusted Contact App

How it works:

A native or separately distributed trusted contact app receives the emergency
token, downloads encrypted evidence, and performs decryption outside the web
viewer. The web viewer may still show metadata and download links.

What the backend sees:

- emergency token and read requests from the app
- encrypted evidence and metadata requests
- no raw private keys or plaintext in normal operation

What the browser sees:

- possibly only metadata and encrypted download links
- no decryption keys if decryption happens entirely in the app

UX strengths:

- reduces reliance on JavaScript served by the same backend
- can use platform key storage and signed app distribution
- may handle large files, background work, and local secure storage better than
  a browser

Security weaknesses:

- users must install and trust another app before or during an emergency
- app distribution, updates, signing, and platform support become operational
  requirements
- phishing and token-sharing problems still exist

Implementation complexity:

High. This likely belongs after key custody and API formats are stable.

Fit for emergency use:

Good for higher assurance, but poor if contacts have not installed or prepared
the app before the emergency.

### 5. Static/Signed Viewer Bundle

How it works:

The decrypting viewer is a static bundle with a pinned release, signature, hash,
or independent hosting path. Contacts use that viewer to fetch encrypted
evidence and decrypt locally. The goal is to reduce trust in dynamic JavaScript
served by the incident backend.

What the backend sees:

- emergency token and encrypted data requests
- no raw keys if the static viewer keeps keys local
- possibly cross-origin requests if hosted separately

What the browser sees:

- pinned viewer code
- emergency token
- decryption capability
- encrypted evidence and decrypted plaintext

UX strengths:

- can improve confidence that the viewer code is the reviewed release
- supports offline or separately hosted verification paths
- can be paired with contact-wrapped keys

Security weaknesses:

- signature/hash verification is hard to make understandable in an emergency
- if the backend still serves the viewer, compromise can replace the page unless
  the contact has an independent verification route
- browser and device compromise remain in scope
- cross-origin and CORS design may complicate deployment

Implementation complexity:

High for a production-quality verification story, moderate for a local static
proof of concept.

Fit for emergency use:

Promising as a mitigation for malicious-server risk, but it needs careful UX so
contacts do not face confusing verification steps during an emergency.

### 6. Browser Decryption Deferred In Favour Of Non-Browser Client

How it works:

The project documents browser limitations and prioritizes a native trusted
contact client or offline decrypt tool before adding browser decryption.

What the backend sees:

- emergency token and encrypted evidence requests from the chosen client
- no raw keys or plaintext in normal operation

What the browser sees:

- metadata and encrypted download links only
- no decryption keys if browser decryption is deferred

UX strengths:

- avoids shipping security-sensitive browser crypto before the key custody model
  is settled
- gives time to design contact keys, wrapping, and live stream sessions
- can produce a higher-assurance decrypt path first

Security weaknesses:

- trusted contacts may have a harder time accessing evidence quickly
- non-browser clients still need distribution, updates, and support
- deferral does not solve emergency availability by itself

Implementation complexity:

Low for now, high later in the chosen non-browser client.

Fit for emergency use:

Reasonable if the project values assurance over immediate web convenience, but
it delays the easiest contact access path.

## URL Fragment Model

URL fragments are not sent in normal HTTP requests. In a URL such as
`https://example.test/e/token#key=value`, the browser requests `/e/token` and
keeps `#key=value` client-side. That can keep key material out of backend HTTP
handlers, reverse-proxy access logs, and referrer paths.

JavaScript running on the page can read the fragment locally through browser
APIs. A future viewer could import a fragment-carried key, use it to unwrap a
media key, then remove the fragment from the visible location bar after import.

Fragments are still sensitive. They can leak through:

- screenshots or screen sharing
- copy/pasted URLs
- browser history behavior
- browser extensions
- local malware or device compromise
- malicious JavaScript served by the viewer origin
- user support transcripts or logs

Fragment keys are therefore not a complete solution against a compromised
server. If the server can serve modified JavaScript, that JavaScript can read
the fragment and send it away. Fragment delivery mainly helps against passive
request logging and passive server storage compromise.

## Malicious Server Limitation

Browser decryption served by the same backend is useful against passive
storage, database, and blob compromise. It is not full end-to-end protection
against a malicious or compromised server that can serve modified JavaScript.

Possible mitigations:

- strict CSP
- static assets
- no inline script
- Subresource Integrity where applicable
- signed or static viewer bundle
- separate trusted viewer app
- pinned viewer release
- offline verification/decryption tool

These mitigations help, but they do not erase the core limitation. If the
backend controls the HTML and JavaScript delivered at access time, a fully
compromised backend can try to alter the decrypting code, hide warnings, or
capture keys and plaintext. Higher-assurance designs need an independently
verified client, signed static release, or offline decrypt path.

## Web Crypto Considerations

Future browser decryption should use stable browser crypto APIs such as Web
Crypto. It must not implement custom cryptographic primitives.

AES-GCM compatibility:

- The current simulator envelope uses `AES-256-GCM`, 32-byte keys, and 12-byte
  nonces.
- Web Crypto supports AES-GCM with 256-bit keys in modern browsers.
- The current envelope stores the nonce in the JSON header as base64url without
  padding.
- The current envelope stores deterministic associated data in the JSON header.
- The AES-GCM authentication tag is included in the ciphertext bytes, which is
  compatible with common Web Crypto AES-GCM usage.

Parsing requirements:

- verify the `SRCENC1\n` magic bytes
- read the 32-bit big-endian header length
- reject missing, truncated, oversized, or non-UTF-8 headers
- parse the JSON header
- verify version, scheme, algorithm, key ID, nonce length, and associated data
- pass the exact associated data bytes into AES-GCM decrypt

Associated data requirements:

Decryption must use the same incident ID, stream ID, media type, and positive
chunk index that the client used during encryption. If the viewer derives
associated data from bundle manifests, it must treat manifest metadata as
security-sensitive input and fail closed on mismatches.

Bundle and memory concerns:

The current emergency download format is a ZIP bundle. Browser-side decryption
of downloaded bundles requires a ZIP-reading strategy before decrypting the
`.enc` entries. Large bundles can be expensive to read fully into memory. Future
design should consider:

- file-by-file processing instead of whole-bundle buffering
- worker-based decryption to avoid blocking the page
- progress and cancellation UI
- size limits and browser memory failure handling
- clearing raw keys and plaintext references where practical
- avoiding plaintext logging, debug globals, URL parameters, or DOM leaks

Live stream considerations:

Completed bundles and live streams may need different handling. Live chunks may
need a stream/session key model, partial manifests, reconnect behavior, key
rotation, and a way for late contacts to obtain the right wrapped keys.

## Emergency UX

A future emergency browser flow should be designed for a trusted contact under
stress.

Possible flow:

1. The trusted contact opens an emergency link.
2. The viewer validates the emergency token and shows incident status.
3. The contact provides or already has a decryption capability.
4. The dashboard shows live status, latest checkin, and location if available.
5. Completed encrypted bundles can be downloaded and decrypted locally.
6. Decrypted output can be saved locally or viewed through a carefully designed
   plaintext handling path.
7. If live streaming exists, the viewer decrypts live chunks using a separate
   stream/session key model.

Failure handling matters. If decryption fails, the viewer should distinguish
safe, user-actionable causes without leaking secrets:

- missing decryption capability
- wrong contact key
- revoked or expired emergency token
- unsupported envelope version
- associated data mismatch
- corrupted or incomplete bundle
- browser compatibility or memory failure

The viewer should still show token-authorized metadata when decryption fails,
as long as the token remains valid and the design allows metadata visibility.

## Threat Model Impacts

Stolen emergency token:

A stolen token grants read access to the token-scoped emergency viewer and
encrypted bundles. It should not grant decryption unless the attacker also has a
decryption capability or a future design deliberately makes the token carry one.

Stolen decryption fragment or capability:

Anyone with both the emergency token and the decryption capability may decrypt
the relevant evidence. Fragment-carrying links and recovery files must be
handled as secrets.

Compromised browser:

A compromised browser, browser profile, extension, or local device can capture
tokens, keys, decrypted plaintext, downloads, and screenshots. Browser-side
decryption does not protect against this.

Compromised backend:

A passive backend compromise may still be unable to decrypt media without keys.
An active backend compromise can serve malicious JavaScript, alter manifests,
omit chunks, block access, or attempt to capture browser-entered keys.

Malicious JavaScript:

Malicious JavaScript can read URL fragments, imported keys, wrapped-key
plaintext after unwrap, media keys, decrypted chunks, and DOM-visible
plaintext. This is the main limit of same-origin browser decryption.

Compromised trusted contact device:

If the contact device or contact private key is compromised, the attacker may
decrypt any evidence available to that contact. Contact revocation can limit
future wrapping but cannot erase already downloaded data.

Network attacker with HTTPS:

TLS should protect tokens, encrypted bundles, and viewer assets in transit
against ordinary network attackers. A network attacker with a compromised CA,
compromised reverse proxy, or endpoint control may still attack the flow.

Logs and referrers:

Emergency tokens in paths are sensitive. Decryption material should not be sent
in paths or query strings. `Referrer-Policy: no-referrer`, no-store responses,
and reverse-proxy log redaction remain important.

Screenshots and browser history:

Emergency links, fragment keys, decrypted evidence, and dashboard metadata can
leak through screenshots, screen sharing, history behavior, clipboard managers,
downloads, and local file previews.

## Recommended Direction

Use a phased approach.

Phase 1: document browser decryption constraints.

Keep this document and the key custody document clear that browser decryption is
future work and does not weaken the current ciphertext-only backend.

Phase 2: simulator/browser-compatible envelope verification.

Verify that the current simulator envelope can be parsed and decrypted with
browser-compatible AES-GCM semantics in a standalone prototype. Do not add
production viewer decryption yet.

Phase 3: static proof-of-concept viewer for downloaded bundles.

Build a local or static prototype that imports a simulator key file, parses a
downloaded encrypted bundle, and decrypts chunks locally. Keep it separate from
the production emergency viewer until the trust model is accepted.

Phase 4: trusted contact key wrapping.

Design and prototype contact public keys, wrapped media keys, key IDs,
revocation behavior, and contact-key loss handling. This should align with
`docs/key-custody.md`.

Phase 5: live stream/session key model.

Define how live chunks, reconnects, late contact enrollment, stream key
rotation, and partial manifests work before adding browser live decryption.

Phase 6: production browser viewer decision.

Decide whether the browser viewer is acceptable as the main trusted-contact UX,
or whether a separate trusted contact app or offline decrypt tool should be the
higher-assurance path.

## Open Questions

- Should browser decryption be the first trusted-contact UX, or should a native
  trusted contact app come first?
- Should a fragment ever carry a raw media key, or only an intermediate
  wrapping/unwrapping capability?
- How should a future viewer clear or hide URL fragments after import?
- How are contact public keys verified before incidents?
- Should the emergency token and decryption capability be delivered separately?
- What bundle or chunk API shape is needed for memory-safe browser decryption?
- Is a static/signed viewer bundle required before production browser
  decryption?
- How should decrypted output be displayed or saved without creating surprising
  plaintext caches?
- What metadata should remain visible when decryption fails?
- How should live stream session keys differ from completed bundle keys?
- What browser versions and platforms are in scope?
- What tests are required before browser decryption can be trusted?
