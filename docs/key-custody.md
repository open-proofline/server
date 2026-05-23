# Key Custody And Emergency Access

This document defines the intended production key custody direction for Safety
Recorder. It is a design document only. It does not change the current backend,
API, database schema, simulator envelope, or encryption code.

## Summary

Safety Recorder currently keeps the backend ciphertext-only. Clients upload
already-encrypted chunk bytes, the backend validates hashes over those ciphertext
bytes, and evidence bundles contain encrypted chunks plus JSON manifests. The
backend does not store media keys and does not decrypt media.

That model is a good confidentiality baseline, but it is not sufficient for the
future product. During a real emergency, the iPhone may be lost, damaged,
powered off, taken, destroyed, or otherwise unavailable. Production key material
therefore must not exist solely on the iPhone.

The preferred long-term direction is a hybrid key custody model:

- clients encrypt media before upload
- the backend stores ciphertext chunks
- the backend may store wrapped or encrypted media keys
- trusted contacts can eventually decrypt emergency evidence without needing the
  phone to survive
- client-side emergency decryption is preferred where practical
- server escrow or server-side decryption is allowed only as an explicit
  break-glass or dead-man-switch mode

Server-side decryption is not forbidden forever, but it must never be introduced
accidentally. Any future key custody, recovery, escrow, browser decryption, or
server decryption work must be deliberate, documented, tested, and
threat-modeled.

## Goals

- Preserve evidence confidentiality where practical.
- Keep evidence accessible during emergencies.
- Allow trusted contacts to access emergency evidence.
- Support future live GPS and emergency dashboard use.
- Support future live audio/video streaming design.
- Support future dead-man-switch flows.
- Avoid making the iPhone the only place where usable keys exist.
- Avoid casual or raw server access to media keys.
- Make key custody decisions auditable and documented.

## Non-Goals For This Milestone

- No implementation.
- No iOS code.
- No browser decryption implementation.
- No server-side decryption implementation.
- No new API routes.
- No database schema changes.
- No playable media export.
- No push, SMS, or Messenger delivery.
- No user account system.

## Key Custody Models Considered

### 1. Client-Only Keys

How it works:

The recording client generates media keys and stores them only on the phone, for
example in the iOS Keychain. Uploaded chunks are encrypted before upload. The
backend never receives media keys or wrapped copies of those keys.

What it protects against:

- backend database compromise
- blob storage compromise
- passive server operator access to uploaded chunks
- accidental key leakage through server logs or bundle manifests

What it does not protect against:

- loss, destruction, seizure, or unavailability of the phone
- emergency-contact access when the phone cannot participate
- dead-man-switch cases where the device is unavailable at trigger time
- phone compromise before or during recording

Availability impact:

Poor for the core emergency requirement. Evidence may be safely stored but
unusable if the only decryption key is gone.

Operational complexity:

Low. The server remains simple and ciphertext-only.

Implementation complexity:

Low to moderate. The client must generate, persist, back up, and use keys
correctly.

Emergency UX impact:

Poor if the phone is unavailable. Trusted contacts may receive encrypted bundles
they cannot decrypt.

Trust assumptions:

The phone remains available and trustworthy, or the user has an out-of-band key
backup.

Fit for this project:

Not sufficient as the production model. It remains useful as a confidentiality
baseline and a development starting point.

### 2. Contact-Wrapped Keys

How it works:

The user pre-registers trusted contacts. Each contact has a public key. For each
incident or stream, the client encrypts media with a media key and wraps that
media key to one or more contact public keys. The backend stores only ciphertext
chunks and wrapped media keys.

What it protects against:

- passive compromise of blob storage
- passive database access, assuming wrapped keys cannot be opened by the
  attacker
- normal backend operation seeing raw media keys
- phone loss after wrapped keys have been uploaded

What it does not protect against:

- compromised trusted contact devices
- malicious replacement of contact public keys during registration
- active server attacks that withhold, replace, or omit wrapped keys
- missing access if no trusted contact key was registered or uploaded in time

Availability impact:

Good when contact keys are available and current. Evidence remains usable by
trusted contacts even if the phone is gone.

Operational complexity:

Moderate. Contact enrollment, verification, revocation, key loss, and key
rotation must be designed.

Implementation complexity:

