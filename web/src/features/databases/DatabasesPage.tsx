import { useState, useEffect } from 'react';
import {
  Database,
  Layers,
  Clock,
  Zap,
  Lock,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Activity,
  Sparkles,
  Copy,
  Check,
  Gauge,
  Server,
} from 'lucide-react';
import { MetricCard } from '../../components/ui/Primitives';
import { api } from '../../api/client';

interface PostgresDiagnostics {
  instance_name: string;
  version: string;
  is_live_db: boolean;
  status: 'OPTIMAL' | 'WARNING' | 'CRITICAL';
  active_connections: number;
  max_connections: number;
  connection_saturation_percent: number;
  cache_hit_ratio: number;
  transactions_per_sec: number;
  rollback_ratio_pct: number;
  replication_lag_bytes: number;
  replication_lag_seconds: number;
  deadlocks_detected: number;
  wal_generation_mbps: number;
  database_size_bytes: number;
  pool_metrics: {
    active_connections: number;
    idle_connections: number;
    idle_in_tx_connections: number;
    waiting_connections: number;
    max_connections: number;
    saturation_percent: number;
    app_pool_acquired: number;
    app_pool_idle: number;
    app_pool_constructing: number;
    app_pool_total: number;
    app_pool_max: number;
    app_pool_acquire_count: number;
    app_pool_empty_acquires: number;
    app_pool_avg_wait_ms: number;
  };
  query_analytics: {
    total_qps: number;
    p50_latency_ms: number;
    p95_latency_ms: number;
    p99_latency_ms: number;
    slow_query_count: number;
    slow_threshold_ms: number;
    select_pct: number;
    insert_pct: number;
    update_pct: number;
    delete_pct: number;
  };
  active_queries: {
    pid: number;
    user: string;
    database: string;
    state: string;
    wait_event_type?: string;
    duration_seconds: number;
    query: string;
    started_at: string;
  }[];
  slow_queries: {
    query_id: string;
    calls: number;
    mean_time_ms: number;
    p95_time_ms: number;
    max_time_ms: number;
    total_time_ms: number;
    rows_affected: number;
    database: string;
    user: string;
    seq_scan_heavy: boolean;
    severity: 'CRITICAL' | 'WARNING' | 'NORMAL';
    query: string;
  }[];
  index_recommendations: {
    id: string;
    schema_name: string;
    table_name: string;
    seq_scans: number;
    idx_scans: number;
    live_rows: number;
    reason: string;
    suggested_index_sql: string;
    impact: 'HIGH' | 'MEDIUM' | 'LOW';
    estimated_speedup: string;
  }[];
  locks: {
    pid: number;
    lock_type: string;
    mode: string;
    granted: boolean;
    relation: string;
    blocked_by_pid: number;
    wait_duration?: string;
    query?: string;
  }[];
}

