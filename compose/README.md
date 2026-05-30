# Compose Smoke Tests

This directory contains local Docker Compose stacks for exercising Proofline
Server backend combinations before release work.

These stacks are for local smoke testing only. They use fixed local test
credentials, publish the private API on loopback by default, and do not make
Proofline production-ready public infrastructure.

## Variants

| Variant | File | Metadata | Blob storage | Coordination |
|---|---|---|---|---|
| `full` | `smoke-full.yml` | PostgreSQL | MinIO S3-compatible bucket | Valkey |
| `sqlite-local` | `smoke-sqlite-local.yml` | SQLite | Local filesystem | none |
| `postgresql-local` | `smoke-postgresql-local.yml` | PostgreSQL | Local filesystem | none |
| `sqlite-s3` | `smoke-sqlite-s3.yml` | SQLite | MinIO S3-compatible bucket | none |

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

Set `KEEP_COMPOSE=1` to leave containers and volumes running after the smoke
test for manual inspection.