Moderate to high. The client, backend metadata, and emergency viewer or trusted
contact client must all understand wrapped keys.

Emergency UX impact:

Good if the contact has a usable private key and a clear decryption flow. Poor
if key setup was incomplete or the contact lost their private key.

Trust assumptions:

Trusted contacts protect their private keys. The user correctly verifies contact
keys during setup. The backend stores wrapped keys faithfully.

Fit for this project:

Strong fit as the default future direction.

### 3. Browser/Client-Side Emergency Viewer Decryption

How it works:

The emergency viewer downloads encrypted evidence and wrapped keys, then
decrypts in the browser or another trusted client. Key material may be delivered
out of band, opened by a contact private key, or placed in a URL fragment so it
is not sent in HTTP requests.

What it protects against:

- passive compromise of server storage
- passive database compromise
- backend access to plaintext during ordinary bundle generation
- leakage of fragment-carried keys through HTTP request paths or query strings

What it does not protect against:

- an actively compromised backend serving malicious JavaScript
- compromised reverse proxy altering viewer assets
- compromised contact browser or device
- accidental key exposure through screen sharing, browser extensions, or local
  downloads

Availability impact:

Good if the contact has the decryption capability and a capable browser.

Operational complexity:

Moderate. Browser support, static asset integrity, CSP, caching, download
handling, and recovery UX need careful review.

Implementation complexity:

Moderate to high. Browser crypto, large-file handling, key import, streaming
decrypt, and export UX must be built and tested.

Emergency UX impact:

Potentially excellent because trusted contacts can use the emergency viewer
directly. It must be designed for stressful, low-context situations.

Trust assumptions:

Contacts trust the emergency viewer code they receive at access time, unless a
future app or independently verified static client is used.

Fit for this project:

Good fit as a future client-side decryption path, especially paired with
contact-wrapped keys. It does not remove the need to think about active server
compromise.

### 4. Server Escrow / Break-Glass Access

How it works:

The backend or deployment environment stores an encrypted or otherwise protected
way to recover media keys. During an explicit emergency, dead-man-switch, or
break-glass event, the server can obtain key access or perform server-side
decryption according to a configured policy.

What it protects against:

- destroyed or unavailable phone
- lost trusted contact private keys, depending on policy
- emergency cases where browser/client decryption is unavailable
- operational inability to help contacts access evidence

What it does not protect against:

- malicious operator misuse if policy and controls are weak
- compromised server or deployment secrets
- compelled access depending on jurisdiction and hosting model
- accidental plaintext exposure if server-side decryption is overused

Availability impact:

High when configured correctly. This is the strongest availability option.

Operational complexity:

High. It needs policy, audit logs, restricted access, incident review,
configuration warnings, secret storage, backup, restore, and rotation plans.

Implementation complexity:

High. Server key custody and decryption create a new security boundary and must
be tested separately from normal ciphertext-only flows.

Emergency UX impact:

Potentially good for dead-man-switch scenarios. It can also be dangerous if it
silently weakens confidentiality or creates false confidence.

Trust assumptions:

The operator, deployment host, access policy, audit trail, and secret storage
are trustworthy enough for the user's risk model.

Fit for this project:

Acceptable only as an optional future mode. It must be disabled by default or
separately configured, clearly documented, audited, and treated as a deliberate
break-glass capability.

### 5. Threshold Or Multi-Party Recovery

How it works:

Key recovery requires multiple parties or shares, such as two trusted contacts,
one contact plus server escrow, or another threshold arrangement. No single
party can recover the media key alone.

What it protects against:

- single trusted contact compromise
- casual operator misuse
- some forms of accidental single-party key disclosure
- unilateral access when policy requires consent from multiple parties

What it does not protect against:

- unavailability of enough parties during an emergency
- collusion between threshold participants
- bad enrollment or recovery UX
- compromised client code creating or distributing shares incorrectly

Availability impact:

Mixed. It improves misuse resistance but can reduce emergency access reliability.

Operational complexity:

High. Enrollment, recovery, revocation, lost shares, and support flows are more
complex than simple wrapped keys.

Implementation complexity:

High. Future implementation must use stable, reviewed threshold or secret
sharing libraries and must not invent cryptographic primitives.

Emergency UX impact:

