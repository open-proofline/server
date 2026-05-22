# Plan iOS Local Recorder Prototype

## Priority

P2

## Type

feature

## Labels

- `backlog`
- `ios`
- `docs`

## Summary

The repository is backend-only, but the intended future client is an iOS recorder. Create a planning issue for a small iOS local recorder prototype before implementation starts.

## Context

`README.md`, `docs/architecture.md`, `docs/simulator.md`, and `docs/encryption.md` describe a planned future iOS app that records audio/video in chunks, encrypts locally, and uploads through the private API. The simulator currently exercises the expected backend flow.

## Proposed change

Draft a prototype plan that defines the minimum local recorder behavior, chunking cadence, background/foreground constraints, local encrypted staging, upload retry behavior, and how it maps to current media stream APIs.

## Acceptance criteria

- [ ] Prototype scope is documented before iOS code is added.
- [ ] Plan maps recorder chunks to current stream upload semantics.
- [ ] Plan includes local encryption and key-storage assumptions or links to the key-sharing design.
- [ ] Plan identifies failure modes such as network loss, app backgrounding, device lock, and interrupted uploads.
- [ ] Plan lists backend API gaps, if any, without implementing them immediately.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] simulator smoke test remains the backend reference flow
- [ ] no Go tests required unless backend code changes

## Out of scope

Do not add iOS code, Swift packages, push notifications, SMS, Messenger, cloud services, user accounts, OAuth, JWT, or backend public authentication in this issue.

## Notes

Related docs: `README.md`, `docs/architecture.md`, `docs/simulator.md`, `docs/encryption.md`.
