# Codex Prompt: Documentation Update

Update documentation to match the current code.

Do not change code unless documentation reveals a clear inconsistency or broken example.

## Source of truth

Treat the current code and top-level `README.md` as the source of truth.

The project currently includes:

- private `/v1` API listener group
- public emergency viewer listener group
- plural bind address variables:
  - `SAFE_PRIVATE_BIND_ADDRS`
  - `SAFE_PUBLIC_BIND_ADDRS`
- uploaded encrypted chunks
- media streams
- completed/failed stream states
- completed encrypted stream/incident ZIP evidence bundles
- emergency viewer tokens
- simulator CLI
- Docker/GHCR/CI support

## Update only relevant docs

Potential docs:

- `README.md`
- `docs/api.md`
- `docs/code-map.md`
- `docs/threat-model.md`
- `docs/security-model.md`
- `CHANGELOG.md`
- `AGENTS.md`
- `codex/README.md`

## Check

Verify documentation for:

- project scope
- current version number, if stated
- endpoint list
- request/response examples
- environment variables
- singular/plural bind address behaviour
- Docker bind caveat
- private/public listener separation
- data directory layout
- media stream lifecycle
- completed evidence bundle downloads
- emergency viewer download buttons
- simulator commands
- security warnings
- known limitations
- test/run/build commands
- CI/GHCR notes
- next steps

## Constraints

- Do not overpromise production readiness.
- Do not imply the backend decrypts chunks.
- Do not imply evidence bundles are playable media.
- Do not imply `/v1` is safe for public exposure.
- Do not add React, Node, npm, OAuth, JWT, user accounts, SMS, Messenger, push notifications, Docker Compose, Kubernetes, or cloud integrations.
- Keep wording clear and concise.

## Validation

After documentation changes, run tests if any code changed:

```bash
gofmt -w .
go test ./...
```

If only docs changed, state that tests were not required.
