# Codex Security Review Prompt

Review the backend as if it may later be exposed through a public emergency viewer.

Do not add new features.

Focus on:

- public route surface
- write/admin route exposure
- emergency token generation
- token hashing
- token expiry and revocation
- raw token logging
- path traversal
- file overwrite behaviour
- upload size limits
- panic/log leakage
- template escaping
- SQLite constraints
- whether v0.2.0 warnings are still accurate

Return a concise review with severity levels.

If fixes are needed, make only minimal fixes and run:

```bash
go test ./...
