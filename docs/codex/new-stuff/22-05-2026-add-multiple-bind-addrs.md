# Codex Prompt: Support Multiple Bind Addresses

Update the Go backend so the private API server and public emergency viewer can each listen on multiple bind addresses.

Do not change application behaviour or routes.

## Goal

The app currently has separate private and public servers.

Update configuration so each server can listen on one or more bind addresses.

Example:

```env
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081
```

This allows deployments where the private API is reachable on localhost and/or WireGuard, while the public emergency viewer can be exposed separately.

## Configuration

Add support for:

- `SAFE_PRIVATE_BIND_ADDRS`
- `SAFE_PUBLIC_BIND_ADDRS`

Each variable should be a comma-separated list of `host:port` bind addresses.

Keep backwards compatibility with the existing singular variables:

- `SAFE_PRIVATE_BIND_ADDR`
- `SAFE_PUBLIC_BIND_ADDR`

## Precedence

Private API:

1. If `SAFE_PRIVATE_BIND_ADDRS` is set, use it.
2. Else if `SAFE_PRIVATE_BIND_ADDR` is set, use it.
3. Else default to `127.0.0.1:8080`.

Public emergency viewer:

1. If `SAFE_PUBLIC_BIND_ADDRS` is set, use it.
2. Else if `SAFE_PUBLIC_BIND_ADDR` is set, use it.
3. Else default to `127.0.0.1:8081`.

## Parsing rules

- Split comma-separated lists.
- Trim whitespace.
- Reject empty entries.
- Return a clear config error if no valid addresses remain.
- Do not silently bind to `0.0.0.0` unless explicitly configured.
- Keep addresses as strings and let `net/http` / `net.Listen` validate them.

Examples:

```env
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,10.66.0.1:8080
SAFE_PUBLIC_BIND_ADDRS=127.0.0.1:8081,192.168.1.20:8081
```

Invalid examples:

```env
SAFE_PRIVATE_BIND_ADDRS=
SAFE_PUBLIC_BIND_ADDRS=,
SAFE_PRIVATE_BIND_ADDRS=127.0.0.1:8080,,10.66.0.1:8080
```

Handle invalid entries consistently and document the behaviour.

## Server lifecycle

Update server startup so:

- each private bind address starts a private API `http.Server`
- each public bind address starts a public viewer `http.Server`
- all private servers share the private API mux
- all public servers share the emergency viewer mux
- startup logs list every bind address
- graceful shutdown stops all servers on `SIGINT` / `SIGTERM`
- if any listener fails at startup, shutdown all already-started servers and return an error

## Security requirements

- Do not mount private API routes on public viewer servers.
- Do not mount public viewer routes on private API servers unless already intentionally designed.
- Do not log raw emergency tokens.
- Do not change token validation, expiry, or revocation behaviour.
- Do not add new auth.
- Do not add React.
- Do not add Node.
- Do not add Docker Compose.
- Do not add Kubernetes.
- Do not add unrelated features.

## Docker caveat

Document that inside Docker containers, binding directly to host IP addresses may not work unless using host networking.

Container defaults should usually bind inside the container:

```env
SAFE_PRIVATE_BIND_ADDRS=0.0.0.0:8080
SAFE_PUBLIC_BIND_ADDRS=0.0.0.0:8081
```

Then restrict exposure using Docker port publishing, firewall rules, WireGuard, or reverse proxy configuration.

Example safe local Docker publish:

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:8081:8081 \
  ghcr.io/OWNER/REPO:latest
```

## Tests

Add tests for config parsing:

- default private/public bind addresses
- singular env vars
- plural comma-separated env vars
- plural vars taking precedence over singular vars
- whitespace trimming
- empty entries rejected or ignored consistently
- fully empty lists returning config error
- invalid comma-separated lists returning clear errors

If there are server construction tests, add coverage that multiple private/public bind addresses create the expected number of server configs.

## Documentation

Update `server/README.md` to document:

- `SAFE_PRIVATE_BIND_ADDRS`
- `SAFE_PUBLIC_BIND_ADDRS`
- singular vars are still supported
- plural vars take precedence over singular vars
- Docker caveat:
  - inside containers, bind to `0.0.0.0`
  - restrict exposure with Docker port mappings, firewall rules, WireGuard, or reverse proxy
- warning not to expose the private API publicly

Update any relevant code map or architecture documentation if present.

## Validation

After implementation:

```bash
gofmt -w .
go test ./...
```

## Summary after implementation

Summarize:

- files changed
- environment variables added
- backwards compatibility behaviour
- server lifecycle changes
- tests added
- any documentation updates
