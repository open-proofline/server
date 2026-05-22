# Backlog Issue Drafts

These are reviewed drafts for future work. They are not GitHub issues yet. Review each draft before creating a public issue, especially anything security-adjacent.

## P1

### Security Hardening

- [001-add-default-emergency-token-expiry-policy.md](001-add-default-emergency-token-expiry-policy.md)
- [003-add-rate-limiting-guidance-and-proxy-examples.md](003-add-rate-limiting-guidance-and-proxy-examples.md)
- [005-define-retention-backup-and-secure-deletion-policy.md](005-define-retention-backup-and-secure-deletion-policy.md)
- [008-design-production-key-sharing-and-emergency-access.md](008-design-production-key-sharing-and-emergency-access.md)

## P2

### Deployment

- [002-add-reverse-proxy-and-wireguard-deployment-examples.md](002-add-reverse-proxy-and-wireguard-deployment-examples.md)

### Correctness / UX

- [004-update-emergency-viewer-dom-during-polling.md](004-update-emergency-viewer-dom-during-polling.md)

### CI / Testing

- [006-add-go-vet-to-ci.md](006-add-go-vet-to-ci.md)

### iOS Planning

- [009-plan-ios-local-recorder-prototype.md](009-plan-ios-local-recorder-prototype.md)

## P3

### Release / CI Polish

- [007-document-branch-protection-and-required-checks.md](007-document-branch-protection-and-required-checks.md)

## Review Before Public Creation

- `001-add-default-emergency-token-expiry-policy.md`
- `003-add-rate-limiting-guidance-and-proxy-examples.md`
- `005-define-retention-backup-and-secure-deletion-policy.md`
- `008-design-production-key-sharing-and-emergency-access.md`

These are normal hardening/design issues, not private vulnerability reports, but they should still avoid exploit detail, raw tokens, private deployment information, and user safety data.

## Not Drafted Because Already Done

- Streamed chunks now require positive `chunk_index` values.
- `schema_migrations` tracking exists.
- Private/public HTTP server timeouts are configurable.
