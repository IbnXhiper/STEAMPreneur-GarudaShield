#!/usr/bin/env bash
# deploy.sh — idempotent deploy runbook for GarudaShield v3.
#
# Usage:
#   ./deploy.sh                # single node (baseline, Redis off, nodeId=node-1)
#   ./deploy.sh node-1         # named cluster node + shared Redis (bus warm)
#   ./deploy.sh ny1 down       # any node id works; action is the 2nd argument
#   ./deploy.sh down           # stop everything
#   ./deploy.sh status         # show container health
#
# Node selection is configuration-driven: pass any node id you like
# (node-1, ny1, ny2, edge-1, ...) as $1. It only sets GS_NODE_ID so each
# instance has a distinct identity on the sync bus. The number of nodes that
# are ACTIVE comes from SYNC_NODE_COUNT in .env (today: 1, because the second
# VPS is down) or from "cluster.nodes" in the JSON config. Unavailable peers
# are reported as offline — they never block a deploy.
#
# Pre-flight (run once per host):
#   cp .env.example .env && nano .env        # fill GS_ADMIN_SECRET etc.
#   openssl rand -hex 32                      # paste into .env
#   # edit garudashield.prod.json: upstream.host, proxy.trustedProxies, GARUDA_HOST
#   # edit Caddyfile: localhost -> your domain, remove `local_certs`
set -euo pipefail

cd "$(dirname "$0")"

NODE="prod"
ACTION="up"
# Argument parsing: the first argument may be a node id OR a reserved action
# (up | down | status), so `./deploy.sh down` really stops the stack instead of
# deploying a node literally named "down". A node id takes the action as $2.
case "${1:-}" in
  ""|prod|lab)
    NODE="${1:-prod}"
    ACTION="${2:-up}"
    ;;
  up|down|status)
    ACTION="$1"
    NODE="${2:-prod}"
    ;;
  *)
    NODE="$1"
    ACTION="${2:-up}"
    ;;
esac

COMPOSE="docker-compose.prod.yml"
COMPOSE_OVERRIDE=""

# Node ids that need the shared Redis sync bus. "prod"/"lab" are the
# single-node baseline; anything else is treated as a named cluster node.
case "$NODE" in
  prod|lab)
    COMPOSE_OVERRIDE=""
    ;;
  *)
    export GS_NODE_ID="$NODE"
    export GS_REDIS_ENABLED=on
    COMPOSE_OVERRIDE="docker-compose.node.yml"
    ;;
esac

validate_env() {
  local missing=0 key value
  for key in GS_ADMIN_SECRET GS_SESSION_SECRET GS_LICENSE_SECRET POSTGRES_PASSWORD; do
    value="${!key:-}"
    if [ -z "$value" ] || [[ "$value" == *CHANGE_ME* ]] || [ "${#value}" -lt 32 ]; then
      echo "ERROR: $key must be set to a non-placeholder value of at least 32 characters" >&2
      missing=1
    fi
  done
  [ "$missing" -eq 0 ]
}

# Per-node overrides for a multi-node cluster.
# Each node runs Redis and joins the shared Redis Pub/Sub for blocklist sync.
# The active node count itself lives in the config / SYNC_NODE_COUNT, not here.

healthcheck() {
  local url="https://localhost/api/status"
  echo "Waiting for engine health (${url})..."
  for i in $(seq 1 30); do
    if curl -sk -o /dev/null "$url"; then
      echo "Engine healthy."
      return 0
    fi
    sleep 2
  done
  echo "Engine did NOT become healthy in 60s. Check: docker compose logs engine" >&2
  return 1
}

case "$ACTION" in
  down)
    docker compose -f "$COMPOSE" down
    ;;
  status)
    docker compose -f "$COMPOSE" ps
    curl -sk https://localhost/api/status | head -c 400; echo
    ;;
  up|"")
    if [ ! -f .env ]; then
      echo "ERROR: .env not found. Run: cp .env.example .env && edit it" >&2
      exit 1
    fi
    set -a
    # shellcheck disable=SC1091
    . ./.env
    set +a
    validate_env
    echo "==> Building images"
    docker compose -f "$COMPOSE" build
    echo "==> Starting (node=$NODE, redis=${GS_REDIS_ENABLED:-off}, syncNodes=${SYNC_NODE_COUNT:-1})"
    if [ -n "$COMPOSE_OVERRIDE" ] && [ -f "$COMPOSE_OVERRIDE" ]; then
      docker compose -f "$COMPOSE" -f "$COMPOSE_OVERRIDE" --env-file .env up -d
    else
      docker compose -f "$COMPOSE" --env-file .env up -d
    fi
    healthcheck
    echo "==> Deploy complete. Dashboard: https://localhost  Engine API: https://localhost/api/status"
    ;;
  *)
    echo "Unknown action: $ACTION (use: up | down | status)" >&2
    exit 1
    ;;
esac
