# Design Production Key Sharing And Emergency Access

## Priority

P1

## Type

documentation

## Labels

- `backlog`
- `security`
- `docs`
- `ios`

## Summary

Before an iOS client or real emergency-contact workflow is implemented, define how production client keys are generated, stored, shared, recovered, and used by trusted contacts.

## Context

`docs/encryption.md` documents a simulator/test AES-256-GCM envelope and explicitly says production client key storage, Keychain integration, emergency-contact key access, key sharing, browser/client-side decryption, and playable export are future work. `README.md` and `docs/threat-model.md` repeat that the backend does not store keys and bundles remain encrypted.

## Proposed change

Write a design document covering client key generation, Keychain storage, per-incident or per-stream key strategy, emergency-contact access, revocation implications, recovery/loss scenarios, and what the backend must never learn.

## Acceptance criteria

- [ ] Design states which keys exist and where they are stored.
- [ ] Design covers emergency-contact access without backend decryption.
- [ ] Design covers key loss, device loss, revocation, and rotation tradeoffs.
- [ ] Design identifies API changes needed before iOS implementation.
- [ ] Design explicitly excludes backend key escrow unless separately accepted.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] threat model updated, if relevant
- [ ] no Go tests required unless implementation work is added later

## Out of scope

Do not implement iOS code, browser decryption, backend decryption, key escrow, user accounts, OAuth, JWT, SMS, Messenger, push notifications, or playable media export in this issue.

## Notes

Related docs: `docs/encryption.md`, `docs/threat-model.md`, `README.md`.
