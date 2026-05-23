# Issue #4: Add Rate Limiting Guidance And Proxy Examples

## Recommendation

keep-open

## Confidence

high

## Summary

The issue is still valid. Current docs identify rate limiting as missing and needed, but do not yet provide route-group guidance or proxy examples.

## Evidence reviewed

- Issue acceptance criteria:
  - Docs describe route groups that should have different limits.
  - Docs warn against logging raw `/e/{token}` paths while implementing limits.
  - Docs include example proxy snippets or pseudo-config that can be adapted safely.
  - Docs explain that app-level rate limiting is still absent.
  - Security-sensitive details are kept high-level and suitable for a public issue.
- Relevant files:
  - `docs/security-model.md:69` lists no built-in rate limiting or abuse throttling as a known gap.
  - `docs/deployment.md:67` lists rate limiting and abuse controls as needed for production-style exposure.
  - `docs/deployment.md:68` lists reverse-proxy log redaction for `/e/{token}` paths.
  - `docs/threat-model.md:50` lists no built-in TLS, rate limiting, abuse throttling, or IP allowlist.
  - `docs/threat-model.md:70` lists rate limiting for token guesses, uploads, and admin actions as a next security step.
- Relevant commits or PRs:
  - No commit found that adds rate-limiting route groups or proxy snippets.

## Analysis

The existing docs correctly call out the absence of app-level rate limiting and the need to avoid raw token logs. They do not yet explain separate limits for public token lookups, public downloads, static assets, private uploads, and private admin-style actions, and they do not include proxy pseudo-config.

## Suggested maintainer action

Keep the issue open. Add high-level, public-safe guidance to deployment/security docs without adding an app-level limiter in this issue.

## Draft comment

Reviewed against current `main`. This still appears valid. The docs mention that app-level rate limiting is absent and that production-style public exposure needs rate limiting and `/e/{token}` log redaction, but the requested route-group guidance and example proxy snippets are not present yet.

## Safe to close automatically?

no

## Notes

No sensitive details found in the issue body or review evidence.
