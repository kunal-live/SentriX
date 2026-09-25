# Changelog

## v1.2.0

### Added
- **Enterprise Observability Suite (Phases 0–25)**:
  - **Distributed Tracing & APM (Phase 6)**: W3C TraceContext propagation (`traceparent`, `tracestate`), span watermarking, latency waterfall views, and service bottleneck analysis.
  - **Dynamic Service Map & Synthetics (Phase 7)**: Live topology graph with edge RPS and p99 latency rates; multi-target synthetic monitoring (HTTP, TCP, DNS) with SSL expiry detection.
  - **SLO & SLI Management (Phase 8)**: Availability and latency Service Level Indicators with real-time error budget burn rate calculations and alerting.
  - **Custom Dashboards (Phase 9)**: Drag-and-drop dashboard grid builder with interactive timeseries charts, gauge clusters, status tables, and markdown widgets.
  - **Agent 2.0 & Systems Collector (Phase 10)**: Low-footprint native C11 daemon with bounded memory footprint, dynamic socket connection tracing, and process tree grouping.
  - **Container & Kubernetes Deep Monitoring (Phases 11–12)**: Unified cgroup v1/v2 container metrics aggregator and Kubernetes cluster topology, pod lifecycle, and node pressure monitoring.
  - **AI-Assisted Root Cause Analysis (Phase 13)**: Metric anomaly detection, log clustering, and automated incident hypothesis generation.
  - **Automated Runbooks & Remediation (Phase 14)**: Parameterized remediation workflows with dry-run capabilities and full audit trails.
  - **Enterprise Alert Engine 2.0 & Incident Management (Phases 2–5, 15)**: Multi-channel notifications (Slack, PagerDuty, Webhooks) with resilient dead-letter queue (DLQ) retry backoff, incident timeline comments, and escalation policies.
  - **Centralized Log Streaming & Full-Text Search (Phase 5)**: Real-time log streaming, full-text regex querying, and severity filters.

---

## v1.1.0

### Added
- **Native C Agent Configuration & Synthetic Checks**:
  - Dynamic configuration loader parsing `/etc/sentrix/agent.env` or local `.env` files.
  - Multi-source environment variable and command line flag overrides (`-s`, `-p`, `-i`, `-t`, `-c`).
  - Integrated periodic synthetic checks execution loop alongside telemetry collection.
  - Multi-interface network traffic aggregator across physical non-loopback Linux network devices.
- **Backend Features & Administration**:
  - Server decommissioning endpoint (`DELETE /api/v1/servers/{serverID}`) with database cascade cleanup.
  - Agent enrollment token generator (`POST /api/v1/agents/enrollment-tokens`) with 24-hour expiration.
  - Team & Access Control management endpoints (`/api/v1/users`) supporting RBAC (`ADMIN`, `OPERATOR`, `VIEWER`).
  - Notification Channels endpoints (`/api/v1/notifications/channels`) supporting Slack, Discord, PagerDuty, and custom webhook dispatch testing.
  - WebSocket ticket authentication alias (`/auth/ws-ticket`).
  - TimescaleDB 30-day retention policies and hourly continuous aggregates migration (`00008_retention_and_aggregates.sql`).
  - Go unit tests for `auth` (password hashing & JWT validation) and `metrics` (credential hashing & payload structure).
- **Web UI & Management Dashboard**:
  - Node Enrollment modal on Overview page with one-click token generation and `curl | bash` installation commands.
  - Team & Access Control page with role indicators, invite modal, and account deletion.
  - Alert Notification Channels page with test dispatch simulations.
  - Server decommissioning dialog on the Node Detail page.

---

## v1.0.0

### Added
- Core telemetry pipeline: agent → API → TimescaleDB.
- Linux C agent with CPU, memory, disk, network, load, and uptime collectors.
- Agent enrollment tokens.
- Unique agent credentials.
- Agent credential rotation.
- Agent revocation.
- Current-state engine with online/suspect/offline detection.
- Historical metric charts.
- Threshold alert engine with pending state, cooldown, and hysteresis.
- Incident lifecycle: open, acknowledge, resolve.
- Incident comments and timeline.
- Process, service, port, and command checks.
- Notification jobs with webhook delivery and retry/backoff.
- Realtime WebSocket updates with polling fallback.
- Authentication with Argon2id, JWT access tokens, and rotating refresh tokens.
- RBAC: ADMIN, OPERATOR, VIEWER.
- Audit logging for sensitive operations.
- Rate limiting for login, refresh, enrollment, and telemetry.
- Telemetry replay protection using sequence numbers and timestamp windows.
- Security headers.
- Body size limits.
- Self-monitoring metrics endpoint.
- Docker server image with embedded web UI.
- Automatic database migrations.
- systemd agent packaging.
- Backup and restore scripts.

### Security
- Login brute-force protection.
- One-time enrollment tokens.
- Hashed agent credentials.
- Restricted CORS in production.
- TLS deployment documentation.

### Notes
- V1 agent transport is HTTP. Use trusted networks or TLS sidecars for untrusted networks. Native agent TLS is planned for v1.2.