Risky unless very carefully designed. In emergencies, requiring several people
or devices may delay access.

Trust assumptions:

Enough parties remain available and honest. The user accepts the complexity.

Fit for this project:

Potentially useful later for high-risk users or escrow modes, but not the first
production default.

### 6. Hybrid Model

How it works:

The client encrypts media before upload. The backend stores ciphertext chunks
and wrapped media keys. Trusted contacts are the default recovery path. Browser
or app-based client-side decryption is preferred where practical. Optional
server escrow or break-glass access can be added as a separate, explicit mode
for dead-man-switch and emergency-access cases.

What it protects against:

- passive storage compromise
- phone loss after keys or wrapped keys are uploaded
- ordinary backend operation seeing plaintext
- some trusted-contact key loss, if optional escrow is enabled

What it does not protect against:

- all active server compromise scenarios
- compromised contact devices
- malicious JavaScript served by a compromised backend
- operator misuse if break-glass controls are weak
- incomplete contact setup

Availability impact:

Best overall balance. It avoids making the phone the single point of failure
while preserving ciphertext-only backend operation for the default path.

Operational complexity:

Moderate by default, high when escrow modes are enabled.

Implementation complexity:

High across all phases, but it can be delivered incrementally.

Emergency UX impact:

Best fit if the contact flow is made simple: emergency token plus decryption
capability, with clear handling for missing or revoked keys.

Trust assumptions:

Contacts protect private keys. The backend stores wrapped keys and ciphertext
correctly. Escrow deployments accept and document their stronger server trust
assumptions.

Fit for this project:

Recommended.

## Recommended Ultimate Model

The recommended direction is a hybrid model.

Default mode:

- The client creates media keys for each incident or stream.
- The client encrypts chunks before upload.
- The backend stores ciphertext chunks and metadata.
- The backend stores wrapped or encrypted copies of media keys for trusted
  contacts.
- The backend does not store raw media keys in the default mode.
- The emergency viewer or a future trusted client performs decryption
  client-side where practical.

Optional future mode:

- A deployment may enable server escrow or break-glass key access for
  dead-man-switch and emergency-access cases.
- This mode must be disabled by default or configured separately from the normal
  ciphertext-only path.
- It must have explicit access policy, audit logging, rate limiting,
  operational warnings, and incident-review expectations.
- It may use deployment-specific key storage such as a KMS, HSM, locked local
  secret store, or another reviewed secret-management system.

This document decides the long-term direction: contact-wrapped keys plus
client-side decryption should be the default production path, with server
escrow or server-side decryption reserved for explicit break-glass modes.

## Key Hierarchy

Future implementations should keep the hierarchy simple and auditable.

Suggested hierarchy:

- Device identity key: a long-lived client key pair controlled by the user's
  device or account model in a future client.
- Contact identity key: a long-lived public/private key pair controlled by each
  trusted contact.
- Incident key: an optional per-incident wrapping or coordination key for all
  streams in one incident.
- Stream media key: the symmetric key used to encrypt chunks in one media
  stream.
- Chunk nonce: a fresh nonce for each encrypted chunk under the relevant stream
  media key.
- Key ID: a non-secret identifier used to match encrypted chunks and wrapped
  keys with the correct decrypting key.
- Wrapped media key: an encrypted copy of an incident or stream key for a
  device, contact, recovery method, or escrow mode.
- Server escrow key: an optional future deployment key used only in explicit
  break-glass mode.

The current simulator uses one AES-256-GCM key for development/test chunks. A
production client should prefer per-stream media keys so compromise or rotation
can be contained to a smaller unit, especially for long-running or multi-media
incidents. A per-incident key may still be useful as a parent wrapping key or
for incident-level metadata, but long live streams should not depend on a single
unbounded encryption context.

Each encrypted chunk must use a unique nonce for its media key. Key IDs and
non-secret envelope metadata may be stored in manifests and database rows. Raw
media keys, contact private keys, escrow private keys, and plaintext must not be
logged or placed in bundle manifests.

Future implementation must use stable, reviewed cryptographic libraries. Do not
implement custom AEAD, block modes, padding, MACs, KDFs, random generators,
public-key wrapping, threshold recovery, or secret-sharing primitives.

## Emergency Contact Access

Trusted contact access should be designed around pre-registration.

Possible flow:

