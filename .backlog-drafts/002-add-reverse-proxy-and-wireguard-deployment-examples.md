# Add Reverse Proxy And WireGuard Deployment Examples

## Priority

P2

## Type

deployment

## Labels

- `backlog`
- `deployment`
- `docs`

## Summary

The docs warn not to expose `/v1` publicly, but they do not yet provide concrete reverse-proxy or WireGuard deployment examples. Add practical examples that preserve the current private/public listener split.

## Context

`README.md`, `docs/deployment.md`, `docs/security-model.md`, and `docs/threat-model.md` all say the private `/v1` API must stay behind localhost, LAN, WireGuard, firewall rules, or a strict reverse proxy. `server/Dockerfile` binds container listeners to `0.0.0.0`, so host publishing and proxy configuration are important.

## Proposed change

Add deployment documentation with at least one localhost-only Docker example, one WireGuard/private-network example, and one HTTPS reverse-proxy example for the public emergency viewer. Include explicit notes that `/v1` remains private and unauthenticated.

## Acceptance criteria

- [ ] Docs show how to expose only the emergency viewer through HTTPS.
- [ ] Docs show how to keep `/v1` on localhost or a private network.
- [ ] Examples include token-path log redaction guidance for `/e/{token}`.
- [ ] Examples mention app/proxy timeout coordination for bundle downloads.
- [ ] Docs do not claim production readiness.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] commands reviewed for consistency with current environment variables
- [ ] no Go tests required unless code changes

## Out of scope

Do not add Docker Compose, Kubernetes, cloud-specific infrastructure, OAuth, JWT, user accounts, or a public admin dashboard.

## Notes

Related docs: `docs/deployment.md`, `docs/configuration.md`, `docs/threat-model.md`, `README.md`.
