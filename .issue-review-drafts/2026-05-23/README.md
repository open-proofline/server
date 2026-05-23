# Issue Review

Reviewed open issues for `TheSilkky/safety-recorder` on 2026-05-23 against current `main`.

## Close as fixed

| Issue | Confidence | Evidence |
|---|---|---|
| [#9 Design Production Key Sharing And Emergency Access](issue-9.md) | high | Resolved by `c9d847ae2b26cfc22c6dbd728b491933466eca35`, with related follow-ups `8a65bffb4a9bc50f1f21c1177baef5b845c34c1a`, `4fe769740b72f26fb50d13f7dc1e57511477f2bd`, and `16a09f5b62e7f136371f49177bb5d66f1f8c737b`; see `docs/key-custody.md`, `docs/browser-decryption.md`, `docs/break-glass-key-access.md`, and `CHANGELOG.md`. |

## Keep open

| Issue | Confidence | Evidence |
|---|---|---|
| [#2 Add Default Emergency-Token Expiry Policy](issue-2.md) | high | `docs/security-model.md` and `docs/threat-model.md` still list missing default expiry; `server/internal/httpapi/emergency.go` still forwards omitted `expires_at` as nil. |
| [#3 Add Reverse Proxy And WireGuard Deployment Examples](issue-3.md) | high | `docs/deployment.md` has warnings and a localhost Docker example, but no concrete WireGuard or HTTPS reverse-proxy examples. |
| [#4 Add Rate Limiting Guidance And Proxy Examples](issue-4.md) | high | Current docs mention rate limiting as missing/needed, but do not define route groups or provide proxy snippets. |
| [#5 Update Emergency Viewer DOM During Polling](issue-5.md) | high | `server/internal/httpapi/web/static/scripts.js` still stores polled JSON on `window.__lastEmergencyData` without updating visible DOM. |
| [#6 Define Retention, Backup, And Secure Deletion Policy](issue-6.md) | high | Current docs still list retention/backup/deletion as missing; no lifecycle policy exists. |
| [#8 Document Branch Protection And Required Checks](issue-8.md) | medium | CI and development docs mention checks, and PR #11 added `go vet`, but no branch-protection or required-check policy is documented. |

## Needs update

| Issue | Confidence | Evidence |
|---|---|---|
| [#10 Plan iOS Local Recorder Prototype](issue-10.md) | medium | The prototype plan is still absent, but its dependency on #9 should be updated if #9 is closed as fixed. |

## Human review

| Issue | Confidence | Evidence |
|---|---|---|
| None | n/a | n/a |

## Sensitive

| Issue | Confidence | Evidence |
|---|---|---|
| None | n/a | n/a |

## Notes

- No GitHub issues were closed.
- No application code or documentation files were changed.
- Closed issue #7 was cross-checked: PR #11 merged as `14003fe8fa4f135355adbe92550928f3c3987161` and added `go vet` to CI.
