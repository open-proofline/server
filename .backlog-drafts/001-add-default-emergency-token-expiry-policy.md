# Add Default Emergency-Token Expiry Policy

## Priority

P1

## Type

security-hardening

## Labels

- `backlog`
- `security`
- `maintenance`

## Summary

Emergency tokens currently accept an optional `expires_at`; if omitted, tokens can remain valid until revoked. Define and implement a default expiry policy so token lifetime is deliberate.

## Context

`README.md`, `docs/security-model.md`, and `docs/threat-model.md` all document that emergency viewer links are bearer-token URLs and that there is no default emergency-token expiry policy. `server/internal/httpapi.createEmergencyToken` accepts an optional `expires_at`, and `server/internal/incidents.Repository.CreateEmergencyToken` stores `nil` expiry when none is provided.

## Proposed change

Decide the default token lifetime and where it should be configured. Implement the selected policy in the private token-creation path and document how callers can override or disable it, if disabling remains allowed.

## Acceptance criteria

- [ ] A default emergency-token expiry is documented.
- [ ] Token creation applies the default when `expires_at` is omitted.
- [ ] Existing explicit `expires_at` behavior is preserved or intentionally updated.
- [ ] Expired, revoked, and invalid tokens still return the same public error.
- [ ] Tests cover default expiry, explicit expiry, expired tokens, and no raw-token storage.

## Tests / validation

- [ ] `cd server && go test ./...`
- [ ] `cd server && go vet ./...`
- [ ] simulator smoke test, if token creation behavior changes in the simulator flow
- [ ] docs updated, if relevant

## Out of scope

Do not add user accounts, OAuth, JWT, public admin dashboards, SMS, push notifications, cloud services, or a new sharing system.

## Notes

Related files: `server/internal/httpapi/emergency.go`, `server/internal/incidents/repository.go`, `docs/security-model.md`, `docs/threat-model.md`, `docs/api.md`.
