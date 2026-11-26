#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────
#  Update a systemd-installed SentriX server from this git checkout.
#
#  Layout it expects (the one the VPS uses):
#    /opt/sentrix/bin/sentrix-server   server binary (sentrix-server.service)
#    /opt/sentrix/bin/sentrix-agent    agent binary served to install.sh
#    /etc/sentrix/server.env           DATABASE_URL, SENTRIX_WEB_DIR, ...
#
#  Builds with Docker only (no Go/CMake needed on the host), backs up the
#  database and current files, swaps in the new build, restarts the service,
#  and rolls the files back automatically if the health check fails.
#
#  Usage:  cd /opt/sentrix/repo && git pull && sudo bash scripts/update-server.sh
# ─────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${SENTRIX_ENV_FILE:-/etc/sentrix/server.env}"
INSTALL_DIR="${SENTRIX_INSTALL_DIR:-/opt/sentrix}"
SERVICE="${SENTRIX_SERVICE:-sentrix-server}"
TS="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="${INSTALL_DIR}/backups/${TS}"

die() { echo "ERROR: $*" >&2; exit 1; }

[[ "$(id -u)" -eq 0 ]] || die "run as root (sudo)."
command -v docker >/dev/null || die "docker is required."
[[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

# Read the service's own config so paths match what it actually uses.
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a
[[ -n "${DATABASE_URL:-}" ]] || die "DATABASE_URL not set in $ENV_FILE"
WEB_DIR="${SENTRIX_WEB_DIR:-${INSTALL_DIR}/web/dist}"
MIG_DIR="${SENTRIX_MIGRATIONS_DIR:-${INSTALL_DIR}/migrations}"
SERVER_BIN="${INSTALL_DIR}/bin/sentrix-server"
AGENT_BIN="${SENTRIX_AGENT_BINARY_PATH:-${INSTALL_DIR}/bin/sentrix-agent}"
PORT="${PORT:-8080}"

COMMIT="$(git -C "$REPO_DIR" rev-parse --short HEAD)"
VERSION="$(cat "$REPO_DIR/VERSION" 2>/dev/null || echo dev)"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

echo "SentriX update -> v${VERSION} (${COMMIT})"
echo "  web dir:        $WEB_DIR"
echo "  migrations dir: $MIG_DIR"
echo "  backups:        $BACKUP_DIR"
echo ""

# ── 1. Database backup ──────────────────────────────────────────
echo "[1/6] Backing up the database..."
mkdir -p "$BACKUP_DIR"
docker run --rm --network host postgres:17-alpine \
  pg_dump --no-owner "$DATABASE_URL" | gzip > "$BACKUP_DIR/db.sql.gz"
[[ "$(stat -c %s "$BACKUP_DIR/db.sql.gz")" -gt 1024 ]] || die "database backup looks empty; aborting before any change."
echo "       $(du -h "$BACKUP_DIR/db.sql.gz" | cut -f1) -> $BACKUP_DIR/db.sql.gz"

# ── 2. Build server + web ───────────────────────────────────────
echo "[2/6] Building server and web UI (Docker)..."
IMAGE="sentrix-server-build:${COMMIT}"
docker build -q -t "$IMAGE" \
  --build-arg VERSION="v${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  --build-arg DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  "$REPO_DIR" >/dev/null
CID="$(docker create "$IMAGE")"
docker cp "$CID:/app/sentrix-server" "$STAGE/sentrix-server"
docker cp "$CID:/app/web" "$STAGE/web"
docker cp "$CID:/app/migrations" "$STAGE/migrations"
docker rm "$CID" >/dev/null
[[ -s "$STAGE/sentrix-server" && -f "$STAGE/web/index.html" ]] || die "server build output incomplete."

# ── 3. Build agent ──────────────────────────────────────────────
# Ubuntu 22.04 (glibc 2.35, OpenSSL 3) so the binary runs on 22.04+ / Debian 12+.
echo "[3/6] Building agent (Docker)..."
docker run --rm -v "$REPO_DIR/agent:/src:ro" -v "$STAGE:/out" ubuntu:22.04 bash -c '
  set -e
  apt-get update -qq >/dev/null
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq build-essential cmake libssl-dev >/dev/null
  cmake -S /src -B /tmp/build -DCMAKE_BUILD_TYPE=Release >/dev/null
  cmake --build /tmp/build -j"$(nproc)" >/dev/null
  cp /tmp/build/sentrix-agent /out/sentrix-agent
'
[[ -s "$STAGE/sentrix-agent" ]] || die "agent build produced no binary."

# ── 4. Back up current files ────────────────────────────────────
echo "[4/6] Backing up current binaries and web UI..."
cp -a "$SERVER_BIN" "$BACKUP_DIR/sentrix-server"
[[ -f "$AGENT_BIN" ]] && cp -a "$AGENT_BIN" "$BACKUP_DIR/sentrix-agent"
[[ -d "$WEB_DIR" ]] && cp -a "$WEB_DIR" "$BACKUP_DIR/web"

rollback() {
  echo "Rolling back binaries and web UI from $BACKUP_DIR ..." >&2
  install -m 0755 "$BACKUP_DIR/sentrix-server" "$SERVER_BIN"
  [[ -f "$BACKUP_DIR/sentrix-agent" ]] && install -m 0755 "$BACKUP_DIR/sentrix-agent" "$AGENT_BIN"
  if [[ -d "$BACKUP_DIR/web" ]]; then rm -rf "$WEB_DIR"; cp -a "$BACKUP_DIR/web" "$WEB_DIR"; fi
  systemctl restart "$SERVICE"
  echo "Rolled back. New migrations (if any ran) stay applied; the database backup is $BACKUP_DIR/db.sql.gz" >&2
}

# ── 5. Install and restart ──────────────────────────────────────
echo "[5/6] Installing and restarting ${SERVICE}..."
# From here on, any failure restores the previous files.
trap 'rollback; rm -rf "$STAGE"' ERR
install -m 0755 "$STAGE/sentrix-server" "${SERVER_BIN}.new" && mv -f "${SERVER_BIN}.new" "$SERVER_BIN"
install -m 0755 "$STAGE/sentrix-agent" "${AGENT_BIN}.new" && mv -f "${AGENT_BIN}.new" "$AGENT_BIN"
mkdir -p "$MIG_DIR"
cp -f "$STAGE"/migrations/*.sql "$MIG_DIR"/
rm -rf "${WEB_DIR}.new"
cp -a "$STAGE/web" "${WEB_DIR}.new"
rm -rf "$WEB_DIR"
mv "${WEB_DIR}.new" "$WEB_DIR"
systemctl restart "$SERVICE"

# ── 6. Health check ─────────────────────────────────────────────
echo "[6/6] Waiting for the server to become ready..."
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${PORT}/health/ready" >/dev/null 2>&1; then
    echo ""
    journalctl -u "$SERVICE" --since "-3 min" --no-pager | grep -iE "applied migration|monitor started|error|fatal" | tail -n 30 || true
    echo ""
    echo "✓ SentriX updated to v${VERSION} (${COMMIT}) and healthy."
    echo "  Backup: $BACKUP_DIR"
    exit 0
  fi
  sleep 2
done

echo "Server did not become ready within 60s. Recent logs:" >&2
journalctl -u "$SERVICE" --since "-3 min" --no-pager | tail -n 40 >&2
rollback
exit 1