1. A trusted contact generates or imports a public/private key pair.
2. The user verifies and registers the contact public key.
3. The recording client creates stream media keys during an incident.
4. The client uploads encrypted chunks and wrapped media keys for the selected
   trusted contacts.
5. The trusted contact receives an emergency viewer token through a separate
   sharing path.
6. The emergency viewer or future trusted contact app downloads ciphertext
   chunks, bundle manifests, and the contact-wrapped key material.
7. The contact private key unwraps the media key and decrypts evidence
   client-side.

The emergency viewer token should authorize read access to incident metadata and
encrypted evidence. It should not, by itself, be the only decryption capability
unless the system intentionally chooses a weaker bearer-token-only emergency
mode.

Lost contact keys must be handled explicitly. If a contact loses their private
key, existing media keys wrapped only to that contact may be unrecoverable by
that contact. The user can remove or revoke a contact for future incidents, but
revocation cannot reliably make already-downloaded ciphertext, wrapped keys, or
plaintext disappear. Future schema and API design should distinguish:

- removing a contact from future incidents
- stopping new key wrapping for a contact
- revoking emergency viewer tokens
- rotating media keys for ongoing streams
- marking older wrapped keys as no longer offered by the server

Future contact decryption may happen in a browser, a native app, or another
trusted client. App-based decryption can reduce malicious-server JavaScript
risk, but it raises distribution and platform-support questions.

## Browser Decryption Considerations

Browser decryption can make emergency access much easier because contacts can
use the same emergency viewer URL to inspect and decrypt evidence. It also has
important limits. A focused browser decryption design spike is available in
[browser-decryption.md](browser-decryption.md).

URL fragment key delivery can keep key material out of HTTP requests because
fragments are not sent to the server. That can be useful for recovery links or
manual key handoff. Fragment keys can still leak through screenshots, browser
history behavior, extensions, copy/paste, device compromise, and user error.

JavaScript served by the backend is a trust problem. A compromised backend or
reverse proxy can potentially serve malicious JavaScript that exfiltrates keys
or plaintext after the contact opens the viewer. Strict CSP, external static
assets, no inline scripts, no-store headers, and Subresource Integrity can help
reduce accidental exposure, but they do not fully solve malicious-server risk
when the server controls the page.

Browser decryption is therefore stronger against passive storage compromise
than against active server compromise. A future high-assurance design may need a
separately distributed trusted contact app, signed static assets, reproducible
viewer builds, or another independent verification story.

## Server Escrow / Break-Glass Considerations

Server-side key access may be acceptable when emergency availability is more
important than keeping all decryption capability away from the server. Examples
include:

- a dead-man-switch trigger
- emergency access escalation after configured conditions are met
- recovery when the phone and contact keys are unavailable
- deployments where a trusted operator is explicitly part of the safety model

Any server escrow mode must be explicit. It should not share code paths or
configuration defaults that make backend decryption easy to enable by accident.
Before implementation, the design must define:

- trigger conditions
- who or what can authorize access
- audit logging requirements
- access policy and review process
- rate limits and abuse controls
- operator warnings
- incident review expectations
- key backup and restore expectations
- key rotation and revocation behavior
- deployment secret storage options

Future storage options may include a cloud KMS, HSM, locked local secret store,
hardware-backed local key, or another deployment-specific secret-management
system. This repository should not assume a cloud service by default. Any such
choice must be documented as a deployment decision.

Server-side decryption output is also a separate product decision. Producing
plaintext or playable exports has different logging, caching, retention, and
access-control risks from serving encrypted bundles.

## Metadata And Live Dashboard Implications

Media chunk encryption does not automatically protect all emergency data.
Incident IDs, stream IDs, media types, timestamps, byte counts, ciphertext
hashes, stream state, and token-scoped summaries are visible to the backend
today. Checkins can include location metadata. Future live GPS and emergency
dashboard features may intentionally expose some metadata to the backend or
emergency viewer for usability.

Live GPS data may need a different privacy model from encrypted media chunks. A
dashboard that shows current location to trusted contacts may require backend
visibility, contact-side decryption, or a mixed design where coarse status is
server-visible and sensitive details are encrypted.

