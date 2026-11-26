#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 4 ]]; then
  echo "Usage: sudo ./scripts/enroll-agent.sh <server_host> <server_port> <enrollment_token> <agent_version>" >&2
  exit 1
fi

SERVER_HOST="$1"
SERVER_PORT="$2"
ENROLLMENT_TOKEN="$3"
AGENT_VERSION="$4"

HOSTNAME="$(hostname)"
PLATFORM="linux"
ARCH="$(uname -m)"

PROTOCOL="https"
USE_TLS="true"
if [[ "${SENTRIX_ALLOW_INSECURE_HTTP:-false}" == "true" || "${SENTRIX_INSECURE_HTTP:-false}" == "true" ]]; then
  echo "WARNING: Insecure plaintext HTTP enrollment allowed by override." >&2
  PROTOCOL="http"
  USE_TLS="false"
fi

if [[ "${SERVER_HOST}" =~ ^https?:// ]]; then
  TARGET_URL="${SERVER_HOST}:${SERVER_PORT}/api/v1/agent/enroll"
else
  TARGET_URL="${PROTOCOL}://${SERVER_HOST}:${SERVER_PORT}/api/v1/agent/enroll"
fi

CURL_OPTS=(-sS --proto '=https,http')
if [[ -n "${SENTRIX_CA_CERT:-}" ]]; then
  CURL_OPTS+=(--cacert "${SENTRIX_CA_CERT}")
fi

RESPONSE="$(
  curl "${CURL_OPTS[@]}" -X POST "${TARGET_URL}" \
    -H 'Content-Type: application/json' \
    -d '{
      "token": "'"${ENROLLMENT_TOKEN}"'",
      "hostname": "'"${HOSTNAME}"'",
      "platform": "'"${PLATFORM}"'",
      "architecture": "'"${ARCH}"'",
      "agent_version": "'"${AGENT_VERSION}"'"
    }'
)"

if ! echo "$RESPONSE" | grep -q '"agent_id"'; then
  echo "Enrollment failed:" >&2
  echo "$RESPONSE" >&2
  exit 1
fi

AGENT_ID="$(echo "$RESPONSE" | sed -n 's/.*"agent_id":"\([^"]*\)".*/\1/p')"
CREDENTIAL="$(echo "$RESPONSE" | sed -n 's/.*"credential":"\([^"]*\)".*/\1/p')"

if [[ -z "$AGENT_ID" || -z "$CREDENTIAL" ]]; then
  echo "Failed to parse enrollment response." >&2
  exit 1
fi

install -d -m 0755 /etc/sentrix

(
  umask 077
  cat > /etc/sentrix/agent.env <<EOF
SENTRIX_SERVER_HOST=${SERVER_HOST}
SENTRIX_SERVER_PORT=${SERVER_PORT}
SENTRIX_USE_TLS=${USE_TLS}
SENTRIX_ALLOW_INSECURE_HTTP=$([[ "${USE_TLS}" == "false" ]] && echo 1 || echo 0)
SENTRIX_AGENT_ID=${AGENT_ID}
SENTRIX_AGENT_CREDENTIAL=${CREDENTIAL}
EOF
)

chmod 0600 /etc/sentrix/agent.env

if id -u sentrix >/dev/null 2>&1; then
  chown root:sentrix /etc/sentrix/agent.env
else
  chown root:root /etc/sentrix/agent.env
fi

echo "Agent enrolled successfully."
echo "Agent ID: ${AGENT_ID}"
echo "Configuration written to /etc/sentrix/agent.env"
