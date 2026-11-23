package agents

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// HandleInstallScript serves a dynamically-generated install.sh bash script.
// GET /install.sh — public, no auth required.
func HandleInstallScript() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverURL := publicServerURL(r)

		script := generateInstallScript(serverURL)

		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		w.Header().Set("Content-Disposition", "inline; filename=\"install.sh\"")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(script))
	}
}

// HandleDownloadAgent serves the agent binary file.
// GET /api/v1/downloads/agent — public, no auth required.
func HandleDownloadAgent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		binaryPath := os.Getenv("SENTRIX_AGENT_BINARY_PATH")
		if binaryPath == "" {
			binaryPath = "/opt/sentrix/bin/sentrix-agent"
		}

		info, err := os.Stat(binaryPath)
		if err != nil || info.IsDir() {
			http.Error(w, "Agent binary not available on this server", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"sentrix-agent\"")
		http.ServeFile(w, r, binaryPath)
	}
}

func generateInstallScript(serverURL string) string {
	// Strip trailing slash if present
	serverURL = strings.TrimRight(serverURL, "/")

	return fmt.Sprintf(`#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────
#  SentriX Universal Agent Installer
#  Usage: curl -sSL %[1]s/install.sh | sudo bash -s -- --token <TOKEN>
# ─────────────────────────────────────────────────────────────────
set -euo pipefail

SENTRIX_SERVER="%[1]s"
TOKEN=""

# ── Parse arguments ──────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --token)  TOKEN="$2"; shift 2 ;;
    --token=*) TOKEN="${1#*=}"; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$TOKEN" ]]; then
  echo "ERROR: --token is required." >&2
  echo "Usage: curl -sSL ${SENTRIX_SERVER}/install.sh | sudo bash -s -- --token <TOKEN>" >&2
  exit 1
fi

# ── Verify root ──────────────────────────────────────────────────
if [[ "$(id -u)" -ne 0 ]]; then
  echo "ERROR: This script must be run as root (use sudo)." >&2
  exit 1
fi

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║          SentriX Agent — Universal Installer            ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "  Server:   ${SENTRIX_SERVER}"
echo "  Hostname: $(hostname)"
echo "  Arch:     $(uname -m)"
echo "  OS:       $(uname -s)"
echo ""

# ── Detect architecture ──────────────────────────────────────────
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)   ARCH="x86_64" ;;
  aarch64|arm64)   ARCH="aarch64" ;;
  *)
    echo "ERROR: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# ── Create system user ──────────────────────────────────────────
echo "[1/6] Creating sentrix system user..."
if ! id -u sentrix >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin sentrix
  echo "       Created user: sentrix"
else
  echo "       User sentrix already exists"
fi

# ── Create directories ──────────────────────────────────────────
echo "[2/6] Creating directories..."
install -d /opt/sentrix/bin
install -d /etc/sentrix
install -d -m 0700 -o sentrix -g sentrix /var/lib/sentrix 2>/dev/null || install -d -m 0700 /var/lib/sentrix
echo "       /opt/sentrix/bin  /etc/sentrix  /var/lib/sentrix"

# ── Download agent binary ───────────────────────────────────────
echo "[3/6] Downloading agent binary from ${SENTRIX_SERVER}..."
TMPBIN="$(mktemp)"
HTTP_CODE=$(curl -sSL -o "$TMPBIN" -w "%%{http_code}" "${SENTRIX_SERVER}/api/v1/downloads/agent")
if [[ "$HTTP_CODE" != "200" ]]; then
  echo "ERROR: Failed to download agent binary (HTTP ${HTTP_CODE})" >&2
  rm -f "$TMPBIN"
  exit 1
fi
install -m 0755 "$TMPBIN" /opt/sentrix/bin/sentrix-agent
rm -f "$TMPBIN"
echo "       Installed → /opt/sentrix/bin/sentrix-agent"

# ── Install systemd unit ────────────────────────────────────────
echo "[4/6] Installing systemd service..."
cat > /etc/systemd/system/sentrix-agent.service <<'UNIT'
[Unit]
Description=SentriX Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=sentrix
Group=sentrix
StateDirectory=sentrix
WorkingDirectory=/var/lib/sentrix
EnvironmentFile=/etc/sentrix/agent.env
ExecStart=/opt/sentrix/bin/sentrix-agent
Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=read-only

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
echo "       sentrix-agent.service installed"

# ── Enroll with SentriX server ──────────────────────────────────
echo "[5/6] Enrolling agent with ${SENTRIX_SERVER}..."
HOSTNAME_VAL="$(hostname)"
PLATFORM="linux"

ENROLL_RESP=$(curl -sS -X POST "${SENTRIX_SERVER}/api/v1/agent/enroll" \
  -H 'Content-Type: application/json' \
  -d "{
    \"token\": \"${TOKEN}\",
    \"hostname\": \"${HOSTNAME_VAL}\",
    \"platform\": \"${PLATFORM}\",
    \"architecture\": \"${ARCH}\",
    \"agent_version\": \"1.2.0\"
  }")

if ! echo "$ENROLL_RESP" | grep -q '"agent_id"'; then
  echo "ERROR: Enrollment failed:" >&2
  echo "$ENROLL_RESP" >&2
  exit 1
fi

AGENT_ID="$(echo "$ENROLL_RESP" | sed -n 's/.*"agent_id":"\([^"]*\)".*/\1/p')"
CREDENTIAL="$(echo "$ENROLL_RESP" | sed -n 's/.*"credential":"\([^"]*\)".*/\1/p')"

if [[ -z "$AGENT_ID" || -z "$CREDENTIAL" ]]; then
  echo "ERROR: Failed to parse enrollment response." >&2
  exit 1
fi

# Parse host and port from server URL
SERVER_HOST_PORT="${SENTRIX_SERVER#*://}"
SERVER_HOST="${SERVER_HOST_PORT%%:*}"
SERVER_PORT="${SERVER_HOST_PORT##*:}"
SERVER_PORT="${SERVER_PORT%%/*}"

# Determine TLS mode
USE_TLS="true"
if echo "$SENTRIX_SERVER" | grep -qi '^http://'; then
  USE_TLS="false"
fi

(
  umask 077
  cat > /etc/sentrix/agent.env <<EOF
SENTRIX_SERVER_HOST=${SERVER_HOST}
SENTRIX_SERVER_PORT=${SERVER_PORT}
SENTRIX_USE_TLS=${USE_TLS}
SENTRIX_AGENT_ID=${AGENT_ID}
SENTRIX_AGENT_CREDENTIAL=${CREDENTIAL}
EOF
)
chmod 0600 /etc/sentrix/agent.env
chown root:sentrix /etc/sentrix/agent.env 2>/dev/null || chown root:root /etc/sentrix/agent.env
echo "       Agent ID:    ${AGENT_ID}"
echo "       Config:      /etc/sentrix/agent.env"

# ── Start the service ────────────────────────────────────────────
echo "[6/6] Starting sentrix-agent service..."
systemctl enable --now sentrix-agent
echo "       Service started and enabled"

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  ✓  SentriX Agent installed and enrolled successfully!  ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "  Agent ID:    ${AGENT_ID}"
echo "  Server:      ${SENTRIX_SERVER}"
echo "  Config:      /etc/sentrix/agent.env"
echo "  Logs:        journalctl -u sentrix-agent -f"
echo ""
echo "  The node will appear on your Fleet Overview within ~30 seconds."
echo ""
`, serverURL)
}