Live audio/video streaming may also require a different key/session model than
completed chunk bundles. Long-running streams need key rotation, late contact
enrollment behavior, partial stream access, reconnect handling, and clear rules
for when wrapped keys are uploaded.

Emergency dashboard usability may trade off against strict confidentiality. The
project should make those tradeoffs explicit rather than letting metadata and
key handling emerge accidentally from implementation details.

## Threat Model Impacts

Compromised backend:

Default contact-wrapped keys keep raw media keys and plaintext away from normal
backend storage, but an active backend may still hide evidence, tamper with
metadata, omit wrapped keys, serve malicious viewer JavaScript, or capture keys
entered into a browser viewer. Server escrow mode increases backend trust
requirements.

Compromised database:

An attacker may learn metadata and obtain wrapped keys. Contact-wrapped keys
protect media only if wrapping keys remain secure and the wrapping scheme is
correct.

Stolen blob storage:

Encrypted chunks should remain confidential without the relevant media keys.
Chunk metadata and bundle manifests may still reveal timing, media type, and
size information.

Compromised emergency viewer token:

A token should grant read access to metadata and encrypted evidence for its
incident. It should not grant decryption by itself unless a future mode
explicitly chooses token-carried or token-derived decryption capability.

Malicious or compromised reverse proxy:

A proxy can observe token-bearing request paths unless logs are redacted, block
or alter responses, and potentially modify browser decryption assets if it
terminates TLS and controls the origin path. Decryption keys should not be sent
in request paths or query strings.

Compromised trusted contact device:

An attacker may obtain contact private keys, decrypt available evidence, or use
valid emergency tokens. Contact revocation limits future wrapping but cannot
erase already accessed material.

Destroyed iPhone:

The hybrid model is designed for this case. Uploaded ciphertext and wrapped keys
remain available after the phone is gone, assuming upload completed and trusted
contact or escrow recovery material exists.

Maintainer/operator misuse:

Default contact-wrapped keys limit casual operator access to plaintext, but
operators can still expose, delete, withhold, or alter stored evidence. Escrow
modes require stronger audit and access policy because they can enable key
access.

Dead-man-switch false positive or false negative:

A false positive may expose evidence or keys before the user intended. A false
negative may prevent emergency access. Future designs must make trigger policy,
cooldowns, cancellation, notification, and audit behavior explicit.

## Open Questions

- Should media encryption use per-stream media keys only, or a per-incident
  parent key plus per-stream keys?
- What exact public-key wrapping scheme should be used for contact-wrapped
  keys?
- How are trusted contact public keys verified during enrollment?
- Does the project need a future account model, or can contact enrollment remain
  local and token-scoped?
- How should contacts recover from lost private keys?
- Can contacts be added to an incident after recording has started, and which
  existing media keys should be wrapped for them?
- What metadata should be encrypted, and what must remain server-visible for the
  emergency dashboard?
- Should browser decryption be the first contact UX, or should a native trusted
  contact app come first?
- Are signed/static viewer assets or reproducible builds needed before browser
  decryption is considered acceptable?
- Should server escrow be supported at all in the first production release, or
  deferred until after contact-wrapped keys are proven?
- What audit log fields are safe to store without leaking tokens, keys,
  plaintext, or sensitive safety data?
- What retention, backup, and deletion policies apply to wrapped keys?

## Proposed Implementation Phases

Phase 1: design and docs.

Create this design, update the security and encryption documentation, and keep
the current backend ciphertext-only.

Phase 2: contact-wrapped key prototype in the simulator.

Prototype media-key wrapping and bundle metadata in development flows only. Do
not add production server decryption.

Phase 3: browser/client-side decrypt prototype.

Prototype emergency viewer decryption with strict CSP, no-store behavior, and a
clear explanation of malicious-server limitations, following the constraints in
[browser-decryption.md](browser-decryption.md).

Phase 4: iOS Keychain and contact-key planning.

Design client key generation, Keychain storage, contact public-key enrollment,
rotation, and revocation before implementing the iOS client.

Phase 5: emergency access and dead-man-switch key policy.

Define trigger behavior, access policy, audit expectations, notification,
cancellation, and false-positive/false-negative handling.

Phase 6: optional server escrow or break-glass implementation.

Implement only if explicitly accepted. Keep it separately configured,
audited/logged, rate-limited, and documented with deployment warnings.
