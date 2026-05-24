# Break-Glass And Dead-Man-Switch Key Access

This document designs possible future server-assisted emergency key access for
Safety Recorder. It is a design document only. It does not implement
server-side decryption, key escrow, API routes, database schema changes,
background jobs, or dead-man-switch logic.

## Summary

Break-glass key access is an explicit emergency mode where a server, operator,
deployment secret store, or policy-controlled process can help recover or use
media keys when normal client-side access is unavailable. A dead-man-switch is a
related policy that grants or escalates access after configured conditions are
met, such as missed checkins or a device being offline too long.

This may be needed because Safety Recorder is meant to preserve evidence when a
phone is lost, damaged, powered off, taken, destroyed, or otherwise unavailable.
Contact-wrapped keys and browser/client-side decryption should be the default
future path, but some deployments may want an additional emergency recovery path
for cases where contacts cannot decrypt quickly enough.

This is not implemented today. The current backend remains ciphertext-only: it
stores encrypted chunk bytes, validates hashes over ciphertext, and produces
encrypted ZIP bundles. It does not store raw media keys or decrypt media.

Any future break-glass mode would increase backend, operator, and deployment
trust requirements. It must be explicit, separately configured, auditable,
tested, threat-modeled, and documented with clear deployment warnings. It must
never appear as an incidental side effect of unrelated key custody, viewer, or
simulator work.

## Availability Requirement

Production key custody must assume the iPhone may be unavailable during the
moment when evidence is needed most. The device may be:

- lost
- damaged
- powered off
- taken
- destroyed
- disconnected from the network
- unable to complete a final key-share upload

Phone-only keys are therefore not sufficient for this product. They protect
confidentiality well when the backend or blob storage is compromised, but they
can turn preserved evidence into unusable ciphertext if the phone is gone.

The preferred availability baseline is contact-wrapped key material: the client
encrypts media, uploads ciphertext, and uploads wrapped media keys for trusted
contacts. Break-glass access is a stronger availability option that should be
treated as optional and higher risk, not as the default path.

## Candidate Access Models

### 1. Server Stores Wrapped Key Material Only

How it works:

The backend stores encrypted or wrapped copies of incident or stream media keys.
Those keys are wrapped for trusted contacts, future devices, recovery keys, or
other explicitly designed recipients. The server can deliver wrapped material
but cannot unwrap it by itself.

What the server can access:

- ciphertext chunks
- metadata
- wrapped keys
- emergency token-gated summaries
- no raw media keys in normal operation

Operator trust requirements:

Moderate. Operators can delete, withhold, corrupt, or expose encrypted material,
but they cannot decrypt media without a recipient key.

Audit requirements:

Audit delivery and management of wrapped keys, especially key creation,
replacement, revocation, and download. Audit logs must not include raw keys,
plaintext, or raw emergency tokens.

Failure modes:

- no trusted contact key exists
- wrapped key upload fails before the phone is unavailable
- contact private key is lost
- server omits or corrupts wrapped keys
- revocation semantics are misunderstood

Usability during emergency:

Good if contacts are pre-enrolled and have a working decrypt path. Poor if setup
was incomplete.

Fit for personal/self-hosted deployment:

Strong fit as the default future production model. It preserves the current
ciphertext-only backend posture while improving availability.

### 2. Server Can Unwrap Keys Under Break-Glass Policy

How it works:

The backend or deployment environment stores media keys wrapped to a server
escrow key. Under an explicit break-glass policy, the server can unwrap the
media key and either return it to an authorized decrypting client or use it in a
closely controlled operation.

What the server can access:

- wrapped keys at rest
- raw media keys during authorized break-glass operations
- potentially plaintext if it uses those keys to decrypt

Operator trust requirements:

High. Operators or compromised server processes may be able to obtain media keys
if controls fail.

Audit requirements:

Every unwrap attempt must be logged with incident ID, timestamp, triggering
policy, caller or actor, decision, and outcome. Logs must never include raw
keys, plaintext, raw emergency tokens, uploaded bytes, or sensitive safety data
beyond minimal audit metadata.

Failure modes:

- unauthorized unwrap due to weak policy
- false dead-man-switch trigger
- escrow key loss
- escrow key compromise
- backup/restore mismatch makes old keys unusable
- user assumes confidentiality that no longer matches deployment reality

Usability during emergency:

Strong if policy is correctly configured and operationally available. It can
help when the phone and contact keys are unavailable.

Fit for personal/self-hosted deployment:

Potentially useful for a self-hosted deployment where the user explicitly trusts
the operator or is the operator. It should be disabled by default and clearly
marked as a higher-trust mode.

