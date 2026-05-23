# Issue #3: Add Reverse Proxy And WireGuard Deployment Examples

## Recommendation

keep-open

## Confidence

high

## Summary

The issue is still valid. Deployment docs warn about the private/public listener split and include localhost Docker publishing, but they do not yet provide concrete WireGuard or HTTPS reverse-proxy examples.

## Evidence reviewed

- Issue acceptance criteria:
  - Docs show how to expose only the emergency viewer through HTTPS.
  - Docs show how to keep `/v1` on localhost or a private network.
  - Examples include token-path log redaction guidance for `/e/{token}`.
  - Examples mention app/proxy timeout coordination for bundle downloads.
  - Docs do not claim production readiness.
- Relevant files:
  - `docs/deployment.md:3` states the project is not production-ready public infrastructure.
  - `docs/deployment.md:5` warns not to expose `/v1` publicly.
  - `docs/deployment.md:32` includes a localhost-only Docker port publishing example.
  - `docs/deployment.md:52` mentions firewall, WireGuard, or reverse proxy restriction at a high level.
  - `docs/deployment.md:58` mentions reverse-proxy timeout coordination.
  - `docs/deployment.md:62` says public exposure should expose only the emergency viewer unless `/v1` has separate authentication.
  - `docs/deployment.md:68` mentions reverse-proxy log redaction for `/e/{token}` paths.
- Relevant commits or PRs:
  - No commit found that adds concrete reverse-proxy or WireGuard deployment examples.

## Analysis

The docs already include important warnings and some supporting deployment notes, so the issue is partly covered by existing text. The central request remains open because there are no concrete adaptable examples for WireGuard/private-network deployment or HTTPS reverse-proxy exposure of only the public emergency viewer.

## Suggested maintainer action

Keep the issue open. Add a deployment examples section or standalone doc with concrete localhost-only, WireGuard/private-network, and public HTTPS reverse-proxy examples that preserve private/public route separation.

## Draft comment

Reviewed against current `main`. This still appears valid. `docs/deployment.md` has the right warnings, a localhost-only Docker example, token-path log redaction notes, and timeout coordination notes, but it does not yet provide the concrete WireGuard/private-network or HTTPS reverse-proxy examples requested by the acceptance criteria.

## Safe to close automatically?

no

## Notes

No sensitive details found in the issue body or review evidence.
