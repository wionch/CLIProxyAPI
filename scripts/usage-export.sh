#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8317}"
MGMT_KEY="${MGMT_KEY:?MGMT_KEY is required}"
OUT_FILE="${OUT_FILE:-/data/usage.json}"

tmp="${OUT_FILE}.tmp"

curl -fsS \
  -H "X-Management-Key: ${MGMT_KEY}" \
  "${BASE_URL}/v0/management/usage/export" > "${tmp}"

mv "${tmp}" "${OUT_FILE}"
echo "[usage-export] saved to ${OUT_FILE}"
