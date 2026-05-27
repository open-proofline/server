# Security Hardening

Use this only for non-sensitive hardening work. Do not include exploit details, raw incident tokens, secrets, private deployment information, request bodies, uploaded bytes, or user safety data.

If this is a vulnerability report, follow `SECURITY.md` instead of opening a public issue.

## Summary

One or two sentences.

## Hardening area

- [ ] private `/v1` route boundary
- [ ] public incident viewer
- [ ] incident tokens
- [ ] upload/storage integrity
- [ ] ZIP bundle safety
- [ ] logging
- [ ] headers / browser security
- [ ] deployment documentation
- [ ] other:

## Current risk or limitation

Describe the limitation without sensitive exploit detail.

## Proposed control

What should change.

## Acceptance criteria

- [ ] ...
- [ ] ...
- [ ] ...

## Tests / docs

- [ ] `go test ./...`
- [ ] `go vet ./...`, if practical
- [ ] simulator smoke test, if relevant
- [ ] docs updated, if relevant

## Out of scope

What this hardening task should not include.