### 3. Server Decrypts/Transcodes Media Under Break-Glass Policy

How it works:

After a break-glass trigger, the server unwraps media keys and decrypts chunks
server-side. It may also merge, transcode, or produce a playable media export
for emergency contacts.

What the server can access:

- raw media keys
- plaintext media
- potentially playable exports
- decrypted temporary files or streams if implementation is not extremely
  careful

Operator trust requirements:

Very high. This mode makes the backend part of the plaintext handling path.

Audit requirements:

Audit must cover authorization, key access, decrypt/transcode start and finish,
output creation, output download, output deletion, and policy decisions. It
must not log raw keys, plaintext, decrypted filenames containing sensitive
content, raw tokens, or uploaded bytes.

Failure modes:

- plaintext cached, logged, indexed, backed up, or left on disk
- transcode errors destroy evidentiary confidence
- playable exports are mistaken for original evidence
- operator or server compromise exposes plaintext
- retention/deletion policy fails to cover derived plaintext

Usability during emergency:

Excellent for non-technical contacts if implemented safely, because playable
media is easier to use than encrypted bundles.

Fit for personal/self-hosted deployment:

Not recommended as an early production mode. It may be acceptable later only as
a deliberate, high-warning break-glass feature with retention, deletion,
logging, and audit design completed first.

### 4. n-of-m Trusted Contacts

How it works:

Media recovery requires a threshold of trusted contacts or shares. For example,
two of three contacts may need to approve or contribute key material before a
media key can be recovered.

What the server can access:

- ciphertext
- metadata
- wrapped shares or approvals
- no raw media key unless the threshold protocol deliberately reconstructs it on
  the server, which should be avoided by default

Operator trust requirements:

Moderate if reconstruction happens client-side. High if the server participates
in reconstruction or receives raw shares.

Audit requirements:

Audit share creation, contact enrollment, contact revocation, recovery attempts,
approvals, denials, and threshold satisfaction. Do not log share contents,
private keys, raw media keys, or plaintext.

Failure modes:

- not enough contacts are available during an emergency
- contacts lose private keys or shares
- contacts collude
- recovery UX is too slow under stress
- revocation and rotation become hard to explain

Usability during emergency:

Mixed. It improves misuse resistance but can slow urgent access.

Fit for personal/self-hosted deployment:

Useful for high-risk users later, but probably too complex for the first
production key custody milestone.

### 5. Maintainer/Operator Assisted Recovery

How it works:

A trusted maintainer or operator manually performs recovery steps, such as
validating a request, unlocking a local secret, approving break-glass access, or
helping a contact retrieve wrapped key material.

What the server can access:

Depends on the deployment. It may range from wrapped-key delivery only to raw
escrow key access.

Operator trust requirements:

High. The operator becomes part of the security and availability model.

Audit requirements:

Manual actions require especially clear audit trails: actor identity, request
source, incident ID, decision, timestamp, reason, and post-incident review
status. Public issue trackers or support logs must not receive sensitive
details.

Failure modes:

- operator unavailable
- operator error
- social engineering
- inconsistent manual policy
- support channel leaks sensitive information
- actions are not reproducible or reviewable

Usability during emergency:

Potentially helpful for a single-user self-hosted setup, but unreliable as the
only emergency path.

Fit for personal/self-hosted deployment:

Reasonable as an explicitly documented local procedure for advanced users. It
should not be implied as a public service or default support model.

### 6. External KMS/HSM/Secret Store

How it works:

Server escrow keys live in a deployment-controlled key management system,
hardware security module, hardware-backed local key, or locked local secret
store. The Safety Recorder backend calls that system only under break-glass
policy.

What the server can access:

- wrapped key material
- key unwrap results if policy allows
- raw media keys during authorized operations, unless the external system can
  keep unwrap/decrypt operations isolated

Operator trust requirements:

High, but potentially better controlled than local raw key files. Trust shifts
to the secret store, its policies, and deployment administration.

Audit requirements:

Audit must include both Safety Recorder policy decisions and the external
secret store's access logs. Clock sync, log retention, and restore testing
matter.

Failure modes:

- secret store unavailable during emergency
- credentials misconfigured
- backup cannot restore keys
- cloud or vendor dependency conflicts with self-hosting goals
- operator misunderstands which system can access raw keys

Usability during emergency:

Good if the deployment is mature and tested. Poor if the secret store is not
available or the operator cannot recover it.

Fit for personal/self-hosted deployment:

