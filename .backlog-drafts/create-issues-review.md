# Backlog Issue Creation Review

This review covers the GitHub issue creation script at `scripts/create-backlog-issues.sh`.

## Included Drafts

| Draft | Title | Labels |
| --- | --- | --- |
| `.backlog-drafts/001-add-default-emergency-token-expiry-policy.md` | Add Default Emergency-Token Expiry Policy | `backlog`, `security`, `maintenance` |
| `.backlog-drafts/002-add-reverse-proxy-and-wireguard-deployment-examples.md` | Add Reverse Proxy And WireGuard Deployment Examples | `backlog`, `deployment`, `docs` |
| `.backlog-drafts/003-add-rate-limiting-guidance-and-proxy-examples.md` | Add Rate Limiting Guidance And Proxy Examples | `backlog`, `security`, `deployment`, `docs` |
| `.backlog-drafts/004-update-emergency-viewer-dom-during-polling.md` | Update Emergency Viewer DOM During Polling | `backlog`, `bug`, `testing` |
| `.backlog-drafts/005-define-retention-backup-and-secure-deletion-policy.md` | Define Retention, Backup, And Secure Deletion Policy | `backlog`, `security`, `deployment`, `docs` |
| `.backlog-drafts/006-add-go-vet-to-ci.md` | Add go vet To CI | `backlog`, `ci`, `testing`, `maintenance` |
| `.backlog-drafts/007-document-branch-protection-and-required-checks.md` | Document Branch Protection And Required Checks | `backlog`, `docs`, `ci` |
| `.backlog-drafts/008-design-production-key-sharing-and-emergency-access.md` | Design Production Key Sharing And Emergency Access | `backlog`, `security`, `docs`, `ios` |
| `.backlog-drafts/009-plan-ios-local-recorder-prototype.md` | Plan iOS Local Recorder Prototype | `backlog`, `ios`, `docs` |

## Excluded Drafts

None. The reviewed drafts do not appear to be marked sensitive, private, or not-public.

## Labels Used

`backlog`, `bug`, `ci`, `deployment`, `docs`, `ios`, `maintenance`, `security`, `testing`

These labels must already exist in `TheSilkky/safety-recorder` or `gh issue create` may fail.

## Run Command

After review, run:

```bash
bash scripts/create-backlog-issues.sh
```

The script checks for `gh`, checks authentication, and asks for confirmation before creating issues.

## Warnings

- Running the script more than once may create duplicate issues.
- Review security-adjacent drafts before creating public issues.
- The script does not create private vulnerability reports or execute any repository changes beyond GitHub issue creation.
