# Key Custody And Emergency Access

This document defines the intended production key custody direction for Proofline. It is a design document only. It does not change the current backend, API, database schema, simulator envelope, or encryption code.

## Summary

Proofline currently keeps the backend ciphertext-only. Clients or the simulator upload already-encrypted chunk bytes, the backend validates hashes over those ciphertext bytes, and evidence bundles contain encrypted chunks plus JSON manifests. The backend does not store media keys and does not decrypt media.

That model is a good confidentiality baseline, but it is not sufficient for the future product. During a real emergency, safety check, or high-risk interaction, the user's phone may be lost, damaged, powered off, taken, destroyed, or otherwise unavailable. Production key material therefore must not exist solely on the phone.

The preferred long-term direction is a hybrid key custody model:

- clients encrypt media before upload
- the backend stores ciphertext chunks
- the backend may store wrapped or encrypted media keys
- trusted contacts can eventually decrypt authorised incident evidence without needing the phone to survive
- client-side or trusted-contact-side decryption is preferred where practical
- server escrow or server-side decryption is allowed only as an explicit break-glass or dead-man-switch mode

Server-side decryption is not forbidden forever, but it must never be introduced accidentally. Any future key custody, recovery, escrow, browser decryption, or server decryption work must be deliberate, documented, tested, and threat-modeled.

## Goals

- Preserve evidence confidentiality where practical.
- Keep authorised evidence accessible when the user's phone is unavailable.
- Allow trusted contacts to access emergency or safety-check evidence when policy permits it.
- Support non-emergency interaction records without forcing emergency escalation.
- Support future live GPS and incident dashboard use.
- Support future live audio/video streaming design.
- Support future dead-man-switch flows.
- Avoid making the phone the only place where usable keys exist.
- Avoid casual or raw server access to media keys.
- Make key custody decisions auditable and documented.

## Non-Goals For This Milestone

- No implementation.
- No iOS, Android, or web-client code.
- No browser decryption implementation.
- No server-side decryption implementation.
- No new API routes.
- No database schema changes.
- No first-class incident type or escalation-policy schema.
- No playable media export.
- No push, SMS, or Messenger delivery.
- No user account system.

## Incident Mode Implications

Planned incident modes change access policy expectations, not the current ciphertext-only backend behavior.

| Incident mode | Key custody implication |
|---|---|
| Emergency incident | Trusted contacts may need access if the phone is unavailable. Wrapped keys or explicit break-glass policy matter most here. |
| Interaction record | The default should be private capture with no automatic escalation. Sharing/export should be deliberate. |
| Safety check | Missed check-ins may trigger trusted-contact access. False positives and cancellation behavior need explicit policy before implementation. |
| Evidence note | Usually private by default. Export, retention, and deletion policy may matter more than live access. |

Do not treat incident-mode labels as sufficient access control. Account-owner, trusted-contact, public-link, admin/operator, and optional escrow access must be designed separately.

## Key Custody Models Considered

### 1. Client-Only Keys

The recording client generates media keys and stores them only on the phone, for example in the iOS Keychain or Android Keystore. Uploaded chunks are encrypted before upload. The backend never receives media keys or wrapped copies.

This protects against passive backend, database, and blob-storage compromise, but it fails the core availability requirement if the phone is lost, destroyed, seized, or unavailable. It remains useful as a development starting point and confidentiality baseline, but it is not sufficient as the production model.

### 2. Contact-Wrapped Keys

The user pre-registers trusted contacts. Each contact has a public key. For each incident or stream, the client encrypts media with a media key and wraps that media key to one or more contact public keys. The backend stores ciphertext chunks and wrapped media keys, but not raw keys.

This is the strongest default fit for Proofline. It preserves ordinary backend ciphertext-only operation while allowing trusted contacts to decrypt authorised evidence if the phone is gone. It requires careful design for contact enrollment, public-key verification, revocation, key loss, and contact-device compromise.

### 3. Browser Or Client-Side Viewer Decryption

The viewer downloads encrypted evidence and wrapped keys, then decrypts in the browser or another trusted client. Key material may be delivered out of band, opened by a contact private key, or placed in a URL fragment so it is not sent in HTTP requests.

