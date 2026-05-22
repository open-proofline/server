# Add Rate Limiting Guidance And Proxy Examples

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

The backend has no built-in rate limiting or abuse throttling. Add guidance and examples for rate limiting token lookups, emergency downloads, uploads, and private admin actions at the deployment edge.

## Context

`docs/security-model.md` and `docs/threat-model.md` list rate limiting as a known gap and next security step. Public emergency routes are bearer-token scoped, and private `/v1` routes are intended to be protected by deployment boundaries rather than app-level public authentication.

## Proposed change

Document reverse-proxy rate-limiting patterns for public emergency viewer routes and private API routes. Include separate treatment for `/e/{token}` lookup/download paths, static assets, private upload endpoints, and private admin-style routes.

## Acceptance criteria

- [ ] Docs describe route groups that should have different limits.
- [ ] Docs warn against logging raw `/e/{token}` paths while implementing limits.
- [ ] Docs include example proxy snippets or pseudo-config that can be adapted safely.
- [ ] Docs explain that app-level rate limiting is still absent.
- [ ] Security-sensitive details are kept high-level and suitable for a public issue.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] no Go tests required unless code changes

## Out of scope

Do not add a new app-level rate limiter, user accounts, OAuth, JWT, CAPTCHA, SMS, push notifications, cloud services, or public admin dashboards in this issue.

## Notes

Related docs: `docs/deployment.md`, `docs/security-model.md`, `docs/threat-model.md`.
