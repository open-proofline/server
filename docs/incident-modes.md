# Incident Capture Modes

Proofline is intended to be broader than an emergency-only recorder. The long-term product direction is private, encrypted incident capture for moments where a user wants a durable record, with emergency escalation available only when the user chooses it or a configured safety check is missed.

This is a planning document. It does not add mobile clients, account management, public `/v1` authentication, push notifications, emergency-services integration, incident-mode schema, key custody, browser decryption, or new backend routes.

## Product Framing

Proofline should preserve encrypted evidence and context while keeping escalation separate from capture.

Core principles:

- capture can be emergency or non-emergency
- upload should preserve already-captured evidence if the device is lost, damaged, powered off, or taken
- emergency-services contact should remain a user or trusted-contact action, not an automatic backend action
- sharing, export, publication, and legal submission should be deliberate user-controlled steps
- recording and sharing laws vary by jurisdiction, so future clients should include clear user-facing guidance without giving legal advice

## Planned Incident Types

Future clients may expose incident types such as:

| Type | Purpose | Default escalation |
|---|---|---|
| Emergency incident | Active safety risk where the user wants recording, upload, and urgent trusted-contact access. | Trusted contacts alerted immediately or after a short configured delay. |
| Interaction record | Non-emergency record of an important interaction, such as with police, security, landlords, employers, service providers, or other authorities. | No automatic escalation by default. |
| Safety check | Timed check-in flow for walking home, meeting someone, travel, fieldwork, or other elevated-risk situations. | Trusted contacts alerted if the user misses the check-in. |
| Evidence note | Quick photo, audio, location, or note bundle for damage, harassment, threats, or disputes. | No automatic escalation by default. |

Avoid product labels such as `police mode`. Use neutral language like `Interaction record` and optional user-selected tags.

## Interaction Records

Interaction records should help users preserve what happened during important encounters without treating every recording as an emergency.

Future client behavior should support:

- starting an interaction record quickly
- recording audio, video, location, notes, and important moments where platform permissions allow
- uploading encrypted chunks continuously when network access exists
- continuing local encrypted staging when upload is unavailable
- adding optional tags such as `police`, `security`, `workplace`, `landlord`, `medical`, or `other`
- adding post-incident notes, reference numbers, involved agency names, badge or vehicle numbers, and user-visible timeline markers
- keeping the incident private unless the user explicitly grants access or exports a bundle

The app should not imply that it reports alleged criminal activity, contacts law enforcement, guarantees admissibility, or provides legal advice.

## Emergency And Dead-Man Escalation

Emergency escalation should be a policy attached to an incident, not a property of all recording.

Possible escalation policies:

| Policy | Behavior |
|---|---|
| `none` | Keep the incident private unless the user shares or exports it. |
| `trusted_contacts_on_start` | Notify selected trusted contacts when the incident starts. |
| `trusted_contacts_on_missed_checkin` | Notify selected trusted contacts only if the user misses a configured check-in. |
| `urgent_trusted_contact_alert` | Notify selected trusted contacts urgently and provide emergency review guidance. |

Dead-man switch handling should rely on trusted contacts to interpret the context and decide whether to contact emergency services. Proofline should not claim that help is on the way or that emergency services have been notified unless a future jurisdiction-specific integration explicitly implements and documents that behavior.

Suggested trusted-contact guidance:

```text
A Proofline safety check was missed.
Review the incident, try to contact the user, and call emergency services if you believe there is immediate danger.
```

## Future Data Model Direction

The current backend only has generic incidents, streams, chunks, checkins, and incident tokens. Future protocol work may add first-class fields such as:

```text
incident_type:
  emergency
  interaction
  safety_check
  evidence_note

escalation_policy:
  none
  trusted_contacts_on_start
  trusted_contacts_on_missed_checkin
  urgent_trusted_contact_alert

capture_profile:
  audio_video_location
  audio_location
  location_checkin
  note_only

sharing_state:
  private
  trusted_contact_access
  legal_export_created
```

Do not add these fields to the current backend incident schema until the protocol, access-control model, mobile client behavior, and migration path are explicitly designed.

## Access And Sharing Direction

Future account-enabled clients should distinguish:

- account-owner access to their own incident data
- trusted-contact access granted by the account owner or emergency policy
- emergency access links or grants for specific incidents
- administrative/operator access, which should not casually expose user safety data
- legal/export workflows controlled by the account owner

The current token-scoped incident viewer is a temporary read-only access model. A future web client may replace it after account management, authorization, key custody, and trusted-contact access are designed.

## Current Implementation Status

Implemented today:

- generic incidents
- media streams
- encrypted chunk upload and immutable storage
- checkins
- incident tokens
- read-only incident viewer
- encrypted ZIP evidence bundles
- simulator upload and decrypt-verification flows

Not implemented today:

- first-class incident types
- account management
- public `/v1` authentication
- trusted-contact accounts
- mobile clients
- non-emergency interaction UX
- dead-man switch notifications
- push/SMS/Messenger integrations
- production key custody
- browser/client-side decryption
- legal export workflow

## Documentation And Review Rules

When future work touches incident modes, update the relevant source-of-truth docs together:

- [README](../README.md)
- [Architecture](architecture.md)
- [API](api.md)
- [iOS local recorder prototype](ios-local-recorder-prototype.md)
- [Security model](security-model.md)
- [Threat model](threat-model.md)
- [Key custody](key-custody.md), if sharing or decryption behavior changes

New ideas discovered while documenting incident modes should become backlog items unless they are required for the scoped task.