This can make trusted-contact access easier, but it does not protect against an actively compromised backend serving malicious JavaScript. Browser decryption is stronger against passive storage compromise than active server compromise. A future high-assurance design may need signed/static viewer assets, reproducible builds, or a native trusted-contact client.

See [browser-decryption.md](browser-decryption.md).

### 4. Server Escrow / Break-Glass Access

The backend or deployment environment stores an encrypted or otherwise protected way to recover media keys. During an explicit emergency, dead-man-switch, or break-glass event, the server can obtain key access or perform server-side decryption according to configured policy.

This improves availability when the phone and trusted-contact keys are unavailable, but it creates serious operator, hosting, audit, and misuse risks. It is acceptable only as an optional future mode. It must be disabled by default or separately configured, clearly documented, audited, rate-limited, and treated as deliberate break-glass capability.

See [break-glass-key-access.md](break-glass-key-access.md).

### 5. Threshold Or Multi-Party Recovery

Key recovery requires multiple parties or shares, such as two trusted contacts, one contact plus server escrow, or another threshold arrangement. No single party can recover the media key alone.

This may reduce unilateral misuse but can hurt emergency availability if enough parties are not reachable. It may be useful later for high-risk users or escrow modes, but it should not be the first production default.

### 6. Hybrid Model

The client encrypts media before upload. The backend stores ciphertext chunks and wrapped media keys. Trusted contacts are the default recovery path. Browser or app-based client-side decryption is preferred where practical. Optional server escrow or break-glass access can be added as a separate explicit mode for dead-man-switch and emergency-access cases.

This is the recommended direction.

## Recommended Ultimate Model

Default mode:

- The client creates media keys for each incident or stream.
- The client encrypts chunks before upload.
- The backend stores ciphertext chunks and metadata.
- The backend stores wrapped or encrypted copies of media keys for trusted contacts.
- The backend does not store raw media keys in the default mode.
- The incident viewer or a future trusted client performs decryption client-side where practical.

Optional future mode:

- A deployment may enable server escrow or break-glass key access for dead-man-switch and emergency-access cases.
- This mode must be disabled by default or configured separately from the normal ciphertext-only path.
- It must have explicit access policy, audit logging, rate limiting, operational warnings, and incident-review expectations.
- It may use deployment-specific key storage such as a KMS, HSM, locked local secret store, or another reviewed secret-management system.

This document decides the long-term direction: contact-wrapped keys plus client-side decryption should be the default production path, with server escrow or server-side decryption reserved for explicit break-glass modes.

## Key Hierarchy

Future implementations should keep the hierarchy simple and auditable.

Suggested hierarchy:

- Device identity key: a long-lived client key pair controlled by the user's device or account model in a future client.
- Contact identity key: a long-lived public/private key pair controlled by each trusted contact.
- Incident key: an optional per-incident wrapping or coordination key for all streams in one incident.
- Stream media key: the symmetric key used to encrypt chunks in one media stream.
- Chunk nonce: a fresh nonce for each encrypted chunk under the relevant stream media key.
- Key ID: a non-secret identifier used to match encrypted chunks and wrapped keys with the correct decrypting key.
- Wrapped media key: an encrypted copy of an incident or stream key for a device, contact, recovery method, or escrow mode.
- Server escrow key: an optional future deployment key used only in explicit break-glass mode.

The current simulator uses one AES-256-GCM key for development/test chunks. A production client should prefer per-stream media keys so compromise or rotation can be contained to a smaller unit, especially for long-running or multi-media incidents.

Each encrypted chunk must use a unique nonce for its media key. Key IDs and non-secret envelope metadata may be stored in manifests and database rows. Raw media keys, contact private keys, escrow private keys, and plaintext must not be logged or placed in bundle manifests.

Future implementation must use stable, reviewed cryptographic libraries. Do not implement custom AEAD, block modes, padding, MACs, KDFs, random generators, public-key wrapping, threshold recovery, or secret-sharing primitives.

## Trusted Contact Access

Trusted contact access should be designed around pre-registration.

Possible flow:

1. A trusted contact generates or imports a public/private key pair.
2. The user verifies and registers the contact public key.
3. The recording client creates stream media keys during an incident.
4. The client uploads encrypted chunks and wrapped media keys for the selected trusted contacts.
5. The trusted contact receives an incident viewer token or future account-based access grant through a separate sharing path.
6. The viewer or future trusted contact app downloads ciphertext chunks, bundle manifests, and contact-wrapped key material.
7. The contact private key unwraps the media key and decrypts evidence client-side.

