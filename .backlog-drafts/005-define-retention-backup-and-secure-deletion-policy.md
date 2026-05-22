# Define Retention, Backup, And Secure Deletion Policy

## Priority

P1

## Type

security-hardening

## Labels

- `backlog`
- `security`
- `deployment`
- `docs`

## Summary

The project stores encrypted blobs and SQLite metadata locally, but there is no retention, backup, restore, or secure deletion policy. Define the operational policy before real-world use.

## Context

`README.md`, `docs/deployment.md`, `docs/security-model.md`, and `docs/threat-model.md` all list retention, backup, deletion, or disk encryption as missing production hardening. `server/internal/storage` can remove individual blob paths, but there is no lifecycle policy for incidents, blobs, database rows, backups, or restores.

## Proposed change

Create an operational design document covering what data is retained, how backups are made and tested, how incidents are deleted or expired, what secure deletion can and cannot promise on modern filesystems, and how disk encryption fits into the deployment model.

## Acceptance criteria

- [ ] Docs define retention choices for incidents, chunks, streams, checkins, tokens, and bundles.
- [ ] Docs define backup and restore expectations for SQLite and blob storage.
- [ ] Docs explain secure deletion limitations and recommended disk encryption posture.
- [ ] Docs identify any future API or repository changes needed to implement deletion.
- [ ] Docs avoid promising unrecoverable deletion unless the deployment model can support it.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] no Go tests required unless implementation work is added later

## Out of scope

Do not implement deletion APIs, background jobs, cloud backups, public admin dashboards, user accounts, or key escrow in this issue.

## Notes

Related docs: `docs/deployment.md`, `docs/security-model.md`, `docs/threat-model.md`; related code: `server/internal/storage/storage.go`.
