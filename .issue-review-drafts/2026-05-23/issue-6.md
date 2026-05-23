# Issue #6: Define Retention, Backup, And Secure Deletion Policy

## Recommendation

keep-open

## Confidence

high

## Summary

The issue is still valid. Retention, backup, secure deletion, and disk encryption remain documented as missing operational policy.

## Evidence reviewed

- Issue acceptance criteria:
  - Docs define retention choices for incidents, chunks, streams, checkins, tokens, and bundles.
  - Docs define backup and restore expectations for SQLite and blob storage.
  - Docs explain secure deletion limitations and recommended disk encryption posture.
  - Docs identify future API or repository changes needed to implement deletion.
  - Docs avoid promising unrecoverable deletion unless the deployment model can support it.
- Relevant files:
  - `docs/security-model.md:72` lists no retention, backup, secure deletion, or disk encryption policy.
  - `docs/threat-model.md:52` lists no retention, backup, secure deletion, or disk encryption policy.
  - `docs/threat-model.md:72` lists defining retention, backup, and deletion policy as a next security step.
  - `docs/deployment.md:70` lists retention, backup, and deletion policy as still needed for production-style exposure.
  - `server/internal/storage/storage.go:136` has a low-level blob `Remove` helper, but no lifecycle policy.
  - `docs/break-glass-key-access.md:512` notes that break-glass modes increase lifecycle policy needs, but does not define the policy.
- Relevant commits or PRs:
  - No commit found that adds a retention, backup, restore, or secure deletion policy.

## Analysis

Current docs repeatedly mark the topic as a missing policy, and the storage helper is not a product-level lifecycle design. Break-glass docs add more reasons the policy matters, but they do not satisfy this issue.

## Suggested maintainer action

Keep the issue open. Add an operational design document before implementing deletion APIs or background jobs.

## Draft comment

Reviewed against current `main`. This still appears valid. Current docs continue to list retention, backup, secure deletion, and disk encryption policy as missing; the low-level storage `Remove` helper is not an incident/data lifecycle policy.

## Safe to close automatically?

no

## Notes

No sensitive details found in the issue body or review evidence.
