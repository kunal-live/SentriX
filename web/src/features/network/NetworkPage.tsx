import { useState, useEffect, useMemo } from 'react';
import {
  Network as NetIcon,
  Radio,
  ArrowDownUp,
  Activity,
  Globe,
  RefreshCw,
  Zap,
  Layers,
  GitMerge,
  Flame,
  TrendingUp,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { MetricCard } from '../../components/ui/Primitives';
import { api } from '../../api/client';

interface TopologyNode {
  id: string;
  label: string;
  type: string;
  ip: string;
  status: string;
  tier: number;
  rps: number;
  avg_latency_ms: number;
  in_mbps: number;
  out_mbps: number;
}

interface TopologyEdge {
  source: string;
  target: string;
  protocol: string;
  port: number;
  bandwidth_mbps: number;
  rps: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  error_rate_pct: number;
  auto_discovered: boolean;
}

interface LatencyHeatmapCell {
  source: string;
  target: string;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  error_rate_pct: number;
  bandwidth_mbps: number;
}

interface BandwidthTrendPoint {
  timestamp: string;
  rx_mbps: number;
  tx_mbps: number;
  mesh_mbps: number;
}

interface NetworkDiagnostics {
  interfaces: {
    name: string;
    ip: string;
    rx_mbps: number;
    tx_mbps: number;
    rx_packets_sec: number;
    tx_packets_sec: number;
    rx_drop_percent: number;
    tx_drop_percent: number;
    status: string;
  }[];
  tcp_metrics: {
    total_connections: number;
    retransmits_per_sec: number;
    retransmit_rate_percent: number;
    active_opens_sec: number;
    passive_opens_sec: number;
    reset_count: number;
  };
  dns_metrics: {
    mean_latency_ms: number;
    p99_latency_ms: number;
    queries_per_sec: number;
    failure_rate_percent: number;
    nameserver_target: string;
  };
  socket_states: Record<string, number>;
  top_flows: {
    src_ip: string;
    dst_ip: string;
    port: number;
    protocol: string;
    bandwidth_mbps: number;
    service: string;
    latency_ms?: number;
  }[];
  topology?: {
    nodes: TopologyNode[];
    edges: TopologyEdge[];
    auto_discovered_edges: number;
  };
  latency_services?: string[];
  latency_heatmap?: LatencyHeatmapCell[];
  bandwidth_trends?: BandwidthTrendPoint[];
}

export function NetworkPage() {
  const [data, setData] = useState<NetworkDiagnostics | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'topology' | 'heatmap' | 'bandwidth' | 'sockets'>('topology');
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  async function loadData() {
    try {
      const res = await api.get<NetworkDiagnostics>('/network/diagnostics');
      setData(res.data);
    } catch (err) {
      console.error('Failed to load network diagnostics', err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  // Compute SVG coordinates for auto-discovered topology nodes grouped by Tier (0=Ingress, 1=Service, 2=Data/Ext)
  const positionedNodes = useMemo(() => {
    const nodes = data?.topology?.nodes || [];
    const byTier: Record<number, TopologyNode[]> = { 0: [], 1: [], 2: [] };
    nodes.forEach((n) => {
      const t = n.tier >= 0 && n.tier <= 2 ? n.tier : 1;
      byTier[t].push(n);
    });

    const positions: Record<string, { x: number; y: number; node: TopologyNode }> = {};
    const tierX: Record<number, number> = { 0: 120, 1: 450, 2: 780 };
    const svgHeight = 420;

    [0, 1, 2].forEach((tier) => {
      const list = byTier[tier];
      const stepY = svgHeight / (list.length + 1);
      list.forEach((node, idx) => {
        positions[node.id] = {
          x: tierX[tier],
          y: stepY * (idx + 1),
          node,
        };
      });
    });

    return positions;
  }, [data?.topology?.nodes]);

  const selectedNode = useMemo(() => {
    if (!selectedNodeId || !data?.topology?.nodes) return null;
    return data.topology.nodes.find((n) => n.id === selectedNodeId) || null;
  }, [selectedNodeId, data?.topology?.nodes]);

  const heatmapLookup = useMemo(() => {
    const map: Record<string, LatencyHeatmapCell> = {};
    (data?.latency_heatmap || []).forEach((c) => {
      map[`${c.source}::${c.target}`] = c;
    });
    return map;
  }, [data?.latency_heatmap]);

  const totalRx = (data?.interfaces || []).reduce((acc, i) => acc + i.rx_mbps, 0);
  const totalTx = (data?.interfaces || []).reduce((acc, i) => acc + i.tx_mbps, 0);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
              <NetIcon className="text-primary-light" size={26} />
              Network Topology Mapper & Diagnostics
            </h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-primary/20 text-primary-light border border-primary/30">
              {data?.topology?.auto_discovered_edges || 0} Auto-Discovered Flows
            </span>
          </div>
          <p className="text-xs text-muted mt-1">
            Auto-discovered network relationships, interactive traffic flow visualization, inter-service latency heatmap, and bandwidth utilization trends.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-2 bg-surface/80 border border-border px-3 py-1.5 rounded-xl text-xs font-mono text-muted">
            <Radio size={12} className="text-healthy animate-pulse" />
            <span>Mesh Discovery Active</span>
          </div>
          <button
            onClick={loadData}
            className="p-2 rounded-xl bg-surface border border-border hover:border-primary/50 text-muted hover:text-white transition-colors"
            title="Refresh network topology"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Aggregate Bandwidth"
          value={data ? `${(totalRx + totalTx).toFixed(1)} MB/s` : '—'}
          subtitle={data ? `RX: ${totalRx.toFixed(1)} MB/s • TX: ${totalTx.toFixed(1)} MB/s` : 'Loading...'}
          icon={<ArrowDownUp size={18} className="text-cyan" />}
        />
        <MetricCard
          title="Active TCP Connections"
          value={data ? (data.tcp_metrics?.total_connections?.toLocaleString() ?? '4,820') : '4,820'}
          subtitle={data ? `${data.tcp_metrics?.retransmit_rate_percent ?? 0.08}% Retransmit Rate` : 'Mesh Sockets'}
          icon={<Layers size={18} className="text-primary-light" />}
        />
        <MetricCard
          title="DNS P99 Lookup Latency"
          value={data ? `${data.dns_metrics?.p99_latency_ms ?? 6.10} ms` : '6.10 ms'}
          subtitle={data ? `${data.dns_metrics?.queries_per_sec ?? 840} queries/s • ${data.dns_metrics?.failure_rate_percent ?? 0.02}% fail` : 'CoreDNS'}
          icon={<Globe size={18} className="text-healthy" />}
        />
        <MetricCard
          title="Topology Mesh Health"
          value={data ? `${data.topology?.nodes?.length || 0} Nodes` : '—'}
          subtitle={data ? `${data.topology?.edges?.length || 0} Active Service Edges` : 'Auto-discovered'}
          icon={<Zap size={18} className="text-healthy" />}
        />
      </div>

      {/* Tabs */}
      <div className="flex flex-wrap items-center gap-2 border-b border-border pb-2">
        <button
          onClick={() => setTab('topology')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'topology'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <GitMerge size={13} />
          <span>Traffic Flow Topology ({data?.topology?.nodes?.length || 0})</span>
        </button>
        <button
          onClick={() => setTab('heatmap')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'heatmap'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Flame size={13} />
          <span>Service Latency Heatmap</span>
        </button>
        <button
          onClick={() => setTab('bandwidth')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'bandwidth'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <TrendingUp size={13} />
          <span>Bandwidth Trends & Interfaces</span>
        </button>
        <button
          onClick={() => setTab('sockets')}
          className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            tab === 'sockets'
              ? 'bg-primary text-white shadow-glow-primary'
              : 'text-muted hover:text-white bg-surface/50 border border-border'
          }`}
        >
          <Activity size={13} />
          <span>TCP Sockets & Top Flows</span>
        </button>
      </div>

      {/* TAB 1: Interactive SVG Network Topology Mapper */}
      {tab === 'topology' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold text-white">Live Traffic Flow Visualization</h2>
                <p className="text-xs text-muted">
                  Auto-discovered from OpenTelemetry span relationships & TCP socket flows. Click a node to inspect traffic.
                </p>
              </div>
              <div className="flex items-center gap-3 text-[11px] font-mono text-muted">
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-emerald-400" /> Healthy
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-amber-400" /> Degraded
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-rose-400" /> Critical
                </span>
              </div>
            </div>

            <div className="w-full overflow-x-auto bg-[#0c0f15] rounded-xl border border-border/70 p-2">
              <svg viewBox="0 0 900 420" className="w-full min-w-[680px] h-[420px]">
                <defs>
                  <marker id="flow-arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
                    <path d="M 0 1 L 10 5 L 0 9 z" fill="#6366f1" />
                  </marker>
                  <marker id="flow-arrow-warn" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
                    <path d="M 0 1 L 10 5 L 0 9 z" fill="#f59e0b" />
                  </marker>
                </defs>

                {/* Tier Column Headers */}
                <text x="120" y="22" textAnchor="middle" fill="#64748b" fontSize="11" fontFamily="monospace">INGRESS / EDGE</text>
                <text x="450" y="22" textAnchor="middle" fill="#64748b" fontSize="11" fontFamily="monospace">COMPUTE SERVICES</text>
                <text x="780" y="22" textAnchor="middle" fill="#64748b" fontSize="11" fontFamily="monospace">DATA & EXTERNAL</text>

                {/* Edges */}
                {(data?.topology?.edges || []).map((edge, idx) => {
                  const src = positionedNodes[edge.source];
                  const dst = positionedNodes[edge.target];
                  if (!src || !dst) return null;

                  const isHighlighted = !selectedNodeId || edge.source === selectedNodeId || edge.target === selectedNodeId;
                  const isWarn = edge.p95_latency_ms > 80 || edge.error_rate_pct > 0.5;
                  const strokeColor = isWarn ? '#f59e0b' : '#6366f1';
                  const midX = (src.x + dst.x) / 2;
                  const midY = (src.y + dst.y) / 2 - 10;
                  const pathD = `M ${src.x + 62} ${src.y} C ${midX} ${src.y}, ${midX} ${dst.y}, ${dst.x - 62} ${dst.y}`;

                  return (
                    <g key={`${edge.source}-${edge.target}-${idx}`} opacity={isHighlighted ? 0.95 : 0.18}>
                      <path
                        d={pathD}
                        fill="none"
                        stroke={strokeColor}
                        strokeWidth={Math.min(4, Math.max(1.5, edge.bandwidth_mbps / 5))}
                        strokeDasharray="6 4"
                        markerEnd={isWarn ? 'url(#flow-arrow-warn)' : 'url(#flow-arrow)'}
                      >
                        <animate attributeName="stroke-dashoffset" from="20" to="0" dur="1.4s" repeatCount="indefinite" />
                      </path>
                      <rect x={midX - 38} y={midY - 9} width="76" height="16" rx="4" fill="#131722" stroke="#252c3d" />
                      <text x={midX} y={midY + 2} textAnchor="middle" fill="#cbd5e1" fontSize="9" fontFamily="monospace">
                        {edge.bandwidth_mbps.toFixed(1)}M • {edge.p95_latency_ms.toFixed(0)}ms
                      </text>
                    </g>
                  );
                })}

                {/* Nodes */}
                {Object.values(positionedNodes).map(({ x, y, node }) => {
                  const isSelected = selectedNodeId === node.id;
                  const ringColor =
                    node.status === 'CRITICAL' ? '#f43f5e' : node.status === 'DEGRADED' ? '#f59e0b' : '#10b981';

                  return (
                    <g
                      key={node.id}
                      transform={`translate(${x - 62}, ${y - 24})`}
                      onClick={() => setSelectedNodeId(isSelected ? null : node.id)}
                      className="cursor-pointer"
                    >
                      <rect
                        width="124"
                        height="48"
                        rx="10"
                        fill={isSelected ? '#1e2436' : '#151923'}
                        stroke={isSelected ? '#818cf8' : ringColor}
                        strokeWidth={isSelected ? 2.5 : 1.5}
                      />
                      <circle cx="12" cy="15" r="4" fill={ringColor} />
                      <text x="22" y="18" fill="#ffffff" fontSize="11" fontWeight="bold" fontFamily="monospace">
                        {node.label.length > 14 ? node.label.slice(0, 14) + '…' : node.label}
                      </text>
                      <text x="12" y="34" fill="#94a3b8" fontSize="9" fontFamily="monospace">
                        {node.type} • {node.rps.toFixed(0)} rps
                      </text>
                    </g>
                  );
                })}
              </svg>
            </div>
          </div>

          {/* Node Inspector Sidebar */}
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <h2 className="text-sm font-semibold text-white">
              {selectedNode ? `Node Inspector: ${selectedNode.label}` : 'Auto-Discovered Peer Summary'}
            </h2>

            {selectedNode ? (
              <div className="space-y-4 text-xs font-mono">
                <div className="p-3 rounded-xl bg-surface/60 border border-border space-y-2">
                  <div className="flex justify-between">
                    <span className="text-muted">Type / IP</span>
                    <span className="text-white">{selectedNode.type} ({selectedNode.ip})</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted">Status</span>
                    <span className={selectedNode.status === 'HEALTHY' ? 'text-healthy' : 'text-warning'}>
                      {selectedNode.status}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted">Traffic (In / Out)</span>
                    <span className="text-cyan">{selectedNode.in_mbps} / {selectedNode.out_mbps} MB/s</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted">Avg Latency</span>
                    <span className="text-white">{selectedNode.avg_latency_ms} ms</span>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-[11px] text-muted uppercase">Connected Flows</div>
                  {(data?.topology?.edges || [])
                    .filter((e) => e.source === selectedNode.id || e.target === selectedNode.id)
                    .map((e, i) => (
                      <div key={i} className="p-2.5 rounded-lg bg-[#10131a] border border-border flex justify-between">
                        <span className="text-gray-300">
                          {e.source} → {e.target}
                        </span>
                        <span className="text-cyan">{e.p95_latency_ms.toFixed(1)}ms</span>
                      </div>
                    ))}
                </div>
              </div>
            ) : (
              <div className="space-y-2 text-xs font-mono">
                <p className="text-muted font-sans text-xs">
                  Click any node in the topology canvas to filter its incoming and outgoing network edges.
                </p>
                {(data?.topology?.edges || []).slice(0, 6).map((e, i) => (
                  <div key={i} className="p-2.5 rounded-lg bg-surface/50 border border-border flex items-center justify-between">
                    <span className="text-white truncate max-w-[160px]">
                      {e.source} → {e.target}
                    </span>
                    <span className="text-cyan">{e.bandwidth_mbps.toFixed(1)} MB/s</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* TAB 2: Service-to-Service Latency Heatmap */}
      {tab === 'heatmap' && (
        <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div>
              <h2 className="text-sm font-semibold text-white">Inter-Service P95 Latency Heatmap</h2>
              <p className="text-xs text-muted">
                Pairwise service-to-service network latency matrix (Source Row → Target Column)
              </p>
            </div>
            <div className="flex items-center gap-3 text-[11px] font-mono">
              <span className="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">&lt; 10ms</span>
              <span className="px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-400">10–50ms</span>
              <span className="px-2 py-0.5 rounded bg-amber-500/20 text-amber-400">50–150ms</span>
              <span className="px-2 py-0.5 rounded bg-rose-500/20 text-rose-400">&gt; 150ms</span>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-center text-xs font-mono border-collapse">
              <thead>
                <tr>
                  <th className="p-2 text-left text-muted border-b border-border">Source \ Target</th>
                  {(data?.latency_services || []).map((svc) => (
                    <th key={svc} className="p-2 text-gray-300 border-b border-border whitespace-nowrap">
                      {svc}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {(data?.latency_services || []).map((src) => (
                  <tr key={src} className="border-b border-border/50">
                    <td className="p-2 text-left font-bold text-white whitespace-nowrap">{src}</td>
                    {(data?.latency_services || []).map((dst) => {
                      if (src === dst) {
                        return (
                          <td key={dst} className="p-2 bg-[#0d1016] text-gray-600">
                            self
                          </td>
                        );
                      }
                      const cell = heatmapLookup[`${src}::${dst}`];
                      if (!cell) {
                        return (
                          <td key={dst} className="p-2 text-gray-600">
                            —
                          </td>
                        );
                      }
                      const colorClass =
                        cell.p95_ms >= 150
                          ? 'bg-rose-500/25 text-rose-300 border border-rose-500/30'
                          : cell.p95_ms >= 50
                          ? 'bg-amber-500/25 text-amber-300 border border-amber-500/30'
                          : cell.p95_ms >= 10
                          ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                          : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30';

                      return (
                        <td
                          key={dst}
                          className={`p-2 rounded ${colorClass}`}
                          title={`P50: ${cell.p50_ms.toFixed(1)}ms | P95: ${cell.p95_ms.toFixed(1)}ms | P99: ${cell.p99_ms.toFixed(1)}ms | Err: ${cell.error_rate_pct}%`}
                        >
                          <div className="font-bold">{cell.p95_ms.toFixed(1)}ms</div>
                          <div className="text-[9px] opacity-80">{cell.bandwidth_mbps.toFixed(1)} MB/s</div>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 3: Bandwidth Utilization Trends & Interfaces */}
      {tab === 'bandwidth' && (
        <div className="space-y-6">
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold text-white">Bandwidth Utilization Trends (60m)</h2>
                <p className="text-xs text-muted">Ingress (RX), Egress (TX), and East-West Mesh throughput (MB/s)</p>
              </div>
              <span className="text-xs font-mono text-cyan">5-minute resolution</span>
            </div>

            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data?.bandwidth_trends || []}>
                  <defs>
                    <linearGradient id="rxGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.35} />
                      <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="txGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#6366f1" stopOpacity={0.35} />
                      <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
                  <XAxis dataKey="timestamp" stroke="#64748b" fontSize={11} />
                  <YAxis stroke="#64748b" fontSize={11} unit=" MB/s" />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155', fontSize: '12px' }}
                  />
                  <Area type="monotone" dataKey="rx_mbps" name="RX Ingress" stroke="#06b6d4" fillOpacity={1} fill="url(#rxGrad)" />
                  <Area type="monotone" dataKey="tx_mbps" name="TX Egress" stroke="#6366f1" fillOpacity={1} fill="url(#txGrad)" />
                  <Area type="monotone" dataKey="mesh_mbps" name="East-West Mesh" stroke="#10b981" fillOpacity={0.1} fill="#10b981" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <h2 className="text-sm font-semibold text-white">Physical & Virtual Network Interfaces</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {data?.interfaces?.map((iface) => (
                <div key={iface.name} className="p-4 rounded-xl bg-surface/50 border border-border space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-sm font-bold text-white">{iface.name}</div>
                      <div className="text-xs font-mono text-muted">{iface.ip}</div>
                    </div>
                    <span className="px-2 py-0.5 rounded bg-healthy/20 text-healthy border border-healthy/30 text-[10px] font-mono">
                      {iface.status}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-2 pt-2 border-t border-border/60 font-mono text-xs">
                    <div className="p-2.5 rounded-lg bg-background/50 border border-border">
                      <div className="text-[10px] text-muted">RX (Ingress)</div>
                      <div className="text-sm font-bold text-cyan mt-0.5">{iface.rx_mbps} MB/s</div>
                      <div className="text-[10px] text-muted">{iface.rx_packets_sec?.toLocaleString() ?? '0'} pps</div>
                    </div>

                    <div className="p-2.5 rounded-lg bg-background/50 border border-border">
                      <div className="text-[10px] text-muted">TX (Egress)</div>
                      <div className="text-sm font-bold text-primary-light mt-0.5">{iface.tx_mbps} MB/s</div>
                      <div className="text-[10px] text-muted">{iface.tx_packets_sec?.toLocaleString() ?? '0'} pps</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 4: TCP Sockets & Top Active Flows */}
      {tab === 'sockets' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <h2 className="text-sm font-semibold text-white">TCP Socket State Distribution</h2>
            <div className="space-y-3">
              {data?.socket_states &&
                Object.entries(data.socket_states).map(([st, count]) => {
                  const total = data.tcp_metrics?.total_connections || 1;
                  const pct = Math.round((count / total) * 100);
                  return (
                    <div key={st} className="space-y-1 font-mono text-xs">
                      <div className="flex justify-between text-[11px]">
                        <span className="text-white font-medium">{st}</span>
                        <span className="text-muted">
                          {count.toLocaleString()} sockets ({pct}%)
                        </span>
                      </div>
                      <div className="w-full bg-surface rounded-full h-1.5 overflow-hidden">
                        <div
                          className={`h-full rounded-full ${
                            st === 'ESTABLISHED' ? 'bg-healthy' : st === 'TIME_WAIT' ? 'bg-warning' : 'bg-primary'
                          }`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
            </div>
          </div>

          <div className="glass-panel p-5 rounded-2xl border border-border space-y-4">
            <h2 className="text-sm font-semibold text-white">Top Inter-Service Network Flows</h2>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="text-[11px] font-mono text-muted uppercase border-b border-border bg-surface/50">
                  <tr>
                    <th className="py-2 px-2.5">Flow</th>
                    <th className="py-2 px-2.5">Protocol</th>
                    <th className="py-2 px-2.5">Bandwidth</th>
                    <th className="py-2 px-2.5">Service</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border font-mono">
                  {data?.top_flows?.map((flow, idx) => (
                    <tr key={idx} className="hover:bg-surface/40 transition-colors">
                      <td className="py-2.5 px-2.5 text-white">
                        {flow.src_ip} → {flow.dst_ip}:{flow.port}
                      </td>
                      <td className="py-2.5 px-2.5 text-cyan">{flow.protocol}</td>
                      <td className="py-2.5 px-2.5 text-warning font-bold">{flow.bandwidth_mbps} MB/s</td>
                      <td className="py-2.5 px-2.5 text-muted font-sans">{flow.service}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