export function DatabasesPage() {
  const [data, setData] = useState<PostgresDiagnostics | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'analytics' | 'indexes' | 'pool' | 'active' | 'locks'>('analytics');
  const [slowThreshold, setSlowThreshold] = useState<number>(100);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  async function loadData(threshold = slowThreshold) {
    try {
      const res = await api.get<PostgresDiagnostics>(`/databases/postgres?slow_threshold_ms=${threshold}`);
      setData(res.data);
    } catch (err) {
      console.error('Failed to load database diagnostics', err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadData(slowThreshold);
    const interval = setInterval(() => loadData(slowThreshold), 5000);
    return () => clearInterval(interval);
  }, [slowThreshold]);

  function copySQL(id: string, sql: string) {
    navigator.clipboard.writeText(sql);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  }

  const sizeGB = data ? (data.database_size_bytes / (1024 * 1024 * 1024)).toFixed(2) : '0.00';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
              <Database className="text-cyan" size={26} />
              Database Performance Insights
            </h1>
            <span
              className={`px-2.5 py-0.5 rounded-full text-xs font-mono border ${
                data?.is_live_db
                  ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                  : 'bg-cyan/20 text-cyan border-cyan/30'
              }`}
            >
              {data?.is_live_db ? 'Live TimescaleDB' : 'Telemetry Simulation'}
            </span>
          </div>
          <p className="text-xs text-muted mt-1">
            Query performance analytics, slow query detection, automated index recommendations, and live connection pool monitoring.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-2 bg-surface/80 border border-border px-3 py-1.5 rounded-xl text-xs font-mono text-muted">
            <span className="w-2 h-2 rounded-full bg-healthy animate-pulse" />
            <span>{data?.version || 'PostgreSQL 16 (TimescaleDB)'}</span>
            <span className="text-border">•</span>
            <span>{sizeGB} GB</span>
          </div>
          <button
            onClick={() => loadData(slowThreshold)}
            className="p-2 rounded-xl bg-surface border border-border hover:border-primary/50 text-muted hover:text-white transition-colors"
            title="Refresh diagnostics"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* Primary KPI Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Connection Pool"
          value={data ? `${data.active_connections} / ${data.max_connections}` : '—'}
          subtitle={
            data
              ? `${data.connection_saturation_percent}% Saturated • App Pool: ${data.pool_metrics?.app_pool_acquired}/${data.pool_metrics?.app_pool_max}`
              : 'Loading...'
          }
          icon={<Layers size={18} className="text-primary-light" />}
        />
        <MetricCard
          title="Query Latency (P95 / P99)"
          value={data ? `${data.query_analytics?.p95_latency_ms}ms / ${data.query_analytics?.p99_latency_ms}ms` : '—'}
          subtitle={data ? `${data.query_analytics?.total_qps.toLocaleString()} QPS • P50: ${data.query_analytics?.p50_latency_ms}ms` : 'Loading...'}
          icon={<Gauge size={18} className="text-cyan" />}
        />
        <MetricCard
          title="Cache Hit Ratio"
          value={data ? `${data.cache_hit_ratio}%` : '—'}
          subtitle={data ? `${data.transactions_per_sec.toLocaleString()} TPS • Deadlocks: ${data.deadlocks_detected}` : 'Shared Buffer Efficiency'}
          icon={<Zap size={18} className="text-healthy" />}
        />
        <MetricCard
          title="Slow Queries & Index Alerts"
          value={data ? `${data.query_analytics?.slow_query_count} Slow` : '—'}
          subtitle={
            data
              ? `${data.index_recommendations?.length || 0} Index Recommendations (>${slowThreshold}ms)`
              : 'Threshold profiler'
          }
          icon={<AlertTriangle size={18} className="text-warning" />}
        />
      </div>

      {/* Tabs selector */}
      <div className="flex flex-wrap items-center gap-2 border-b border-border pb-2">
        <button
          onClick={() => setTab('analytics')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'analytics'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Activity size={13} />
          <span>Query Analytics & Slow Queries ({data?.slow_queries?.length || 0})</span>
        </button>
        <button
          onClick={() => setTab('indexes')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'indexes'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Sparkles size={13} />
          <span>Index Recommendations ({data?.index_recommendations?.length || 0})</span>
        </button>
        <button
          onClick={() => setTab('pool')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'pool'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Server size={13} />
          <span>Connection Pool Monitoring</span>
        </button>
        <button
          onClick={() => setTab('active')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'active'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Clock size={13} />
          <span>Active Queries ({data?.active_queries?.length || 0})</span>
        </button>
        <button
          onClick={() => setTab('locks')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'locks'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Lock size={13} />
          <span>Lock Contention ({data?.locks?.length || 0})</span>
        </button>
      </div>

      {/* TAB 1: Query Performance Analytics & Slow Query Detection */}
      {tab === 'analytics' && (
        <div className="space-y-4">
          {/* Operation Mix & Latency Distribution Bar */}
          {data?.query_analytics && (
            <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div>
                  <h2 className="text-sm font-semibold text-white">Query Performance Analytics</h2>
                  <p className="text-xs text-muted">
                    Workload operation mix and percentile latency distribution across {data.query_analytics.total_qps.toLocaleString()} queries/sec
                  </p>
                </div>

                <div className="flex items-center gap-2 text-xs font-mono">
                  <span className="text-muted">Slow Threshold:</span>
                  {[50, 100, 250].map((ms) => (
                    <button
                      key={ms}
                      onClick={() => setSlowThreshold(ms)}
                      className={`px-2.5 py-1 rounded-lg border transition-colors ${
                        slowThreshold === ms
                          ? 'bg-warning/20 text-warning border-warning/40 font-bold'
                          : 'bg-surface text-muted border-border hover:text-white'
                      }`}
                    >
                      &gt; {ms}ms
                    </button>
                  ))}
                </div>
              </div>

              {/* Stacked SQL Operation Mix */}
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs font-mono">
                  <span className="text-blue-400">SELECT {data.query_analytics.select_pct}%</span>
                  <span className="text-emerald-400">INSERT {data.query_analytics.insert_pct}%</span>
                  <span className="text-amber-400">UPDATE {data.query_analytics.update_pct}%</span>
                  <span className="text-rose-400">DELETE {data.query_analytics.delete_pct}%</span>
                </div>
                <div className="w-full h-2.5 rounded-full bg-surface overflow-hidden flex">
                  <div className="bg-blue-500 h-full" style={{ width: `${data.query_analytics.select_pct}%` }} />
                  <div className="bg-emerald-500 h-full" style={{ width: `${data.query_analytics.insert_pct}%` }} />
                  <div className="bg-amber-500 h-full" style={{ width: `${data.query_analytics.update_pct}%` }} />
                  <div className="bg-rose-500 h-full" style={{ width: `${data.query_analytics.delete_pct}%` }} />
                </div>
              </div>
            </div>
          )}

          {/* Slow Query Profiler Table */}
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-white">
                Slow Query Detection (pg_stat_statements)
              </h2>
              <span className="text-xs text-muted font-mono">
                Flagging queries exceeding {slowThreshold}ms mean execution time
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="text-[11px] font-mono text-muted uppercase border-b border-border bg-surface/50">
                  <tr>
                    <th className="py-2.5 px-3">Severity</th>
                    <th className="py-2.5 px-3">Calls</th>
                    <th className="py-2.5 px-3">Mean / P95</th>
                    <th className="py-2.5 px-3">Max Time</th>
                    <th className="py-2.5 px-3">Total Time</th>
                    <th className="py-2.5 px-3">SQL Pattern</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border font-mono">
                  {data?.slow_queries?.map((sq) => (
                    <tr key={sq.query_id} className="hover:bg-surface/40 transition-colors">
                      <td className="py-3 px-3">
                        <span
                          className={`px-2 py-0.5 rounded-full text-[10px] font-bold border ${
                            sq.severity === 'CRITICAL'
                              ? 'bg-critical/20 text-critical border-critical/30'
                              : sq.severity === 'WARNING'
                              ? 'bg-warning/20 text-warning border-warning/30'
                              : 'bg-healthy/20 text-healthy border-healthy/30'
                          }`}
                        >
                          {sq.severity}
                        </span>
                      </td>
                      <td className="py-3 px-3 text-white">{sq.calls.toLocaleString()}</td>
                      <td className="py-3 px-3">
                        <span className={sq.mean_time_ms >= slowThreshold ? 'text-warning font-bold' : 'text-cyan'}>
                          {sq.mean_time_ms.toFixed(1)}ms
                        </span>
                        <span className="text-muted"> / {sq.p95_time_ms.toFixed(1)}ms</span>
                      </td>
                      <td className="py-3 px-3 text-rose-400">{sq.max_time_ms.toFixed(1)}ms</td>
                      <td className="py-3 px-3 text-muted">{(sq.total_time_ms / 1000).toFixed(2)}s</td>
                      <td className="py-3 px-3 max-w-md">
                        <div className="text-text-dim truncate font-mono text-[11px]" title={sq.query}>
                          {sq.query}
                        </div>
                        {sq.seq_scan_heavy && (
                          <span className="inline-block mt-1 px-1.5 py-0.2 rounded text-[10px] bg-amber-500/15 text-amber-400 border border-amber-500/30">
                            Seq Scan Heavy
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* TAB 2: Index Recommendations */}
      {tab === 'indexes' && (
        <div className="space-y-4">
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold text-white flex items-center gap-2">
                  <Sparkles size={16} className="text-primary-light" />
                  <span>Automated Index Recommendations</span>
                </h2>
                <p className="text-xs text-muted mt-0.5">
                  Analyzed from PostgreSQL <code className="text-cyan">pg_stat_user_tables</code> sequential vs. index scan ratios
                </p>
              </div>
              <span className="text-xs font-mono text-muted">
                {data?.index_recommendations?.length || 0} opportunities detected
              </span>
            </div>

            <div className="grid grid-cols-1 gap-4">
              {data?.index_recommendations?.map((rec) => (
                <div
                  key={rec.id}
                  className="p-4 rounded-xl bg-surface/60 border border-border space-y-3 hover:border-primary/40 transition-colors"
                >
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-mono font-bold border ${
                          rec.impact === 'HIGH'
                            ? 'bg-critical/20 text-critical border-critical/30'
                            : 'bg-warning/20 text-warning border-warning/30'
                        }`}
                      >
                        {rec.impact} IMPACT
                      </span>
                      <span className="font-mono text-sm font-bold text-white">
                        {rec.schema_name}.{rec.table_name}
                      </span>
                      <span className="px-2 py-0.5 rounded bg-healthy/15 text-healthy text-[11px] font-mono">
                        {rec.estimated_speedup}
                      </span>
                    </div>
                    <div className="text-xs font-mono text-muted">
                      Seq Scans: <span className="text-warning font-bold">{rec.seq_scans.toLocaleString()}</span> • Idx Scans:{' '}
                      <span className="text-healthy">{rec.idx_scans.toLocaleString()}</span> • Live Rows:{' '}
                      <span className="text-white">{rec.live_rows.toLocaleString()}</span>
                    </div>
                  </div>

                  <p className="text-xs text-text-dim">{rec.reason}</p>

                  <div className="flex items-center justify-between gap-2 p-3 rounded-lg bg-[#0d1016] border border-border font-mono text-xs text-cyan">
                    <code className="truncate">{rec.suggested_index_sql}</code>
                    <button
                      onClick={() => copySQL(rec.id, rec.suggested_index_sql)}
                      className="flex items-center gap-1 px-2.5 py-1 rounded bg-surface hover:bg-primary text-text-dim hover:text-white text-[11px] shrink-0 transition-colors"
                    >
                      {copiedId === rec.id ? <Check size={12} className="text-healthy" /> : <Copy size={12} />}
                      <span>{copiedId === rec.id ? 'Copied' : 'Copy SQL'}</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 3: Connection Pool Monitoring */}
      {tab === 'pool' && data?.pool_metrics && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Application Go pgxpool Metrics */}
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-white">Application Pool (Go pgxpool)</h2>
              <span className="text-xs font-mono text-healthy">
                {data.pool_metrics.app_pool_total} / {data.pool_metrics.app_pool_max} Conns
              </span>
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs font-mono">
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Acquired (In Use)</div>
                <div className="text-lg font-bold text-cyan mt-1">{data.pool_metrics.app_pool_acquired}</div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Idle Ready</div>
                <div className="text-lg font-bold text-healthy mt-1">{data.pool_metrics.app_pool_idle}</div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Total Acquire Count</div>
                <div className="text-lg font-bold text-white mt-1">{data.pool_metrics.app_pool_acquire_count.toLocaleString()}</div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Empty Acquire Waits</div>
                <div className={`text-lg font-bold mt-1 ${data.pool_metrics.app_pool_empty_acquires > 0 ? 'text-warning' : 'text-healthy'}`}>
                  {data.pool_metrics.app_pool_empty_acquires} ({data.pool_metrics.app_pool_avg_wait_ms}ms avg)
                </div>
              </div>
            </div>
          </div>

          {/* PostgreSQL Server Backend Connections */}
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-white">PostgreSQL Server Backends</h2>
              <span className="text-xs font-mono text-cyan">
                {data.pool_metrics.saturation_percent}% of {data.pool_metrics.max_connections} max_connections
              </span>
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs font-mono">
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Active Executing</div>
                <div className="text-lg font-bold text-healthy mt-1">{data.pool_metrics.active_connections}</div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Idle Backends</div>
                <div className="text-lg font-bold text-white mt-1">{data.pool_metrics.idle_connections}</div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Idle in Transaction</div>
                <div className={`text-lg font-bold mt-1 ${data.pool_metrics.idle_in_tx_connections > 0 ? 'text-warning' : 'text-healthy'}`}>
                  {data.pool_metrics.idle_in_tx_connections}
                </div>
              </div>
              <div className="p-3 rounded-xl bg-surface/60 border border-border">
                <div className="text-muted text-[11px]">Waiting on Locks</div>
                <div className={`text-lg font-bold mt-1 ${data.pool_metrics.waiting_connections > 0 ? 'text-critical' : 'text-healthy'}`}>
                  {data.pool_metrics.waiting_connections}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* TAB 4: Active Queries */}
      {tab === 'active' && (
        <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white">Currently Executing SQL Queries</h2>
            <span className="text-xs text-muted font-mono">pg_stat_activity realtime feed</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="text-[11px] font-mono text-muted uppercase border-b border-border bg-surface/50">
                <tr>
                  <th className="py-2.5 px-3">PID</th>
                  <th className="py-2.5 px-3">User</th>
                  <th className="py-2.5 px-3">Database</th>
                  <th className="py-2.5 px-3">State</th>
                  <th className="py-2.5 px-3">Duration</th>
                  <th className="py-2.5 px-3">Query</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-mono">
                {data?.active_queries?.map((q) => (
                  <tr key={q.pid} className="hover:bg-surface/40 transition-colors">
                    <td className="py-3 px-3 text-cyan">{q.pid}</td>
                    <td className="py-3 px-3 text-white">{q.user}</td>
                    <td className="py-3 px-3 text-muted">{q.database}</td>
                    <td className="py-3 px-3">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] ${
                          q.state === 'active'
                            ? 'bg-healthy/20 text-healthy border border-healthy/30'
                            : 'bg-surface-highlight text-muted border border-border'
                        }`}
                      >
                        {q.state}
                      </span>
                    </td>
                    <td className="py-3 px-3 text-warning">{q.duration_seconds}s</td>
                    <td className="py-3 px-3 text-text-dim max-w-md truncate font-sans text-xs">{q.query}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 5: Lock Contention */}
      {tab === 'locks' && (
        <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white">PostgreSQL Lock Contention (pg_locks)</h2>
            <span className="text-xs text-muted font-mono">Granted & blocked relation locks</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="text-[11px] font-mono text-muted uppercase border-b border-border bg-surface/50">
                <tr>
                  <th className="py-2.5 px-3">PID</th>
                  <th className="py-2.5 px-3">Relation</th>
                  <th className="py-2.5 px-3">Type / Mode</th>
                  <th className="py-2.5 px-3">Granted</th>
                  <th className="py-2.5 px-3">Query</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-mono">
                {data?.locks?.map((l, idx) => (
                  <tr key={`${l.pid}-${idx}`} className="hover:bg-surface/40 transition-colors">
                    <td className="py-3 px-3 text-cyan">{l.pid}</td>
                    <td className="py-3 px-3 text-white">{l.relation}</td>
                    <td className="py-3 px-3 text-muted">
                      {l.lock_type} • {l.mode}
                    </td>
                    <td className="py-3 px-3">
                      {l.granted ? (
                        <span className="inline-flex items-center gap-1 text-healthy">
                          <CheckCircle2 size={12} /> Granted
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-critical font-bold">
                          <AlertTriangle size={12} /> Waiting
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-3 text-text-dim max-w-md truncate">{l.query || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
