# Issue #10: Plan iOS Local Recorder Prototype

## Recommendation

needs-update

## Confidence

medium

## Summary

The prototype planning work is still valid, but the issue's dependency on #9 appears stale if #9 is accepted as fixed. The current repo still has no iOS local recorder prototype plan.

## Evidence reviewed

- Issue acceptance criteria:
  - Prototype scope is documented before iOS code is added.
  - Plan maps recorder chunks to current stream upload semantics.
  - Plan includes local encryption and key-storage assumptions or links to the key-sharing design.
  - Plan identifies failure modes such as network loss, app backgrounding, device lock, and interrupted uploads.
  - Plan lists backend API gaps without implementing them immediately.
- Relevant files:
  - `README.md:19` says the intended future client is an iOS app, but the repository is backend-only.
  - `README.md:40` says there is no iOS app.
  - `docs/architecture.md:5` says the repository does not contain an iOS app or recording implementation.
  - `docs/architecture.md:11` shows the planned iOS app as not implemented.
  - `docs/simulator.md:3` says the simulator exercises the current ingest flow a future recording client is expected to use.
  - `docs/simulator.md:30` documents stream upload semantics exercised by the simulator.
  - `docs/encryption.md:11` says future production client key storage, sharing, recovery, and emergency-contact access are designed separately in `docs/key-custody.md`.
  - `docs/key-custody.md:670` says iOS Keychain and contact-key planning should happen before implementing the iOS client.
- Relevant commits or PRs:
  - Commit `c9d847ae2b26cfc22c6dbd728b491933466eca35` added the key custody design that likely satisfies the issue's dependency on #9.
  - No commit found that adds an iOS local recorder prototype plan.

## Analysis

The issue should remain open for the actual iOS prototype plan. Current docs describe the future iOS app and the simulator reference flow, but they do not define prototype scope, background behavior, local staging, retry behavior, device-lock implications, or backend gaps for an iOS recorder.

The issue likely needs a maintenance update because it depends on #9, and #9 appears fixed by the new key custody docs.

## Suggested maintainer action

Update the issue to reference `docs/key-custody.md`, `docs/browser-decryption.md`, and `docs/break-glass-key-access.md` as existing design inputs, then keep it open for a focused iOS prototype plan.

## Draft comment

Reviewed against current `main`. The iOS prototype plan is still not present, so this should not close yet. However, the dependency on #9 looks stale if #9 is accepted as fixed by the current key custody docs. I recommend updating this issue to reference `docs/key-custody.md`, `docs/browser-decryption.md`, and `docs/break-glass-key-access.md` as inputs, then keeping it open for the actual local recorder prototype plan.

## Safe to close automatically?

no

## Notes

Confidence is medium because this recommendation depends on maintainer acceptance of the #9 close-fixed recommendation.