Optional. A cloud KMS should not be required by default. A local locked secret
store or hardware-backed key may fit self-hosted deployments better, but each
choice needs explicit deployment docs.

### 7. No Server Break-Glass Support

How it works:

The project does not support server-assisted key access. Recovery relies on
contact-wrapped keys, browser/client-side decryption, trusted contact apps,
offline tools, or out-of-band recovery material.

What the server can access:

- ciphertext
- metadata
- wrapped keys it cannot open
- no raw media keys or plaintext

Operator trust requirements:

Lower. Operators still control availability and metadata, but not media
plaintext.

Audit requirements:

Audit remains focused on token access, wrapped-key delivery, metadata changes,
and administrative actions.

Failure modes:

- phone destroyed before key wrapping completes
- contacts unavailable
- contact keys lost
- no path to decrypt in a worst-case emergency

Usability during emergency:

Good if contact setup works. Bad when all client-side recovery paths fail.

Fit for personal/self-hosted deployment:

Safe and simple, but may not meet the strongest availability goal. It remains a
valid deployment policy for users who prioritize confidentiality over server
assisted recovery.

## Trigger Policy

Break-glass access requires a trigger policy. Possible triggers include:

- explicit user panic or incident start
- missed checkins
- dead-man-switch timeout
- trusted contact request
- manual maintainer/operator action
- repeated failed uploads
- device offline threshold

These triggers must not be treated as equivalent. Some are weak signals, and
some are strong explicit signals. A future design should define which triggers
can only notify contacts, which can expose metadata, which can release wrapped
keys, and which can authorize server-side key access.

False positives matter. A missed checkin may happen because of battery drain,
network outage, travel, sleep, or app failure. An incorrect trigger could expose
keys or evidence before the user intended.

False negatives matter too. A device may be destroyed before sending a final
panic signal. A conservative trigger policy may delay or prevent emergency
access when evidence is urgently needed.

Any trigger design should define:

- trigger source
- minimum confidence threshold
- delay or cooldown
- cancellation path
- contact notification behavior
- policy escalation stages
- audit entry contents
- post-incident review process

## Access Controls

If server-assisted key access exists, it must have explicit controls.

Required controls:

- explicit enable/disable configuration
- per-incident policy
- trusted contact authorization rules
- operator authentication for private/admin actions
- local-only or private API boundary for administrative controls
- audit log for decisions and key access attempts
- rate limits and abuse controls
- notification events for user, contacts, or operators where appropriate
- revocation for tokens, contacts, and future key wrapping
- least-privilege separation between metadata viewing, key unwrap, and plaintext
  export
- clear separation between emergency viewer token and decryption authority

The emergency viewer token should authorize read access to the incident's
emergency view and encrypted evidence. It should not by itself grant decryption
authority unless a future design explicitly accepts a bearer-token-only
decryption mode with documented risks.

Break-glass controls should be separate from ordinary bundle download controls.
Downloading encrypted bundles and unwrapping media keys are different security
events.

## Audit And Logging

Audit logs should record security-relevant decisions without storing secrets.

Log:

- key unwrap attempts
- successful key access
- failed key access
- who or what triggered access
- incident ID
- timestamp
- policy decision
- actor or subsystem identity
- reason code
- outcome
- post-incident review status when available

Never log:

- raw media keys
- raw escrow keys
- contact private keys
- plaintext
- decrypted media bytes
- raw emergency tokens
- uploaded bytes
- request bodies
- Authorization headers
- sensitive user safety data beyond necessary audit metadata

Audit logs need their own retention, backup, and access policy. They may reveal
that an incident exists, when access was attempted, and who was involved.

## Deployment Assumptions

Self-hosted local server:

The simplest deployment may have one operator who is also the user. Break-glass
can be useful here, but it should still be explicit because malware or another
local user on the server could abuse raw key access.

Docker deployment:

Containerized deployments must account for persistent volumes, container
environment variables, host logs, backup jobs, and where escrow secrets live.
Raw keys should not be stored casually in environment variables or image layers.

WireGuard/private API:

Private administrative actions must stay behind localhost, LAN, WireGuard,
firewall, or an equivalent private boundary. Break-glass controls must not be
mounted on the public emergency viewer listener.

HTTPS emergency viewer:

The public emergency viewer may expose metadata and encrypted bundles through
HTTPS. Public viewer access must not become an administrative key-access path
unless separately designed and authorized.

External KMS/HSM future option:

Some deployments may choose a KMS, HSM, hardware-backed local key, or locked
secret store. This repository should not require cloud services by default.

Disk encryption:

Disk encryption can protect stored database, blobs, wrapped keys, audit logs,
and local secret stores when the server is offline. It does not protect against
a running compromised server.

