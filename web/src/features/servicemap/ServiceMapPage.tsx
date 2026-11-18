import { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { useServiceMap, useSynthetics, MapNode, MapEdge } from '../../hooks/useServiceMap';
import {
  Layers,
  Activity,
  CheckCircle2,
  AlertTriangle,
  Server,
  ArrowRight,
  Radio,
  Globe,
  Clock,
  ExternalLink,
  ShieldCheck,
  Zap,
  Filter,
  RefreshCw,
  Database,
  Box,
  CornerDownRight,
  XCircle,
} from 'lucide-react';

export function ServiceMapPage() {
  const { data: serviceMap, isLoading: mapLoading, refetch } = useServiceMap();
  const { data: synthetics, isLoading: synLoading } = useSynthetics();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [tierFilter, setTierFilter] = useState<'ALL' | 'INGRESS' | 'SERVICE' | 'DATABASE' | 'CACHE'>('ALL');

  const nodes = useMemo(() => serviceMap?.nodes || [], [serviceMap]);
  const edges = useMemo(() => serviceMap?.edges || [], [serviceMap]);

  // Group nodes by tier
  const tiers = useMemo(() => ({
    ingress: nodes.filter((n) => n.type === 'INGRESS'),
    services: nodes.filter((n) => n.type === 'SERVICE'),
    storage: nodes.filter((n) => n.type === 'DATABASE' || n.type === 'CACHE'),
  }), [nodes]);

  // Compute node positions on the 960x480 SVG coordinate space
  const nodeCoords = useMemo(() => {
    const coords: Record<string, { x: number; y: number; width: number; height: number }> = {};
    const colWidth = 220;
    const colHeight = 84;
    const canvasHeight = 480;

    const layoutColumn = (items: MapNode[], colX: number) => {
      const count = items.length;
      if (count === 0) return;
      const spacing = canvasHeight / (count + 1);
      items.forEach((item, idx) => {
        coords[item.id] = {
          x: colX,
          y: Math.round(spacing * (idx + 1) - colHeight / 2),
          width: colWidth,
          height: colHeight,
        };
      });
    };

    layoutColumn(tiers.ingress, 30);
    layoutColumn(tiers.services, 370);
    layoutColumn(tiers.storage, 710);

    return coords;
  }, [tiers]);

  // Selected node inspection relationships
  const selectedNode = useMemo(
    () => nodes.find((n) => n.id === selectedNodeId) || null,
    [nodes, selectedNodeId]
  );

  const upstreamEdges = useMemo(
    () => (selectedNodeId ? edges.filter((e) => e.target === selectedNodeId) : []),
    [edges, selectedNodeId]
  );

  const downstreamEdges = useMemo(
    () => (selectedNodeId ? edges.filter((e) => e.target !== selectedNodeId && e.source === selectedNodeId) : []),
    [edges, selectedNodeId]
  );

  const isConnected = (nodeId: string) => {
    if (!selectedNodeId) return true;
    if (nodeId === selectedNodeId) return true;
    return (
      upstreamEdges.some((e) => e.source === nodeId) ||
      downstreamEdges.some((e) => e.target === nodeId)
    );
  };

  const isEdgeHighlighted = (edge: MapEdge) => {
    if (!selectedNodeId) return true;
    return edge.source === selectedNodeId || edge.target === selectedNodeId;
  };

  return (
    <div className="max-w-7xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight flex items-center gap-3">
            Service Dependency Graph & Topology
            <span className="text-xs font-mono font-bold px-2.5 py-1 rounded-full bg-primary/20 text-primary-light border border-primary/30">
              Live Topology Canvas
            </span>
          </h1>
          <p className="text-sm text-muted mt-1">
            Interactive microservice mesh topology, directional call flows, latency bottlenecks, and synthetic availability probes.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {selectedNodeId && (
            <button
              onClick={() => setSelectedNodeId(null)}
              className="px-3 py-1.5 rounded-xl bg-surface border border-border text-xs font-mono text-muted hover:text-white transition-colors"
            >
              Reset Focus
            </button>
          )}
          <button
            onClick={() => refetch()}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-surface hover:bg-surface-highlight border border-border text-xs font-mono text-muted hover:text-white transition-colors shadow-sm"
          >
            <RefreshCw size={12} className={mapLoading ? 'animate-spin' : ''} />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Interactive Service Topology Map Canvas */}
      <div className="glass-card rounded-2xl p-6 border border-border shadow-glass space-y-6">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 border-b border-border/80 pb-4">
          <div className="flex items-center gap-2.5">
            <Layers size={18} className="text-primary-light" />
            <div>
              <h2 className="text-base font-bold text-white">Topological Call Flow & Dependency Mesh</h2>
              <p className="text-xs text-muted">Click any node to isolate upstream callers and downstream blast radius.</p>
            </div>
          </div>

          <div className="flex items-center gap-4 text-xs font-mono text-muted">
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-healthy" /> Healthy
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-warning" /> Degraded
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-critical" /> Failing
            </span>
          </div>
        </div>

        {/* Tier Columns Header */}
        <div className="grid grid-cols-3 gap-4 text-[11px] font-mono uppercase tracking-widest text-muted/70 px-4">
          <div className="flex items-center gap-1.5">
            <Globe size={13} className="text-primary-light" />
            <span>Tier 1: Ingress / Edge</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Layers size={13} className="text-primary-light" />
            <span>Tier 2: Microservices</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Database size={13} className="text-primary-light" />
            <span>Tier 3: Storage & Caches</span>
          </div>
        </div>

        {/* SVG Topology Graph Canvas */}
        <div className="relative w-full rounded-2xl bg-[#090b10]/90 border border-border/80 overflow-hidden shadow-inner p-2">
          <svg
            viewBox="0 0 960 480"
            className="w-full h-auto min-h-[420px] select-none"
            preserveAspectRatio="xMidYMid meet"
          >
            <defs>
              <linearGradient id="edgeGradNormal" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="#6366f1" stopOpacity="0.7" />
                <stop offset="100%" stopColor="#06b6d4" stopOpacity="0.7" />
              </linearGradient>
              <linearGradient id="edgeGradSlow" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.9" />
                <stop offset="100%" stopColor="#d97706" stopOpacity="0.9" />
              </linearGradient>
              <linearGradient id="edgeGradError" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="#f43f5e" stopOpacity="1" />
                <stop offset="100%" stopColor="#e11d48" stopOpacity="1" />
              </linearGradient>
            </defs>

            {/* Render Directional SVG Edges */}
            {edges.map((edge, idx) => {
              const srcCoord = nodeCoords[edge.source];
              const tgtCoord = nodeCoords[edge.target];
              if (!srcCoord || !tgtCoord) return null;

              const x1 = srcCoord.x + srcCoord.width;
              const y1 = srcCoord.y + srcCoord.height / 2;
              const x2 = tgtCoord.x;
              const y2 = tgtCoord.y + tgtCoord.height / 2;

              const dx = (x2 - x1) * 0.5;
              const pathD = `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;

              const isHighlighted = isEdgeHighlighted(edge);
              const isError = edge.status === 'ERROR';
              const isSlow = edge.status === 'SLOW';

              let strokeColor = 'url(#edgeGradNormal)';
              let strokeWidth = isHighlighted ? 2.5 : 1.5;
              if (isError) {
                strokeColor = 'url(#edgeGradError)';
                strokeWidth = isHighlighted ? 3 : 2;
              } else if (isSlow) {
                strokeColor = 'url(#edgeGradSlow)';
                strokeWidth = isHighlighted ? 3 : 2;
              }

              // Midpoint for latency badge
              const midX = (x1 + x2) / 2;
              const midY = (y1 + y2) / 2;

              return (
                <g
                  key={idx}
                  className={`transition-opacity duration-200 ${
                    isHighlighted ? 'opacity-100' : 'opacity-20'
                  }`}
                >
                  {/* Outer glow line */}
                  <path
                    d={pathD}
                    fill="none"
                    stroke={strokeColor}
                    strokeWidth={strokeWidth}
                    strokeDasharray={isHighlighted ? '6 4' : undefined}
                    className={isHighlighted ? 'animate-pulse' : ''}
                  />

                  {/* Flowing traffic particle */}
                  {isHighlighted && (
                    <circle r={3} fill="#ffffff">
                      <animateMotion path={pathD} dur={isSlow ? '3s' : '1.5s'} repeatCount="indefinite" />
                    </circle>
                  )}

                  {/* Edge Call Metrics Badge */}
                  <g transform={`translate(${midX - 35}, ${midY - 11})`}>
                    <rect
                      width={70}
                      height={22}
                      rx={6}
                      fill="#111520"
                      stroke={isError ? '#f43f5e' : isSlow ? '#f59e0b' : '#334155'}
                      strokeWidth={1}
                    />
                    <text
                      x={35}
                      y={14}
                      textAnchor="middle"
                      fill={isError ? '#f43f5e' : isSlow ? '#f59e0b' : '#cbd5e1'}
                      fontSize={10}
                      fontFamily="monospace"
                      fontWeight="bold"
                    >
                      {edge.latency_ms}ms
                    </text>
                  </g>
                </g>
              );
            })}

            {/* Render Node Cards on SVG Canvas */}
            {nodes.map((node) => {
              const coord = nodeCoords[node.id];
              if (!coord) return null;

              const isSelected = selectedNodeId === node.id;
              const connected = isConnected(node.id);
              const isDegraded = node.health === 'DEGRADED';
              const isFailing = node.health === 'FAILING';

              let borderColor = '#334155';
              let badgeColor = '#10b981';
              if (isFailing) {
                borderColor = '#f43f5e';
                badgeColor = '#f43f5e';
              } else if (isDegraded) {
                borderColor = '#f59e0b';
                badgeColor = '#f59e0b';
              } else if (isSelected) {
                borderColor = '#6366f1';
              }

              return (
                <g
                  key={node.id}
                  transform={`translate(${coord.x}, ${coord.y})`}
                  onClick={() => setSelectedNodeId(isSelected ? null : node.id)}
                  className={`cursor-pointer transition-all duration-200 select-none ${
                    connected ? 'opacity-100' : 'opacity-25'
                  }`}
                >
                  {/* Card Background */}
                  <rect
                    width={coord.width}
                    height={coord.height}
                    rx={12}
                    fill="#111624"
                    stroke={isSelected ? '#818cf8' : borderColor}
                    strokeWidth={isSelected ? 2.5 : 1.5}
                    className="transition-colors hover:fill-[#171e31]"
                  />

                  {/* Header: Type and Health status */}
                  <rect x={12} y={10} width={62} height={18} rx={4} fill="#1e293b" />
                  <text x={43} y={23} textAnchor="middle" fill="#94a3b8" fontSize={9} fontFamily="monospace" fontWeight="bold">
                    {node.type}
                  </text>

                  {/* Health Dot & Badge */}
                  <circle cx={coord.width - 20} cy={19} r={4.5} fill={badgeColor} className={isFailing || isDegraded ? 'animate-pulse' : ''} />

                  {/* Service Label */}
                  <text x={12} y={46} fill="#ffffff" fontSize={13} fontWeight="bold" fontFamily="sans-serif">
                    {node.label}
                  </text>

                  {/* ID */}
                  <text x={12} y={60} fill="#64748b" fontSize={10} fontFamily="monospace">
                    {node.id}
                  </text>

                  {/* Metrics bar: RPS, Latency, Errors */}
                  <line x1={12} y1={68} x2={coord.width - 12} y2={68} stroke="#1e293b" strokeWidth={1} />
                  <text x={12} y={79} fill="#cbd5e1" fontSize={9.5} fontFamily="monospace">
                    {node.rps} rps · <tspan fill="#38bdf8">{node.latency_ms}ms</tspan> · <tspan fill={node.error_pct > 1 ? '#f43f5e' : '#10b981'}>{node.error_pct}% err</tspan>
                  </text>
                </g>
              );
            })}
          </svg>
        </div>

        {/* Selected Node Blast Radius & Upstream/Downstream Inspector */}
        {selectedNode && (
          <div className="p-5 rounded-2xl bg-surface border border-primary/40 shadow-xl space-y-4 animate-in fade-in duration-150">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border/80 pb-3">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-xl bg-primary/20 border border-primary/30 text-primary-light">
                  <Activity size={18} />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-base font-bold text-white">{selectedNode.label}</h3>
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-surface-highlight border border-border text-primary-light">
                      {selectedNode.type}
                    </span>
                    <span
                      className={`text-[10px] font-mono font-bold px-2 py-0.5 rounded-full border ${
                        selectedNode.health === 'HEALTHY'
                          ? 'bg-healthy/15 text-healthy border-healthy/30'
                          : 'bg-warning/20 text-warning border-warning/30 animate-pulse'
                      }`}
                    >
                      {selectedNode.health}
                    </span>
                  </div>
                  <div className="text-xs font-mono text-muted mt-0.5">Telemetry node ID: {selectedNode.id}</div>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Link
                  to={`/logs?search=${selectedNode.id}`}
                  className="px-3 py-1.5 rounded-xl bg-surface-highlight hover:bg-surface-elevated border border-border text-xs font-mono text-white transition-colors"
                >
                  Inspect Live Logs
                </Link>
                <Link
                  to={`/traces?service=${selectedNode.id}`}
                  className="px-3 py-1.5 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary transition-all"
                >
                  View Waterfall Traces
                </Link>
              </div>
            </div>

            {/* Dependency Blast Radius Breakdown */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Upstream Callers */}
              <div className="p-3.5 rounded-xl bg-background/70 border border-border space-y-2">
                <div className="text-xs font-mono font-semibold text-muted uppercase tracking-wider flex items-center justify-between">
                  <span>Upstream Callers ({upstreamEdges.length})</span>
                  <span className="text-[10px] text-primary-light">Inbound Traffic</span>
                </div>
                {upstreamEdges.length === 0 ? (
                  <div className="text-xs text-muted/70 italic py-2">No upstream internal callers (Edge entrypoint).</div>
                ) : (
                  <div className="space-y-1.5">
                    {upstreamEdges.map((e, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between p-2 rounded-lg bg-surface/80 border border-border text-xs font-mono"
                      >
                        <span className="font-semibold text-white">{e.source}</span>
                        <div className="flex items-center gap-2 text-muted">
                          <span>{e.call_rate_rps} rps</span>
                          <span className="text-primary-light font-bold">{e.latency_ms} ms</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Downstream Dependencies */}
              <div className="p-3.5 rounded-xl bg-background/70 border border-border space-y-2">
                <div className="text-xs font-mono font-semibold text-muted uppercase tracking-wider flex items-center justify-between">
                  <span>Downstream Dependencies ({downstreamEdges.length})</span>
                  <span className="text-[10px] text-warning">Blast Radius Risk</span>
                </div>
                {downstreamEdges.length === 0 ? (
                  <div className="text-xs text-muted/70 italic py-2">No downstream internal dependencies (Leaf node).</div>
                ) : (
                  <div className="space-y-1.5">
                    {downstreamEdges.map((e, idx) => (
                      <div
                        key={idx}
                        className={`flex items-center justify-between p-2 rounded-lg border text-xs font-mono ${
                          e.status === 'ERROR'
                            ? 'bg-critical/10 border-critical/40 text-critical'
                            : e.status === 'SLOW'
                            ? 'bg-warning/10 border-warning/40 text-warning'
                            : 'bg-surface/80 border-border text-white'
                        }`}
                      >
                        <span className="font-semibold">{e.target}</span>
                        <div className="flex items-center gap-2">
                          <span className="opacity-80">{e.call_rate_rps} rps</span>
                          <span className="font-bold">{e.latency_ms} ms</span>
                          {e.status !== 'OK' && (
                            <span className="text-[10px] font-bold px-1.5 py-0.2 rounded bg-black/40 uppercase">
                              {e.status}
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Live Call Flows Table */}
        <div className="p-4 rounded-xl bg-surface/50 border border-border space-y-3">
          <div className="text-xs font-mono text-muted uppercase tracking-wider">
            All Inter-Service Link Latency & Throughput Metrics
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {edges.map((edge, idx) => (
              <div
                key={idx}
                className="p-2.5 rounded-lg bg-background/60 border border-border flex items-center justify-between text-xs font-mono hover:border-primary/40 transition-colors"
              >
                <div className="flex items-center gap-1.5 truncate">
                  <span className="text-white font-bold">{edge.source}</span>
                  <ArrowRight size={12} className="text-primary-light shrink-0" />
                  <span className="text-text-dim truncate">{edge.target}</span>
                </div>
                <div className="text-right shrink-0 ml-2">
                  <div className={`font-bold ${edge.status === 'ERROR' ? 'text-critical' : edge.status === 'SLOW' ? 'text-warning' : 'text-white'}`}>
                    {edge.latency_ms} ms
                  </div>
                  <div className="text-[10px] text-muted">{edge.call_rate_rps} rps</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* External Synthetic Monitoring Section */}
      <div className="glass-card rounded-2xl p-6 border border-border shadow-glass space-y-6">
        <div className="flex items-center justify-between border-b border-border/80 pb-3">
          <div className="flex items-center gap-2">
            <Globe size={18} className="text-primary-light" />
            <h2 className="text-base font-bold text-white">External Synthetic Availability Probes</h2>
          </div>
          <div className="text-xs font-mono text-muted">Global SRE Healthchecks</div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          {(synthetics || []).map((syn) => {
            const isPassing = syn.status === 'PASSING';

            return (
              <div
                key={syn.id}
                className="glass-card rounded-2xl p-5 border border-border flex flex-col justify-between space-y-4 shadow-sm"
              >
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-surface border border-border text-primary-light">
                      {syn.type} PROBE ({syn.interval_sec}s)
                    </span>
                    <span
                      className={`inline-flex items-center gap-1 text-[11px] font-mono font-bold px-2 py-0.5 rounded-full border ${
                        isPassing
                          ? 'bg-healthy/15 text-healthy border-healthy/30'
                          : 'bg-critical/15 text-critical border-critical/30'
                      }`}
                    >
                      <CheckCircle2 size={11} />
                      {syn.status}
                    </span>
                  </div>

                  <h3 className="text-sm font-bold text-white">{syn.name}</h3>
                  <div className="text-[11px] font-mono text-muted truncate">{syn.target_url}</div>

                  <div className="p-3 rounded-xl bg-background/60 border border-border/70 grid grid-cols-4 gap-1 text-center text-xs font-mono">
                    <div>
                      <div className="text-[9px] text-muted uppercase">DNS</div>
                      <div className="font-bold text-white mt-0.5">{syn.dns_ms}ms</div>
                    </div>
                    <div>
                      <div className="text-[9px] text-muted uppercase">TLS</div>
                      <div className="font-bold text-white mt-0.5">{syn.tls_ms}ms</div>
                    </div>
                    <div>
                      <div className="text-[9px] text-muted uppercase">TTFB</div>
                      <div className="font-bold text-white mt-0.5">{syn.ttfb_ms}ms</div>
                    </div>
                    <div>
                      <div className="text-[9px] text-muted uppercase">Uptime</div>
                      <div className="font-bold text-healthy mt-0.5">{syn.availability}%</div>
                    </div>
                  </div>
                </div>

                <div className="pt-2 border-t border-border/60 flex items-center justify-between text-[11px] font-mono text-muted">
                  <span>Latency: <strong className="text-white">{syn.latency_ms} ms</strong></span>
                  <span>Checked: {new Date(syn.last_check_at).toLocaleTimeString()}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
