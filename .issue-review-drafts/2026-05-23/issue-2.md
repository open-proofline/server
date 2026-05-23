# Issue #2: Add Default Emergency-Token Expiry Policy

## Recommendation

keep-open

## Confidence

high

## Summary

The issue is still valid. The current code and docs still allow emergency tokens with no default expiration when `expires_at` is omitted.

## Evidence reviewed

- Issue acceptance criteria:
  - A default emergency-token expiry is documented.
  - Token creation applies the default when `expires_at` is omitted.
  - Explicit `expires_at` behavior is preserved or intentionally updated.
  - Expired, revoked, and invalid tokens still return the same public error.
  - Tests cover default expiry, explicit expiry, expired tokens, and no raw-token storage.
- Relevant files:
  - `server/internal/httpapi/emergency.go:94` defines optional `expires_at`.
  - `server/internal/httpapi/emergency.go:102` passes `request.ExpiresAt` through unchanged.
  - `server/internal/incidents/repository.go:393` accepts `expiresAt *time.Time`.
  - `server/internal/incidents/repository.go:414` stores `utcTimePtr(expiresAt)`.
  - `server/internal/incidents/repository.go:468` rejects expired tokens only when `ExpiresAt` is non-nil.
  - `docs/security-model.md:70` lists no default emergency-token expiry policy as a known gap.
  - `docs/threat-model.md:51` says callers choose `expires_at`.
- Relevant commits or PRs:
  - No commit found that implements or documents a default token expiry policy.

## Analysis

Current token creation preserves nil expiration. The public error behavior for expired, revoked, and invalid tokens is already covered in `server/internal/httpapi/emergency.go`, and raw token storage is already tested, but the default-expiry acceptance criteria remain unmet.

## Suggested maintainer action

Keep the issue open. Decide the default lifetime and whether it is configurable, then implement and document it with focused tests.

## Draft comment

Reviewed against current `main`. This still appears valid: omitted `expires_at` is passed through as nil and stored without a default, while `docs/security-model.md` and `docs/threat-model.md` still list default emergency-token expiry as a known gap. Existing explicit expiry, revoked-token, invalid-token, and no-raw-token behavior appears related but does not satisfy the default-expiry acceptance criteria.

## Safe to close automatically?

no

## Notes

No sensitive details found in the issue body or review evidence.
