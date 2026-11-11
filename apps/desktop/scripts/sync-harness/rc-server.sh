#!/usr/bin/env bash
# Starts croffle-server 0.1.0-rc.1 (the verification target, V-17) and creates an account.
# Usage: rc-server.sh <username> [port]   — the password is read from stdin, never an argument.
# Then, for a second device: docker exec croffle-rc croffle-server token issue <username> --device-name device-b
set -euo pipefail
user="${1:?username}"
port="${2:-8787}"
image="ghcr.io/team-croffle/croffle-server:0.1.0-rc.1"
docker rm -f croffle-rc >/dev/null 2>&1 || true
docker run -d --name croffle-rc -p "127.0.0.1:${port}:8787" \
  -e CROFFLE_DB_PATH=/tmp/server.db "${image}" >/dev/null
until curl -sf "http://127.0.0.1:${port}/health" >/dev/null; do sleep 1; done
docker exec -i croffle-rc croffle-server user create "${user}" --password-stdin
curl -s "http://127.0.0.1:${port}/health"; echo
