# SentriX — High-Performance Infrastructure Monitoring & Observability

[![CI](https://github.com/kunal-live/SentriX/actions/workflows/ci.yml/badge.svg)](https://github.com/kunal-live/SentriX/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/kunal-live/SentriX?color=6366f1)](https://github.com/kunal-live/SentriX/releases)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![Agent](https://img.shields.io/badge/Agent-C11%20Native-blue?logo=c)](./agent)
[![Database](https://img.shields.io/badge/TimescaleDB-PostgreSQL%2016-336791?logo=postgresql)](https://www.timescale.com/)
[![Frontend](https://img.shields.io/badge/UI-React%20%2B%20Vite%20%2B%20Tailwind-61DAFB?logo=react)](./web)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

SentriX is a modern, lightweight, high-performance infrastructure monitoring and observability platform engineered for sub-millisecond metric collection, OpenTelemetry distributed tracing, central log aggregation, Kubernetes fleet tracking, AI-assisted root cause analysis (RCA), and automated operational runbooks across distributed Linux environments.

---

## ⚡ Key Highlights

- **Ultra-Lightweight C Agent**: Native C11 daemon with sub-10ms collection latency, negligible CPU footprint (<0.5%), and minimal memory consumption (<15MB RSS) reading directly from Linux `/proc`.
- **High-Throughput Go Ingestion Core**: Non-blocking concurrent metric ingestion engine built in Go with streaming buffers, rate-limiting, and sequence-based replay protection.
- **OpenTelemetry Distributed Tracing**: Native OTLP HTTP ingestion (`/otlp/v1/traces`), microsecond-precision span waterfall timelines, service dependency maps, and latency percentiles.
- **Centralized Log Aggregation**: Multi-tenant log search, severity level filtering (`DEBUG`, `INFO`, `WARN`, `ERROR`), and seamless correlation with trace IDs and server IDs.
- **Kubernetes Fleet Observability**: Multi-node health, cluster CPU/Memory pressure, Pod phase tracking (`Running`, `Pending`, `Failed`, `CrashLoopBackOff`), and deployment replica status.
- **AI-Powered Root Cause Analysis (RCA)**: Correlates metric spikes, anomaly signals, error log patterns, and failing trace spans to generate instant postmortems and remediation advice.
- **Operational Runbooks & Safe Automation**: Pre-flight checks, automated remediation procedures (connection pool drains, cache flushes, agent diagnostics), and execution audit logs.
- **TimescaleDB & PostgreSQL 16**: High-velocity time-series storage with continuous aggregates, automated rollups, and 30-day retention policies.
- **Obsidian Glassmorphic UI**: Sleek dark-mode interface built with React 18, Tailwind CSS, Recharts, and TanStack Query.
- **Zero-Dependency Demo Mode**: Standalone in-memory fleet simulation for instant evaluation without requiring a live database.

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
                                |  |  - Synthetic Probes   |  |
                                |  +-----------+-----------+  |
                                +--------------|--------------+
                                               | HTTP POST (JSON / Telemetry)
                                               v
+-------------------------------------------------------------------------------------------------+
|                                    SentriX Server (Go)                                          |
|                                                                                                 |
|   +-----------------------+     +-----------------------+     +-----------------------------+   |
|   |  Ingestion & Security | --> |   Alerting Engine     | --> |  Notification Hub           |   |
|   |  - Token & JWT Guard  |     |   - Threshold Rules   |     |  - Slack Webhook            |   |
|   |  - Sequence Replay    |     |   - Incident Engine   |     |  - Discord Webhook          |   |
|   |  - Rate Limiter       |     |   - Hysteresis/Cool   |     |  - PagerDuty API            |   |
|   +-----------+-----------+     +-----------------------+     +-----------------------------+   |
|               |                                                                                 |
|               +-----------------------------+-----------------------------+                     |
|               |                             |                             |                     |
|               v                             v                             v                     |
|   +-----------------------+     +-----------------------+     +-----------------------------+   |
|   | Observability Engines |     | Intelligence & Ops    |     | Streaming & Realtime        |   |
|   | - OTLP Traces (Spans) |     | - AI RCA Synthesis    |     | - WebSocket Hub             |   |
|   | - Centralized Logs    |     | - Auto Runbooks       |     | - Live Metric Streaming     |   |
|   | - Kubernetes Nodes    |     | - Service Dependency  |     | - State Sync                |   |
|   +-----------+-----------+     +-----------+-----------+     +--------------+--------------+   |
|               |                             |                                |                  |
|               +-----------------------------+--------------------------------+                  |
|                                             |                                                   |
|                                             v                                                   |
|                                  +--------------------+                                         |
|                                  | TimescaleDB (pg16) |                                         |
|                                  | - Hypertables      |                                         |
|                                  | - Continuous Aggs  |                                         |
|                                  | - Retention Policy |                                         |
|                                  +--------------------+                                         |
+-------------------------------------------------------------------------------------------------+
                                              |
                                              v
                         +------------------------------------------+
                         |           React Web Dashboard            |
                         |  - Fleet Bento Grid & 1-Click Enrollment |
                         |  - OTel Traces Waterfall & Central Logs  |
                         |  - Kubernetes Cluster & Workload Views   |
                         |  - AI Incident Triage & Runbook Runner   |
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

> **Platform Note**: The SentriX server control plane is cross-platform and can be compiled and run on Linux, macOS, or Windows. The native C agent (`sentrix-agent`) is specifically engineered for Linux server hosts, reading directly from the Linux `/proc` kernel filesystem with systemd service integration.

> **Production Security Note**: When connecting to a persistent database in production, SentriX generates a cryptographically secure 20-character random password on first run and outputs it to the server logs (or uses `SENTRIX_ADMIN_PASSWORD`). Change the administrator password immediately upon first login. Brute-force protection and IP rate limiting are active by default on `/api/v1/auth/login`.

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

## 🔍 Observability Capabilities

### 1. OpenTelemetry Distributed Tracing
SentriX provides full trace visibility across distributed microservices:
- **OTLP Ingestion**: Native endpoints at `/api/v1/otlp/v1/traces`, `/api/v1/v1/traces`, and `/api/v1/traces`.
- **Waterfall Timeline**: Detailed span duration, child spans, parent-child relationships, and root operations.
- **Latency & Error Breakdown**: Highlights slowest database queries, external API calls, and unhandled 5xx errors.

### 2. Centralized Log Aggregation
- **Multi-Stream Search**: Filter logs by service name, log severity (`DEBUG`, `INFO`, `WARN`, `ERROR`), host ID, and time window.
- **Trace-to-Log Correlation**: Jump directly from a failing trace span to its correlated log entries using `trace_id`.
- **High-Velocity Ingest**: Batch ingestion endpoint at `/api/v1/logs/batch` with streaming ingestion buffers.

### 3. Kubernetes & Container Observability
- **Cluster & Node Health**: Monitor CPU/Memory node capacity, kernel version, OS image, and kubelet status.
- **Workload & Pod Lifecycle**: Real-time distribution of Pod phases (`Running`, `Pending`, `Failed`, `CrashLoopBackOff`).
- **Deployment Tracking**: Track desired vs. available replicas, restart counters, and container crash events.

### 4. AI-Powered Root Cause Analysis (RCA)
- **Multi-Signal Anomaly Correlation**: When an incident occurs, SentriX correlates metric spikes, log errors, and trace spans.
- **Automated Incident Narratives**: Generates an incident summary, identified root cause hypothesis, and actionable remediation steps (`/api/v1/ai/rca`).

### 5. Operational Runbooks & Safe Automation
- **Automated Remediation**: Trigger verified operational runbooks (e.g., PostgreSQL connection pool drain, cache purges, agent offline buffer diagnostics) directly from the UI or via API (`/api/v1/runbooks/{id}/execute`).
- **Safety Gates & Auditing**: Review pre-execution step plans and track execution history with operator audit trails.

---

## 💻 Native C Agent Installation

### Secure Fleet Node Enrollment
1. Navigate to the **Fleet Overview** in the dashboard.
2. Click **"Enroll Node"**.
3. Generate a single-use, 24-hour cryptographically hashed enrollment token.
4. Execute the enrollment command over HTTPS with certificate verification:

```bash
# Download and install agent binary
sudo ./scripts/install-agent.sh /path/to/sentrix-agent

# Enroll the node with single-use token over HTTPS
sudo ./scripts/enroll-agent.sh <your-sentrix-host> 443 <ENROLLMENT_TOKEN> 1.0.0

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
```

---

## 📦 API Reference Overview

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Authenticate user credentials & issue JWT tokens |
| `POST` | `/api/v1/auth/refresh` | Refresh expired access tokens |
| `POST` | `/api/v1/auth/ws-ticket` | Generate temporary single-use ticket for WebSocket streaming |
| `GET` | `/api/v1/dashboard/summary` | Global fleet counts, health statistics, and metrics |
| `GET` | `/api/v1/servers` | List registered server nodes and online/offline status |
| `DELETE` | `/api/v1/servers/{id}` | Decommission server node and purge telemetry (`ADMIN` only) |
| `GET` | `/api/v1/servers/{id}/metrics` | Historical time-series metrics query |
| `POST` | `/api/v1/agent/telemetry` | High-throughput metric ingestion from C agents |
| `POST` | `/api/v1/agents/enrollment-tokens` | Generate 24-hour single-use agent enrollment tokens |
| `GET` | `/api/v1/traces` | Query distributed OpenTelemetry trace spans |
| `POST` | `/api/v1/otlp/v1/traces` | OTLP/HTTP distributed trace span ingestion |
| `GET` | `/api/v1/logs` | Centralized log search across services and severity levels |
| `POST` | `/api/v1/logs/batch` | High-throughput structured log ingestion |
| `GET` | `/api/v1/kubernetes/overview` | Kubernetes cluster overview and node capacity status |
| `GET` | `/api/v1/kubernetes/workloads` | Kubernetes pod health, phases, and container restarts |
| `POST` | `/api/v1/ai/rca` | Trigger AI multi-signal root cause analysis on incidents |
| `GET` | `/api/v1/runbooks` | List executable operational remediation runbooks |
| `POST` | `/api/v1/runbooks/{id}/execute` | Execute an operational remediation playbook |
| `GET` | `/api/v1/users` | List workspace team members |
| `POST` | `/api/v1/users/invite` | Invite team members with RBAC (`ADMIN`, `OPERATOR`, `VIEWER`) |
| `GET` | `/api/v1/notifications/channels` | List notification destinations (masked URLs) |
| `POST` | `/api/v1/notifications/channels` | Create Slack/Discord/PagerDuty webhook channels |
| `POST` | `/api/v1/notifications/channels/{id}/test` | Trigger SSRF-safe instant webhook verification test |

---

## 🧪 Testing & Verification

Run the test suite across components:

```bash
# Run Go unit tests
cd server
go test -v ./...

# Build and type-check frontend
cd ../web
npm run build
```

---

## 📄 License

SentriX is licensed under the [Apache 2.0 License](LICENSE).
