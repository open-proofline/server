# Compose Smoke Tests

This directory contains local Docker Compose stacks for exercising Proofline
Server backend combinations before release work.

These stacks are for local smoke testing only. They use fixed local test
credentials, publish the private API on loopback by default, and do not make
Proofline production-ready public infrastructure.

The smoke script starts the server with a local bootstrap secret, creates a
temporary admin account over the private loopback API, and runs the simulator
with that account. The default bootstrap secret and password are placeholders
for local throwaway smoke volumes only. The script waits for
`GET /v1/health/ready` on the private loopback port before bootstrapping the
test account.

## Variants

| Variant | File | Metadata | Blob storage | Coordination |
|---|---|---|---|---|
| `full` | `compose-full.yml` | PostgreSQL | MinIO S3-compatible bucket | Valkey |
| `sqlite-local` | `compose-sqlite-local.yml` | SQLite | Local filesystem | none |
| `postgresql-local` | `compose-postgresql-local.yml` | PostgreSQL | Local filesystem | none |
| `sqlite-s3` | `compose-sqlite-s3.yml` | SQLite | MinIO S3-compatible bucket | none |

Run the default full-stack smoke test from the repository root:

```bash
compose/smoke-test.sh
```

Run a specific variant:

```bash
compose/smoke-test.sh sqlite-local
compose/smoke-test.sh postgresql-local
compose/smoke-test.sh sqlite-s3
```

Pass additional simulator arguments after `--`:

```bash
compose/smoke-test.sh full -- --chunks 5 --simulate-failure-every 2
```

The script uses `PROOFLINE_PRIVATE_PORT` and `PROOFLINE_PUBLIC_PORT` when set,
defaulting to `18080` and `18081`.

```bash
PROOFLINE_PRIVATE_PORT=28080 PROOFLINE_PUBLIC_PORT=28081 compose/smoke-test.sh full
```

The local smoke auth values can also be overridden:

```bash
PROOFLINE_SMOKE_BOOTSTRAP_SECRET='replace-with-local-compose-bootstrap-secret' \
PROOFLINE_SMOKE_USERNAME=admin \
PROOFLINE_SMOKE_PASSWORD='replace-with-a-long-local-password' \
compose/smoke-test.sh sqlite-local
```

Set `KEEP_COMPOSE=1` to leave containers and volumes running after the smoke
test for manual inspection.
