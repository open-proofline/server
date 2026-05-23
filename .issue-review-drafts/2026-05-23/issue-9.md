# Issue #9: Design Production Key Sharing And Emergency Access

## Recommendation

close-fixed

## Confidence

high

## Summary

The issue appears fixed by the current design docs. `docs/key-custody.md` now defines the production key custody direction, and follow-up browser-decryption and break-glass docs cover the related emergency-access paths.

## Evidence reviewed

- Issue acceptance criteria:
  - Design states which keys exist and where they are stored.
  - Design covers emergency-contact access without backend decryption.
  - Design covers key loss, device loss, revocation, and rotation tradeoffs.
  - Design identifies API changes needed before iOS implementation.
  - Design explicitly excludes backend key escrow unless separately accepted.
- Relevant files:
  - `docs/key-custody.md:19` defines the preferred hybrid key custody model.
  - `docs/key-custody.md:30` says future server decryption must be deliberate, documented, tested, and threat-modeled.
  - `docs/key-custody.md:375` recommends the hybrid model.
  - `docs/key-custody.md:381` through `docs/key-custody.md:388` describe default-mode keys and storage.
  - `docs/key-custody.md:390` through `docs/key-custody.md:403` reserve server escrow/server-side decryption for explicit break-glass modes.
  - `docs/key-custody.md:405` through `docs/key-custody.md:438` define the future key hierarchy and logging constraints.
  - `docs/key-custody.md:444` through `docs/key-custody.md:481` describe emergency contact access, lost contact keys, revocation, and rotation implications.
  - `docs/key-custody.md:652` through `docs/key-custody.md:684` describe phased implementation, including simulator prototype, browser/client decrypt prototype, iOS Keychain/contact-key planning, and optional break-glass.
  - `docs/browser-decryption.md` expands the browser/client-side emergency decryption design.
  - `docs/break-glass-key-access.md` documents explicit optional server-assisted break-glass/dead-man-switch access.
  - `CHANGELOG.md` Unreleased notes the new production key custody, browser decryption, and break-glass design docs.
- Relevant commits or PRs:
  - Commit `c9d847ae2b26cfc22c6dbd728b491933466eca35` added `docs/key-custody.md` and updated related docs.
  - Commit `8a65bffb4a9bc50f1f21c1177baef5b845c34c1a` added `docs/browser-decryption.md`.
  - Commit `4fe769740b72f26fb50d13f7dc1e57511477f2bd` added `docs/break-glass-key-access.md`.
  - Commit `16a09f5b62e7f136371f49177bb5d66f1f8c737b` aligned README and docs with the new design docs.
  - No merged PR found for these docs; they appear to be direct commits on `main`.

## Analysis

The core design requested by the issue is now present. The key custody doc states that keys must not live only on the iPhone, chooses a hybrid trusted-contact model, defines the key hierarchy, describes emergency-contact access without default backend decryption, covers loss/revocation/rotation tradeoffs, and identifies future phases before iOS work. The related browser-decryption and break-glass docs cover the two major sub-designs called out by the issue.

The wording in the original issue says backend key escrow should be excluded unless separately accepted. Current docs satisfy that spirit by making escrow/break-glass optional, disabled by default or separately configured, and explicitly documented as a higher-trust mode.

## Suggested maintainer action

Close as fixed if the maintainer agrees the design-doc scope satisfies the issue. Open narrower follow-up issues for implementation-specific key wrapping, API/schema changes, or iOS planning instead of keeping this broad design issue open.

## Draft comment

Reviewed against current `main`. This appears fixed by `c9d847ae2b26cfc22c6dbd728b491933466eca35`, with related follow-ups `8a65bffb4a9bc50f1f21c1177baef5b845c34c1a`, `4fe769740b72f26fb50d13f7dc1e57511477f2bd`, and `16a09f5b62e7f136371f49177bb5d66f1f8c737b`.

`docs/key-custody.md` now defines the hybrid key custody direction, key hierarchy, trusted-contact access flow, key loss/revocation/rotation tradeoffs, and implementation phases before iOS work. `docs/browser-decryption.md` and `docs/break-glass-key-access.md` cover the browser/client decrypt and explicit optional break-glass paths. I recommend closing this as fixed and tracking any remaining implementation work in narrower follow-up issues.

## Safe to close automatically?

yes

## Notes

No sensitive implementation details or secrets are included in the suggested public comment.