Backup and restore:

Escrow material, wrapped keys, audit logs, and ciphertext blobs must be backed
up and restored consistently. A restored database without the matching secret
store may make emergency access impossible. A restored secret without audit logs
may make access unreviewable.

Retention and deletion:

Break-glass modes increase the need for retention and deletion policy.
Plaintext exports, temporary decrypted files, derived media, audit logs, wrapped
keys, and escrow keys all need explicit lifecycle rules before implementation.

## Threat Model Impacts

Backend compromise:

Default contact-wrapped keys limit a passive backend compromise. Break-glass
mode increases risk because a compromised backend may be able to request key
unwraps, alter policy decisions, or capture plaintext during server-side
decrypt operations.

Database compromise:

An attacker may obtain wrapped keys, policy metadata, trigger state, and audit
metadata. If server escrow wrapping is weak or escrow keys are stored nearby,
database compromise becomes more dangerous.

Blob storage compromise:

Ciphertext chunks remain protected without media keys. If break-glass produces
plaintext exports on disk, blob or backup compromise may expose those derived
files unless retention is strict.

Operator misuse:

Break-glass explicitly makes operators more powerful. Controls must assume that
mistakes, curiosity, coercion, or malicious action are possible.

Malicious trusted contact:

A trusted contact may request emergency access falsely, misuse decrypted
evidence, or share plaintext. Contact authorization and audit should separate
contact read access from server-assisted key access.

Stolen emergency token:

A stolen token should not grant key unwrap by itself. If a future policy allows
token-only decryption, that must be called out as a weaker mode.

Stolen key escrow material:

Escrow key compromise can expose all media keys protected by that escrow key.
Rotation, incident scoping, backup handling, and revocation expectations must be
designed before implementation.

False dead-man-switch trigger:

A false trigger may expose wrapped keys, raw keys, or plaintext early. Policies
need delays, cancellation, notifications, and review.

Destroyed phone:

This is the main availability case break-glass tries to address. It only works
if the phone uploaded ciphertext and the relevant wrapped or escrowed key
material before becoming unavailable.

Compromised server during browser decryption:

If browser decryption is served by the same backend, a compromised backend can
serve malicious JavaScript to capture keys. Break-glass does not solve this; it
may compound the problem if the same server can also unwrap keys.

## Recommended Direction

Default model:

- use contact-wrapped media keys
- keep backend storage ciphertext-only in normal operation
- support browser/client-side or app-based contact decryption where practical
- keep emergency viewer tokens separate from decryption authority

Optional future mode:

- allow server escrow or break-glass access only for dead-man-switch and
  emergency-access cases
- keep it disabled by default
- require explicit deployment configuration
- require per-incident policy
- require audit logging and post-incident review
- require operator access controls
- document the higher backend/operator trust model clearly

Not recommended early:

- server-side decrypt/transcode/playable export as an initial break-glass
  feature
- bearer-token-only decryption authority
- cloud KMS dependency as a default requirement

Break-glass access should be documented now, optional later, and implemented
only after the default contact-wrapped key path is accepted.

## Implementation Prerequisites

Before implementing break-glass access, complete or accept:

- key custody design
- browser/contact decryption design
- exact key hierarchy and wrapping scheme
- retention, backup, and deletion policy
- emergency-token expiry and revocation workflow for tokens, contacts, and wrapped keys
- `/v1` access-control story
- operator authentication story
- audit log design and retention policy
- trigger policy and false-positive/false-negative handling
- deployment hardening guidance
- secret-store choice for self-hosted deployments
- test plan for authorization, audit, key unwrap, and failure cases
- documentation updates for security model, threat model, encryption, API, and
  deployment guidance

## Open Questions

- Should break-glass be supported in the first production key custody release,
  or deferred until contact-wrapped decryption is proven?
- Should server escrow be per-incident, per-stream, per-user, or deployment
  wide?
- Should the server ever return raw media keys to a client, or only perform
  constrained operations?
- Is server-side decrypt/transcode acceptable for any deployment mode?
- What actor can approve break-glass access?
- Can trusted contacts request break-glass without operator involvement?
- How are false dead-man-switch triggers cancelled?
- What audit log fields are useful without exposing sensitive safety data?
- Where should escrow secrets live in a self-hosted non-cloud deployment?
- How should backup and restore verify that wrapped keys, blobs, database rows,
  and secret stores still match?
- How should break-glass interact with browser decryption and malicious-server
  risk?
- What should happen when a contact is revoked after wrapped or escrowed key
  material was already created?
