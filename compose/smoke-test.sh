#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: compose/smoke-test.sh [full|sqlite-local|postgresql-local|sqlite-s3] [-- <simclient args>]

Runs a Docker Compose smoke stack, waits for the public listener, then runs the
Go simulator against the containerized server.

Environment:
  PROOFLINE_PRIVATE_PORT  Host port for the private API. Default: 18080
  PROOFLINE_PUBLIC_PORT   Host port for the public viewer. Default: 18081
  COMPOSE_PROJECT_NAME    Compose project name. Default: proofline-smoke-<variant>
  KEEP_COMPOSE=1          Leave containers and volumes running after the test.
USAGE
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

variant="full"
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi
if [[ $# -gt 0 && "$1" != "--" ]]; then
  variant="$1"
  shift
fi
if [[ $# -gt 0 && "$1" == "--" ]]; then
  shift
fi

case "$variant" in
  full)
    compose_file="$script_dir/smoke-full.yml"
    ;;
  sqlite-local)
    compose_file="$script_dir/smoke-sqlite-local.yml"
    ;;
  postgresql-local)
    compose_file="$script_dir/smoke-postgresql-local.yml"
    ;;
  sqlite-s3)
    compose_file="$script_dir/smoke-sqlite-s3.yml"
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

if docker compose version >/dev/null 2>&1; then
  compose=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose=(docker-compose)
else
  echo "docker compose or docker-compose is required" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required to wait for the containerized public listener" >&2
  exit 1
fi

export PROOFLINE_PRIVATE_PORT="${PROOFLINE_PRIVATE_PORT:-18080}"
export PROOFLINE_PUBLIC_PORT="${PROOFLINE_PUBLIC_PORT:-18081}"
project="${COMPOSE_PROJECT_NAME:-proofline-smoke-${variant}}"
sim_args=("$@")

cleanup() {
  status=$?
  if [[ "${KEEP_COMPOSE:-0}" == "1" ]]; then
    echo "Leaving compose stack running: project=$project file=$compose_file"
    exit "$status"
  fi
  "${compose[@]}" -p "$project" -f "$compose_file" down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

wait_for_public_listener() {
  local url="http://127.0.0.1:${PROOFLINE_PUBLIC_PORT}/static/styles.css"
  for _ in $(seq 1 60); do
    if curl --fail --silent --show-error --output /dev/null "$url"; then
      return 0
    fi
    sleep 1
  done
  return 1
}

cd "$repo_root"

"${compose[@]}" -p "$project" -f "$compose_file" down -v --remove-orphans >/dev/null 2>&1 || true
if ! "${compose[@]}" -p "$project" -f "$compose_file" up --build -d; then
  "${compose[@]}" -p "$project" -f "$compose_file" ps || true
  "${compose[@]}" -p "$project" -f "$compose_file" logs --no-color || true
  exit 1
fi

if ! wait_for_public_listener; then
  "${compose[@]}" -p "$project" -f "$compose_file" ps
  "${compose[@]}" -p "$project" -f "$compose_file" logs --no-color
  echo "server did not become ready on public port ${PROOFLINE_PUBLIC_PORT}" >&2
  exit 1
fi

go run ./cmd/simclient \
  --api "http://127.0.0.1:${PROOFLINE_PRIVATE_PORT}" \
  --viewer "http://127.0.0.1:${PROOFLINE_PUBLIC_PORT}" \
  --chunks 3 \
  --interval 0s \
  --download-bundle \
  "${sim_args[@]}"
