#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8317}"
MGMT_KEY="${MGMT_KEY:?MGMT_KEY is required}"
IN_FILE="${IN_FILE:-/data/usage.json}"

if [ ! -f "${IN_FILE}" ]; then
  echo "[usage-import] no snapshot found, skip"
  exit 0
fi

curl -fsS -X POST \
  -H "X-Management-Key: ${MGMT_KEY}" \
  -H "Content-Type: application/json" \
  --data-binary "@${IN_FILE}" \
  "${BASE_URL}/v0/management/usage/import" >/dev/null

echo "[usage-import] restored from ${IN_FILE}"