The viewer token should authorize read access to incident metadata and encrypted evidence. It should not, by itself, be the only decryption capability unless the system intentionally chooses a weaker bearer-token-only emergency mode.

Lost contact keys must be handled explicitly. If a contact loses their private key, existing media keys wrapped only to that contact may be unrecoverable by that contact. Future schema and API design should distinguish removing a contact from future incidents, stopping new key wrapping, revoking viewer tokens, rotating media keys, and marking older wrapped keys as no longer offered by the server.

## Metadata And Live Dashboard Implications

Media chunk encryption does not automatically protect all incident data. Incident IDs, stream IDs, media types, timestamps, byte counts, ciphertext hashes, stream state, and token-scoped summaries are visible to the backend today. Checkins can include location metadata.

Live GPS data may need a different privacy model from encrypted media chunks. A dashboard that shows current location to trusted contacts may require backend visibility, contact-side decryption, or a mixed design where coarse status is server-visible and sensitive details are encrypted.

Live audio/video streaming may also require a different key/session model than completed chunk bundles. Long-running streams need key rotation, late contact enrollment behavior, partial stream access, reconnect handling, and clear rules for when wrapped keys are uploaded.

Interaction records may be non-emergency but still highly sensitive. Do not assume lower urgency means lower confidentiality.

## Threat Model Impacts

Future key custody work must consider:

- compromised backend or malicious viewer code
- compromised database or blob storage
- compromised viewer token
- malicious or compromised reverse proxy
- compromised trusted contact device
- destroyed or unavailable phone
- maintainer/operator misuse
- dead-man-switch false positives and false negatives
- accidental sharing/export of non-emergency interaction records

The hybrid model is designed to keep uploaded ciphertext useful after the phone is gone while limiting ordinary backend access to plaintext. Escrow modes increase backend trust requirements and must be explicit.

## Open Questions

- Should media encryption use per-stream media keys only, or a per-incident parent key plus per-stream keys?
- What exact public-key wrapping scheme should be used for contact-wrapped keys?
- How are trusted contact public keys verified during enrollment?
- What account model is required for web, iOS, and Android clients?
- How should contacts recover from lost private keys?
- Can contacts be added to an incident after recording has started, and which existing media keys should be wrapped for them?
- What metadata should be encrypted, and what must remain server-visible for the incident dashboard?
- Should browser decryption be the first contact UX, or should a native trusted contact app come first?
- Are signed/static viewer assets or reproducible builds needed before browser decryption is considered acceptable?
- Should server escrow be supported at all in the first production release, or deferred until after contact-wrapped keys are proven?
- What audit log fields are safe to store without leaking tokens, keys, plaintext, or sensitive safety data?
- What retention, backup, and deletion policies apply to wrapped keys?
- How do incident modes affect default sharing, retention, escalation, and access policies?

## Proposed Implementation Phases

Phase 1: design and docs.

Create this design, update the security and encryption documentation, and keep the current backend ciphertext-only.

Phase 2: protocol and incident-mode design.

Define first-class incident types, escalation policies, account/trusted-contact roles, and compatibility expectations before implementing public client workflows.

Phase 3: contact-wrapped key prototype in the simulator.

Prototype media-key wrapping and bundle metadata in development flows only. Do not add production server decryption.

Phase 4: browser/client-side decrypt prototype.

Prototype viewer decryption with strict CSP, no-store behavior, and a clear explanation of malicious-server limitations, following the constraints in [browser-decryption.md](browser-decryption.md).

Phase 5: iOS/Android keychain and contact-key planning.

Design client key generation, platform key storage, contact public-key enrollment, rotation, and revocation before implementing production mobile clients.

Phase 6: emergency access and dead-man-switch key policy.

Define trigger behavior, access policy, audit expectations, notification, cancellation, and false-positive/false-negative handling. See [break-glass-key-access.md](break-glass-key-access.md).

Phase 7: optional server escrow or break-glass implementation.

Implement only if explicitly accepted. Keep it separately configured, audited/logged, rate-limited, and documented with deployment warnings.
