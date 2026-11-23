# SentriX — High-Performance Infrastructure Monitoring & Observability

[![CI](https://github.com/kunal-live/SentriX/actions/workflows/ci.yml/badge.svg)](https://github.com/kunal-live/SentriX/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/kunal-live/SentriX?color=6366f1)](https://github.com/kunal-live/SentriX/releases)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![Agent](https://img.shields.io/badge/Agent-C11%20Native-blue?logo=c)](./agent)
[![Database](https://img.shields.io/badge/TimescaleDB-PostgreSQL%2016-336791?logo=postgresql)](https://www.timescale.com/)
[![Frontend](https://img.shields.io/badge/UI-React%20%2B%20Vite%20%2B%20Tailwind-61DAFB?logo=react)](./web)
[![Pricing](https://img.shields.io/badge/Plans-100%25%20Free%20%26%20Open-success)](LICENSE)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

SentriX is a modern, lightweight, high-performance infrastructure monitoring and observability platform engineered for sub-millisecond metric collection, real-time WebSocket telemetry streaming, centralized structured log search, synthetic health probes, automated incident triaging, and security auditing across distributed Linux server fleets.

**100% Free & Open Source**: No tiers, no billing gates, no metering, and no locked features.

---

## ⚡ Key Highlights

- **Ultra-Lightweight C Agent (`sentrix-agent`)**: Native C11 daemon with sub-10ms collection latency, negligible CPU footprint (<0.5%), and minimal memory consumption (<15MB RSS) reading directly from Linux `/proc`.
- **Hardened Agent TLS Transport**: Mandatory HTTPS/TLS hostname verification (`SSL_set1_host` / `X509_check_host`), strict certificate validation, partial write protection, and fail-safe credential handling.
- **High-Throughput Go Ingestion Core**: Non-blocking concurrent metric ingestion engine built in Go with streaming buffers, request body size limiting, IP rate limiting, and trusted-proxy validation.
- **TimescaleDB & PostgreSQL 16**: High-velocity time-series storage with continuous aggregates, automated rollups, and 30-day retention policies.
- **Real-Time Fleet Overview**: Live node health matrix, CPU/RAM/Disk saturation gauges, instantaneous filtering, and single-use 24-hour cryptographically hashed agent enrollment.
- **Custom Observability Workspaces**: Dynamic dashboards with live time-series charts, customizable metric gauges, real error/warning log streams, and active incident triages.
- **Cross-Node Metrics Explorer**: Multi-server time-series analytics powered by TimescaleDB queries (`/api/v1/metrics/query`) across CPU, memory, and disk utilization.
- **Centralized Structured Logs**: High-velocity log aggregation with multi-field search, severity level filtering (`DEBUG`, `INFO`, `WARN`, `ERROR`), and real-time live tailing.
- **Reliability, Incidents & Alert Rules**: Metric threshold evaluation engine with hysteresis/cooling, incident lifecycles (`OPEN`, `ACKNOWLEDGED`, `RESOLVED`), silence windows, and multi-channel notifications (Slack, Discord, PagerDuty, Webhooks).
- **Synthetic Health Probes**: Automated HTTP and TCP uptime checks with response latency and availability tracking.
- **Developer Platform & RBAC**: Scoped ingestion tokens, programmatic developer API keys, role-based access control (`ADMIN`, `OPERATOR`, `VIEWER`), and immutable security audit logs.
- **Enterprise Security Hardened**: HttpOnly + SameSite cookies, session revocation on password changes, brute-force mitigation, and zero plaintext credential exposure.

---

## 🏗️ Architecture

```
                                +-----------------------------+
                                |    Target Host (Linux)      |
                                |  +-----------------------+  |
                                |  |  sentrix-agent (C11)  |  |
                                |  |  - CPU / Load Avg     |  |
                                |  |  - RAM / Swap         |  |
                                |  |  - Multi-NIC Traffic  |  |
                                |  |  - Disk / I/O IOPS    |  |
                                |  +-----------+-----------+  |
                                +--------------|--------------+
                                               | HTTPS POST (TLS Hostname Verified)
                                               v
+-------------------------------------------------------------------------------------------------+
|                                    SentriX Server (Go)                                          |
|                                                                                                 |
|   +-----------------------+     +-----------------------+     +-----------------------------+   |
|   |  Ingestion & Security | --> |   Alerting Engine     | --> |  Notification Hub           |   |
|   |  - Token & JWT Guard  |     |   - Threshold Rules   |     |  - Slack Webhook            |   |
|   |  - Body Size Limiter  |     |   - Incident Engine   |     |  - Discord Webhook          |   |
|   |  - Trusted Proxy IP   |     |   - Silence Windows   |     |  - PagerDuty API            |   |
|   |  - Rate Limiter       |     |   - Hysteresis/Cool   |     |  - Custom Webhooks          |   |
|   +-----------+-----------+     +-----------------------+     +-----------------------------+   |
|               |                                                                                 |
|               +-----------------------------+-----------------------------+                     |
|               |                             |                             |                     |
|               v                             v                             v                     |
|   +-----------------------+     +-----------------------+     +-----------------------------+   |
|   | Observability Core    |     | Synthetic Checks      |     | Streaming & Realtime        |   |
|   | - TimescaleDB Metrics |     | - HTTP Endpoint Probe |     | - WebSocket Hub             |   |
|   | - Centralized Logs    |     | - TCP Port Probe      |     | - Live Metric Streaming     |   |
|   | - Multi-Node Query    |     | - Latency & Uptime    |     | - State Sync                |   |
|   +-----------+-----------+     +-----------+-----------+     +--------------+--------------+   |
|               |                             |                                |                  |
|               +-----------------------------+--------------------------------+                  |
|                                             |                                                   |
|                                             v                                                   |
|                                  +--------------------+                                         |
|                                  | TimescaleDB (pg16) |                                         |
|                                  | - metric_cpu       |                                         |
|                                  | - metric_memory    |                                         |
|                                  | - metric_disk      |                                         |
|                                  | - logs & incidents |                                         |
|                                  +--------------------+                                         |
+-------------------------------------------------------------------------------------------------+
                                              |
                                              v
                         +------------------------------------------+
                         |       React 18 Observability Canvas      |
                         |  - Fleet Overview & 1-Click Enrollment   |
                         |  - Custom Dashboards & Live Telemetry    |
                         |  - Metrics Explorer (TimescaleDB Query)  |
                         |  - Central Logs Search & Live Tail       |
                         |  - Incident Triage, Rules & Silences     |
                         |  - Synthetic Health Checks               |
                         |  - Developer API Keys & Audit Trail      |
                         +------------------------------------------+
```

---

## 🚀 Quick Start

### Option 1: Standalone Demo Mode (Zero Dependencies)

Run SentriX locally in seconds without needing PostgreSQL or TimescaleDB installed:

```bash
# 1. Build the web frontend
cd web
npm install
npm run build

# 2. Build and run the server in demo mode
# On Linux / macOS:
cd ../server
go build -o ../dist/sentrix-server ./cmd/sentrix-server
cd ../dist
SENTRIX_DEMO_MODE=true ./sentrix-server

# On Windows (PowerShell):
cd ../server
go build -o ../dist/sentrix-server.exe ./cmd/sentrix-server
cd ../dist
$env:SENTRIX_DEMO_MODE="true"; .\sentrix-server.exe
```

Access the dashboard at **[http://localhost:8080](http://localhost:8080)**:
- **Demo User**: `admin@sentrix.local`
- **Demo Password**: `admin12345` (only enabled in standalone demo mode)

> **Platform Note**: The SentriX server control plane is cross-platform (Linux, macOS, Windows). The native C agent (`sentrix-agent`) is specifically engineered for Linux server hosts, reading directly from the Linux `/proc` kernel filesystem with systemd service integration.

---

### Option 2: Docker Compose (Full Production Stack)

Deploy the entire production observability suite (TimescaleDB database + Go server + built React web frontend) in a single command:

```bash
# 1. Clone the repository
git clone https://github.com/kunal-live/SentriX.git
cd SentriX

# 2. Copy example environment configuration and generate secure secrets
cp .env.example .env

# 3. Spin up the entire production stack
docker compose up -d --build
```

The stack automatically:
- Starts **TimescaleDB 16** with persistent volume storage, loopback port binding (`127.0.0.1:5432`), and healthchecks.
- Automatically executes database schema migrations from `server/migrations`.
- Builds and runs the **SentriX Server** in a minimal distroless container running as a non-root user.
- Serves the integrated **React Web Dashboard** on **[http://localhost:8080](http://localhost:8080)**.

To stop the stack:
```bash
docker compose down
```

---

## 🔍 Core Observability Features

### 1. Fleet Overview (`/`)
- **Real-Time Host Grid**: Live CPU, memory, and disk saturation bars, host uptime, kernel version, and agent release.
- **Instant Search & Status Filter**: Search by hostname, IP address, or node ID, and filter by status (`ONLINE`, `SUSPECT`, `OFFLINE`).
- **1-Click Agent Enrollment**: Generate cryptographically hashed, single-use tokens valid for 24 hours with a universal `curl | bash` one-liner installer.
- **Server Details Drill-Down** (`/servers/:id`): Deep inspection with interactive time-series charts across multiple lookback windows (15m, 1h, 6h, 24h, 7d).

### 2. Custom Dashboards (`/dashboards`)
- **Operational Workspaces**: Build custom monitoring canvases with multi-metric line charts, saturation gauges, error log streams, and active incident lists.
- **Node Switching**: Toggle metrics instantly between any enrolled production node.
- **Real-Time Auto-Refresh**: 10-second automatic background refresh synchronized with incoming telemetry.

### 3. Metrics Explorer (`/metrics`)
- **TimescaleDB Multi-Node Analytics**: Query and compare metrics across multiple server nodes simultaneously.
- **Flexible Metrics & Aggregation**: Select metrics (`system.cpu.utilization`, `system.memory.utilization`, `system.disk.utilization`) across custom time intervals (`15m`, `1h`, `6h`, `24h`, `7d`).
- **Raw Export & API Access**: Integrated endpoint at `/api/v1/metrics/query` for automated scraping or external tooling.

### 4. Centralized Structured Logs (`/logs`)
- **High-Velocity Aggregation**: Central log repository powered by PostgreSQL with full-text search.
- **Multi-Field Filtering**: Filter by service name, log level (`DEBUG`, `INFO`, `WARN`, `ERROR`), host ID, and time window.
- **Live Tail Mode**: Real-time log stream streaming directly to the browser console.

### 5. Incidents, Alert Rules & Silences (`/incidents`, `/alerts`)
- **Threshold Rule Engine**: Configure rules with custom thresholds, duration windows, severity levels (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`), and automatic resolution.
- **Incident Lifecycle**: Comprehensive triage workflows supporting `OPEN`, `ACKNOWLEDGED`, and `RESOLVED` statuses with operator notes.
- **Maintenance Windows (Silences)**: Suppress alert firing during scheduled maintenance windows to avoid notification fatigue.
- **Notification Channels**: Webhook dispatchers for Slack, Discord, PagerDuty, and custom HTTPS endpoints with SSRF protection.

### 6. Synthetic Health Checks (`/checks`)
- **Multi-Protocol Monitoring**: Create recurring HTTP/HTTPS and TCP endpoint probe checks.
- **Availability & Latency Tracking**: Tracks response times, DNS lookup latency, TLS negotiation times, and uptime percentages.

### 7. Developer Platform & Scoped API Keys (`/api-keys`)
- **Programmatic Ingestion**: Issue scoped API keys for external metric or log shippers.
- **Secret Security**: API keys are shown only once upon generation and stored as SHA-256 hashes in the database.

### 8. Team Management & Audit Trail (`/users`, `/audit`)
- **Role-Based Access Control**: Assign granular roles (`ADMIN`, `OPERATOR`, `VIEWER`) to team members.
- **Immutable Security Audit Log**: Tracks all security-critical operations, authentication attempts, rule creations, and user actions with actor IP addresses and timestamps.

---

## 💻 Agent Installation — Universal 1-Line Installer

### Quick Enrollment (Datadog / Netdata Style)

SentriX provides a **single `curl | bash` command** to install, enroll, and start the agent on any Linux host — no manual downloads, no multi-step scripts:

```bash
curl -sSL http://<YOUR_SENTRIX_SERVER>:8080/install.sh | sudo bash -s -- --token <ENROLLMENT_TOKEN>
```

**What this does automatically:**
1. Creates the `sentrix` system user and directories
2. Downloads the agent binary directly from your SentriX server
3. Installs and configures the systemd service
4. Enrolls the node using the single-use token
5. Writes secure credentials to `/etc/sentrix/agent.env` (mode `0600`)
6. Starts and enables the agent service

**To generate a token:**
1. Navigate to the **Fleet Overview** in the dashboard
2. Click **"Enroll Node"** — a token and the full `curl` command are auto-generated
3. Copy and run the command on the target server

The node will appear on your Fleet Overview within ~30 seconds.

> **API Endpoints**: The installer uses `GET /install.sh` (dynamic script) and `GET /api/v1/downloads/agent` (binary download), both served directly by your SentriX server instance.

### Manual Installation (Advanced)

For air-gapped environments or custom setups:

```bash
# Download and install agent binary
sudo ./scripts/install-agent.sh /path/to/sentrix-agent

# Enroll the node with single-use token
sudo ./scripts/enroll-agent.sh <your-sentrix-host> 8080 <ENROLLMENT_TOKEN> 1.0.0

# Start and enable the agent service (runs non-root with 0600 config)
sudo systemctl enable --now sentrix-agent
```

### Manual Compilation on Linux

```bash
cd agent
mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make -j$(nproc)

# Run agent (configuration loaded from /etc/sentrix/agent.env):
./sentrix-agent -s "192.168.1.50" -p 8080 -i "prod-app-01" -c 5
```

Agent configuration is stored securely in `/etc/sentrix/agent.env` (`chmod 0600`):
```env
SENTRIX_HOST=192.168.1.50
SENTRIX_PORT=8080
SENTRIX_AGENT_ID=srv-node-01
SENTRIX_TOKEN=stx_enroll_token_xyz
SENTRIX_INTERVAL=5
SENTRIX_ENFORCE_TLS=true
```

---

## 🔒 Security Hardening

SentriX incorporates strict enterprise security controls:
- **Mandatory TLS Hostname Verification**: Agent enforces `SSL_set1_host` and `X509_check_host` to prevent MITM attacks.
- **Zero Refresh Token Leakage**: Refresh tokens are delivered strictly via `HttpOnly`, `SameSite=Strict` cookies and are never exposed in JSON response bodies.
- **Trusted Reverse Proxy IP Handling**: IP rate limiting and audit logging sanitize forwarded headers, trusting `X-Forwarded-For` only from configured proxy CIDRs (`SENTRIX_TRUSTED_PROXIES`).
- **Request Body Size Limits**: Global and per-route `http.MaxBytesReader` enforcement protects against payload-based memory exhaustion attacks.
- **Session Revocation on Password Change**: Changing a user's password immediately invalidates all issued refresh tokens for that account.
- **Hashed Agent Credentials**: Agents authenticate using SHA-256 hashed credentials stored in TimescaleDB with zero plaintext fallbacks.

---

## 📦 API Reference Overview

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Authenticate user credentials & issue JWT tokens |
| `POST` | `/api/v1/auth/refresh` | Refresh expired access tokens via HttpOnly cookie |
| `POST` | `/api/v1/auth/logout` | Revoke active refresh session and clear cookies |
| `POST` | `/api/v1/auth/change-password` | Update user password & revoke all active sessions |
| `POST` | `/api/v1/auth/ws-ticket` | Generate temporary single-use ticket for WebSocket streaming |
| `GET` | `/api/v1/dashboard/summary` | Global fleet counts, health statistics, and metrics |
| `GET` | `/api/v1/servers` | List registered server nodes and online/offline status |
| `GET` | `/api/v1/servers/{id}` | Retrieve details and latest metrics for a single node |
| `DELETE` | `/api/v1/servers/{id}` | Decommission server node and purge telemetry (`ADMIN` only) |
| `GET` | `/api/v1/servers/{id}/metrics` | Historical time-series metrics query for a single node |
| `GET` | `/api/v1/metrics/query` | Cross-node multi-server TimescaleDB time-series query |
| `POST` | `/api/v1/agent/telemetry` | High-throughput metric ingestion from C agents |
| `POST` | `/api/v1/agents/enrollment-tokens` | Generate 24-hour single-use agent enrollment tokens |
| `GET`  | `/install.sh` | Dynamic universal installer shell script |
| `GET`  | `/api/v1/downloads/agent` | Download agent binary from server |
| `GET` | `/api/v1/logs` | Centralized log search across services and severity levels |
| `POST` | `/api/v1/logs/batch` | High-throughput structured log ingestion |
| `GET` | `/api/v1/incidents` | List active, acknowledged, and resolved incidents |
| `PATCH` | `/api/v1/incidents/{id}/status` | Update incident triage status and resolution notes |
| `GET` | `/api/v1/alerts/rules` | List configured metric alert threshold rules |
| `POST` | `/api/v1/alerts/rules` | Create a new alert rule with threshold and duration |
| `GET` | `/api/v1/checks` | List synthetic HTTP/TCP health checks |
| `POST` | `/api/v1/checks` | Create a new synthetic health check probe |
| `GET` | `/api/v1/api-keys` | List active developer API keys |
| `POST` | `/api/v1/api-keys` | Generate a new developer API key |
| `DELETE` | `/api/v1/api-keys/{id}` | Revoke an API key |
| `GET` | `/api/v1/users` | List workspace team members |
| `POST` | `/api/v1/users/invite` | Invite team members with RBAC (`ADMIN`, `OPERATOR`, `VIEWER`) |
| `GET` | `/api/v1/audit/logs` | Query immutable security audit log trail |
| `GET` | `/api/v1/notifications/channels` | List notification destinations (masked URLs) |
| `POST` | `/api/v1/notifications/channels` | Create Slack/Discord/PagerDuty webhook channels |
| `POST` | `/api/v1/notifications/channels/{id}/test` | Trigger SSRF-safe instant webhook verification test |

---

## 🧪 Testing & Verification

Run the test suite across components:

```bash
# Run Go unit tests and vet
cd server
go test -v ./...
go vet ./...

# Build and type-check frontend
cd ../web
npm run build
```

---

## 📄 License

SentriX is licensed under the [Apache 2.0 License](LICENSE).
