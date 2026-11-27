import React, { useState, useEffect } from 'react';
import { api } from '../../api/client';
import { 
  Network, Cpu, HardDrive, RefreshCw, Activity, Layers, 
  CheckCircle2, AlertTriangle, XCircle, Clock, Server, Shield,
  Terminal, Copy, Check, Search, Filter, Box, AlertOctagon,
  ExternalLink, ArrowUpRight
} from 'lucide-react';
import { MetricCard, LoadingState } from '../../components/ui/Primitives';

interface K8sOverview {
  cluster_name: string;
  version: string;
  status: string;
  total_nodes: number;
  ready_nodes: number;
  total_pods: number;
  running_pods: number;
  pending_pods: number;
  failed_pods: number;
  total_cpu_milli: number;
  allocated_cpu_milli: number;
  total_memory_bytes: number;
  allocated_memory_bytes: number;
  last_sync_at: string;
  is_live_collector?: boolean;
}

interface K8sNode {
  name: string;
  role: string;
  status: string;
  internal_ip: string;
  os_image: string;
  kubelet_version: string;
  cpu_usage_pct: number;
  memory_usage_pct: number;
  memory_pressure: boolean;
  disk_pressure: boolean;
  pid_pressure: boolean;
  pod_count: number;
  pod_capacity: number;
  labels: Record<string, string>;
}

interface K8sDeployment {
  name: string;
  namespace: string;
  desired_replicas: number;
  available_replicas: number;
  updated_replicas: number;
  image: string;
  status: string;
}

interface K8sPod {
  name: string;
  namespace: string;
  node: string;
  phase: string;
  restart_count: number;
  oom_killed: boolean;
  ip: string;
  image: string;
  cpu_usage_pct: number;
  cpu_limit_milli?: number;
  mem_usage_mb: number;
  mem_limit_mb?: number;
  status_reason?: string;
  created_at: string;
}

interface K8sClusterEvent {
  id: string;
  type: string;
  reason: string;
  message: string;
  object: string;
  namespace: string;
  timestamp: string;
}

export const KubernetesPage: React.FC = () => {
  const [overview, setOverview] = useState<K8sOverview | null>(null);
  const [nodes, setNodes] = useState<K8sNode[]>([]);
  const [deployments, setDeployments] = useState<K8sDeployment[]>([]);
  const [pods, setPods] = useState<K8sPod[]>([]);
  const [events, setEvents] = useState<K8sClusterEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'nodes' | 'workloads' | 'pods' | 'events'>('nodes');

  const [search, setSearch] = useState('');
  const [namespaceFilter, setNamespaceFilter] = useState('ALL');
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [copiedDeployCmd, setCopiedDeployCmd] = useState(false);

  const fetchK8sData = async () => {
    try {
      setLoading(true);
      const [ovRes, nodesRes, workloadsRes] = await Promise.all([
        api.get('/kubernetes/overview'),
        api.get('/kubernetes/nodes'),
        api.get('/kubernetes/workloads'),
      ]);
      setOverview(ovRes.data);
      setNodes(nodesRes.data);
      setDeployments(workloadsRes.data.deployments || []);
      setPods(workloadsRes.data.pods || []);
      setEvents(workloadsRes.data.events || []);
    } catch (err) {
      console.error('Failed to load Kubernetes data', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchK8sData();
  }, []);

  const formatGB = (bytes: number) => {
    return (bytes / (1024 * 1024 * 1024)).toFixed(0) + ' GB';
  };

  // Collect unique namespaces
  const namespaces = Array.from(new Set([
    ...deployments.map(d => d.namespace),
    ...pods.map(p => p.namespace),
  ])).filter(Boolean);

  // Filter pods
  const filteredPods = pods.filter(p => {
    const matchesNs = namespaceFilter === 'ALL' || p.namespace === namespaceFilter;
    const matchesSearch = !search || 
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.image.toLowerCase().includes(search.toLowerCase()) ||
      p.node.toLowerCase().includes(search.toLowerCase());
    return matchesNs && matchesSearch;
  });

  // Filter deployments
  const filteredDeployments = deployments.filter(d => {
    const matchesNs = namespaceFilter === 'ALL' || d.namespace === namespaceFilter;
    const matchesSearch = !search || 
      d.name.toLowerCase().includes(search.toLowerCase()) ||
      d.image.toLowerCase().includes(search.toLowerCase());
    return matchesNs && matchesSearch;
  });

  // Total pod restarts & OOM kills
  const totalRestarts = pods.reduce((acc, p) => acc + (p.restart_count || 0), 0);
  const totalOOM = pods.filter(p => p.oom_killed).length;

  const deployCommand = `kubectl apply -f http://169.58.138.136:8080/deploy/sentrix-daemonset.yaml`;

  const copyDeploy = () => {
    navigator.clipboard.writeText(deployCommand);
    setCopiedDeployCmd(true);
    setTimeout(() => setCopiedDeployCmd(false), 2000);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight text-white">Kubernetes & Container Fleet</h1>
            <span className="px-2.5 py-0.5 text-xs font-mono rounded-full bg-blue-500/20 text-blue-400 border border-blue-500/30">
              K8s EKS / Vanilla
            </span>
            {overview?.is_live_collector ? (
              <span className="flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                Live DaemonSet Active
              </span>
            ) : (
              <span className="flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono bg-white/10 text-white border border-white/25">
                <span className="w-2 h-2 rounded-full bg-white" />
                Cluster Fleet Telemetry
              </span>
            )}
          </div>
          <p className="text-sm text-gray-400 mt-1">
            Hierarchical container visibility: node pressure, cAdvisor pod resource quotas, restarts, OOM kills, and scheduling events.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowDeployModal(true)}
            className="flex items-center gap-2 px-3.5 py-2 bg-primary hover:bg-primary-hover text-white text-xs font-semibold rounded-xl shadow-glow-primary transition-all"
          >
            <Terminal className="w-4 h-4" />
            <span>Deploy K8s Collector</span>
          </button>
          <button
            onClick={fetchK8sData}
            className="flex items-center gap-2 px-3 py-2 bg-[#1b1e24] hover:bg-[#252932] border border-[#2e3440] text-gray-300 text-xs font-medium rounded-xl transition-colors"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            <span>Sync</span>
          </button>
        </div>
      </div>

      {/* Cluster Overview KPIs */}
      {overview && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Cluster State"
            value={overview.status}
            subtitle={`${overview.cluster_name} (${overview.version})`}
            icon={<Shield className="w-5 h-5 text-emerald-400" />}
          />
          <MetricCard
            title="Ready Nodes"
            value={`${overview.ready_nodes} / ${overview.total_nodes}`}
            subtitle="100% Schedulable"
            icon={<Server className="w-5 h-5 text-blue-400" />}
          />
          <MetricCard
            title="Active Pods"
            value={`${overview.running_pods} / ${overview.total_pods}`}
            subtitle={`${overview.pending_pods} Pending, ${overview.failed_pods} Failed`}
            icon={<Layers className="w-5 h-5 text-purple-400" />}
          />
          <MetricCard
            title="Resource Allocation"
            value={`${(overview.allocated_cpu_milli / 1000).toFixed(0)} / ${(overview.total_cpu_milli / 1000).toFixed(0)} Cores`}
            subtitle={`RAM: ${formatGB(overview.allocated_memory_bytes)} / ${formatGB(overview.total_memory_bytes)}`}
            icon={<Cpu className="w-5 h-5 text-cyan-400" />}
          />
        </div>
      )}

      {/* Filter & Search Bar */}
      <div className="flex flex-col sm:flex-row gap-3 items-center justify-between bg-[#13161b] p-3 rounded-xl border border-[#232730]">
        <div className="flex items-center gap-2 overflow-x-auto w-full sm:w-auto">
          <span className="text-xs font-mono text-muted mr-1">Namespace:</span>
          {['ALL', ...namespaces].map((ns) => (
            <button
              key={ns}
              onClick={() => setNamespaceFilter(ns)}
              className={`px-2.5 py-1 rounded-lg text-xs font-mono transition-colors shrink-0 ${
                namespaceFilter === ns
                  ? 'bg-blue-600 text-white font-semibold'
                  : 'bg-[#181c23] text-gray-400 hover:text-white border border-[#2e3440]'
              }`}
            >
              {ns}
            </button>
          ))}
        </div>

        <div className="relative w-full sm:w-72">
          <Search className="w-3.5 h-3.5 text-gray-500 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search pods, nodes, or images..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[#181c23] border border-[#2e3440] rounded-lg pl-9 pr-3 py-1.5 text-xs text-white focus:outline-none focus:border-blue-500 transition-colors"
          />
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-2 border-b border-[#232730] pb-2">
        <button
          onClick={() => setActiveTab('nodes')}
          className={`px-4 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            activeTab === 'nodes'
              ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30'
              : 'text-gray-400 hover:text-white hover:bg-[#181c23]'
          }`}
        >
          <Server className="w-3.5 h-3.5" />
          <span>Nodes ({nodes.length})</span>
        </button>
        <button
          onClick={() => setActiveTab('workloads')}
          className={`px-4 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            activeTab === 'workloads'
              ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30'
              : 'text-gray-400 hover:text-white hover:bg-[#181c23]'
          }`}
        >
          <Box className="w-3.5 h-3.5" />
          <span>Deployments ({filteredDeployments.length})</span>
        </button>
        <button
          onClick={() => setActiveTab('pods')}
          className={`px-4 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            activeTab === 'pods'
              ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30'
              : 'text-gray-400 hover:text-white hover:bg-[#181c23]'
          }`}
        >
          <Layers className="w-3.5 h-3.5" />
          <span>Pods ({filteredPods.length})</span>
          {totalRestarts > 0 && (
            <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-amber-500/20 text-amber-400 font-mono">
              {totalRestarts} restarts
            </span>
          )}
        </button>
        <button
          onClick={() => setActiveTab('events')}
          className={`px-4 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 ${
            activeTab === 'events'
              ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30'
              : 'text-gray-400 hover:text-white hover:bg-[#181c23]'
          }`}
        >
          <Activity className="w-3.5 h-3.5" />
          <span>Cluster Events ({events.length})</span>
        </button>
      </div>

      {loading ? (
        <LoadingState message="Querying Kubernetes cluster telemetry..." />
      ) : (
        <div className="bg-[#13161b] border border-[#232730] rounded-xl overflow-hidden shadow-xl">
          {/* TAB 1: NODES */}
          {activeTab === 'nodes' && (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-gray-300">
                <thead className="bg-[#171a21] text-xs uppercase tracking-wider text-gray-400 border-b border-[#232730]">
                  <tr>
                    <th className="px-6 py-3 font-medium">Node Name & Role</th>
                    <th className="px-6 py-3 font-medium">Status</th>
                    <th className="px-6 py-3 font-medium">CPU Saturation</th>
                    <th className="px-6 py-3 font-medium">Memory Saturation</th>
                    <th className="px-6 py-3 font-medium">Node Pressure Conditions</th>
                    <th className="px-6 py-3 font-medium">Pod Capacity</th>
                    <th className="px-6 py-3 font-medium">Kubelet</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#1e222a]">
                  {nodes.map((n) => {
                    const hasPressure = n.memory_pressure || n.disk_pressure || n.pid_pressure;
                    return (
                      <tr key={n.name} className="hover:bg-[#181c23] transition-colors">
                        <td className="px-6 py-4">
                          <div className="font-mono text-sm font-medium text-white">{n.name}</div>
                          <div className="text-xs text-gray-400">{n.role} • {n.internal_ip}</div>
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            {n.status}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="text-xs font-mono font-medium text-white mb-1">{n.cpu_usage_pct.toFixed(1)}%</div>
                          <div className="w-24 bg-[#1e222a] rounded-full h-1.5 overflow-hidden">
                            <div className="bg-white h-full rounded-full" style={{ width: `${n.cpu_usage_pct}%` }} />
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="text-xs font-mono font-medium text-white mb-1">{n.memory_usage_pct.toFixed(1)}%</div>
                          <div className="w-24 bg-[#1e222a] rounded-full h-1.5 overflow-hidden">
                            <div className="bg-emerald-500 h-full rounded-full" style={{ width: `${n.memory_usage_pct}%` }} />
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1.5 text-xs font-mono">
                            {!hasPressure ? (
                              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                                <CheckCircle2 className="w-3 h-3" />
                                Healthy
                              </span>
                            ) : (
                              <div className="flex items-center gap-1">
                                {n.memory_pressure && (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-rose-500/20 text-rose-400 border border-rose-500/30">
                                    <AlertTriangle className="w-3 h-3" /> MemPressure
                                  </span>
                                )}
                                {n.disk_pressure && (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
                                    <AlertTriangle className="w-3 h-3" /> DiskPressure
                                  </span>
                                )}
                              </div>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs">
                          <div className="text-gray-300">{n.pod_count} / {n.pod_capacity} pods</div>
                          <div className="w-20 bg-[#1e222a] rounded-full h-1 overflow-hidden mt-1">
                            <div
                              className="bg-blue-400 h-full rounded-full"
                              style={{ width: `${Math.min(100, (n.pod_count / n.pod_capacity) * 100)}%` }}
                            />
                          </div>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-blue-400">
                          {n.kubelet_version}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* TAB 2: DEPLOYMENTS */}
          {activeTab === 'workloads' && (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-gray-300">
                <thead className="bg-[#171a21] text-xs uppercase tracking-wider text-gray-400 border-b border-[#232730]">
                  <tr>
                    <th className="px-6 py-3 font-medium">Deployment Name</th>
                    <th className="px-6 py-3 font-medium">Namespace</th>
                    <th className="px-6 py-3 font-medium">Replicas (Ready / Desired)</th>
                    <th className="px-6 py-3 font-medium">Container Image</th>
                    <th className="px-6 py-3 font-medium">Health Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#1e222a]">
                  {filteredDeployments.map((d) => (
                    <tr key={d.name} className="hover:bg-[#181c23] transition-colors">
                      <td className="px-6 py-4 font-medium text-white">{d.name}</td>
                      <td className="px-6 py-4">
                        <span className="px-2 py-0.5 rounded text-xs font-mono bg-purple-500/10 text-purple-400 border border-purple-500/20">
                          {d.namespace}
                        </span>
                      </td>
                      <td className="px-6 py-4 font-mono text-xs">
                        <span className={d.available_replicas === d.desired_replicas ? 'text-emerald-400 font-bold' : 'text-amber-400 font-bold'}>
                          {d.available_replicas}
                        </span> / {d.desired_replicas} Ready
                      </td>
                      <td className="px-6 py-4 font-mono text-xs text-gray-400">{d.image}</td>
                      <td className="px-6 py-4">
                        <span className={`px-2.5 py-1 rounded-full text-xs font-semibold ${
                          d.status === 'HEALTHY'
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                        }`}>
                          {d.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* TAB 3: PODS */}
          {activeTab === 'pods' && (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-gray-300">
                <thead className="bg-[#171a21] text-xs uppercase tracking-wider text-gray-400 border-b border-[#232730]">
                  <tr>
                    <th className="px-6 py-3 font-medium">Pod Name & Container Image</th>
                    <th className="px-6 py-3 font-medium">Namespace</th>
                    <th className="px-6 py-3 font-medium">Assigned Node</th>
                    <th className="px-6 py-3 font-medium">Phase</th>
                    <th className="px-6 py-3 font-medium">Restarts & OOM</th>
                    <th className="px-6 py-3 font-medium">CPU (Usage vs Quota)</th>
                    <th className="px-6 py-3 font-medium">Memory (Usage vs Limit)</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#1e222a]">
                  {filteredPods.map((p) => {
                    const isPending = p.phase === 'Pending';
                    const isFailed = p.phase === 'Failed' || p.phase === 'CrashLoopBackOff';
                    const isRunning = p.phase === 'Running';
                    const memPct = p.mem_limit_mb ? Math.min(100, (p.mem_usage_mb / p.mem_limit_mb) * 100) : 50;

                    return (
                      <tr key={p.name} className="hover:bg-[#181c23] transition-colors">
                        <td className="px-6 py-4">
                          <div className="font-mono text-xs font-medium text-white">{p.name}</div>
                          <div className="text-[11px] font-mono text-gray-400 truncate max-w-xs">{p.image}</div>
                          {p.status_reason && (
                            <div className="text-[10px] text-amber-400 font-mono mt-0.5 flex items-center gap-1">
                              <AlertTriangle className="w-3 h-3" />
                              <span>{p.status_reason}</span>
                            </div>
                          )}
                        </td>
                        <td className="px-6 py-4">
                          <span className="px-2 py-0.5 rounded text-xs font-mono bg-purple-500/10 text-purple-400">
                            {p.namespace}
                          </span>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-gray-400">
                          {p.node || 'Pending Scheduler'}
                        </td>
                        <td className="px-6 py-4">
                          <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                            isRunning
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : isPending
                              ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                              : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                          }`}>
                            {isRunning ? <CheckCircle2 className="w-3 h-3" /> : <Clock className="w-3 h-3" />}
                            {p.phase}
                          </span>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs">
                          <div className="flex items-center gap-1.5">
                            <span className={p.restart_count > 0 ? 'text-amber-400 font-bold' : 'text-gray-300'}>
                              {p.restart_count} restarts
                            </span>
                            {p.oom_killed && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] bg-rose-500/20 text-rose-400 font-bold border border-rose-500/30">
                                OOMKilled
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs">
                          <div className="text-white">{p.cpu_usage_pct.toFixed(1)}%</div>
                          <div className="text-[10px] text-gray-500">
                            {p.cpu_limit_milli ? `limit: ${p.cpu_limit_milli}m` : 'no limit'}
                          </div>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs">
                          <div className="flex items-center justify-between text-gray-300 mb-1">
                            <span>{p.mem_usage_mb} MB</span>
                            <span className="text-[10px] text-gray-500">/ {p.mem_limit_mb || 512} MB</span>
                          </div>
                          <div className="w-24 bg-[#1e222a] rounded-full h-1.5 overflow-hidden">
                            <div
                              className={`h-full rounded-full ${memPct > 85 ? 'bg-rose-500' : memPct > 65 ? 'bg-amber-500' : 'bg-emerald-500'}`}
                              style={{ width: `${memPct}%` }}
                            />
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* TAB 4: EVENTS */}
          {activeTab === 'events' && (
            <div className="divide-y divide-[#1e222a]">
              {events.map((ev) => (
                <div key={ev.id} className="p-4 flex items-start gap-3 hover:bg-[#181c23] transition-colors">
                  {ev.type === 'Warning' ? (
                    <AlertTriangle className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
                  ) : (
                    <CheckCircle2 className="w-5 h-5 text-blue-400 flex-shrink-0 mt-0.5" />
                  )}
                  <div className="flex-1 space-y-1">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-white">{ev.reason}</span>
                        <span className="font-mono text-xs text-purple-400">{ev.object}</span>
                        <span className="px-1.5 py-0.2 rounded bg-[#1e222a] text-[10px] text-gray-400 font-mono">
                          {ev.namespace}
                        </span>
                      </div>
                      <span className="text-xs text-gray-500">{new Date(ev.timestamp).toLocaleTimeString()}</span>
                    </div>
                    <p className="text-xs text-gray-300">{ev.message}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Deploy K8s Collector Modal */}
      {showDeployModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="glass-card rounded-2xl p-6 border border-primary/40 shadow-glow-primary max-w-xl w-full space-y-5">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <Terminal className="w-5 h-5 text-primary-light" />
                <span>Deploy SentriX Kubernetes Collector</span>
              </h2>
              <button
                onClick={() => setShowDeployModal(false)}
                className="text-gray-400 hover:text-white text-sm"
              >
                ✕
              </button>
            </div>

            <p className="text-xs text-muted leading-relaxed">
              Run this 1-line command in your cluster to deploy the SentriX Collector DaemonSet. It automatically discovers pods, nodes, restarts, OOM events, and cAdvisor resource utilization, streaming telemetry securely to this SentriX server.
            </p>

            <div className="space-y-2">
              <label className="text-xs font-mono text-gray-400">Kubernetes One-Liner Manifest</label>
              <div className="flex items-center justify-between p-3 rounded-xl bg-[#0e1116] border border-[#232730] font-mono text-xs text-gray-200">
                <code className="truncate mr-2">{deployCommand}</code>
                <button
                  onClick={copyDeploy}
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-primary hover:bg-primary-hover text-white text-xs font-semibold shrink-0"
                >
                  {copiedDeployCmd ? <Check size={13} /> : <Copy size={13} />}
                  <span>{copiedDeployCmd ? 'Copied!' : 'Copy'}</span>
                </button>
              </div>
            </div>

            <div className="p-3 rounded-xl bg-[#14171e] border border-border/70 space-y-2 text-xs text-gray-300 font-mono">
              <div className="text-[11px] text-primary-light font-bold">What this DaemonSet provides:</div>
              <ul className="list-disc list-inside space-y-1 text-[11px] text-gray-400">
                <li>Non-invasive read-only cluster role for node & pod metrics</li>
                <li>Zero binary footprint: uses native kubelet summary API</li>
                <li>Automatic detection of CrashLoopBackOff, OOMKilled, and scheduling backpressure</li>
                <li>Real-time sync interval: 15 seconds</li>
              </ul>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setShowDeployModal(false)}
                className="px-4 py-2 rounded-xl bg-surface hover:bg-[#20242c] text-white text-xs font-semibold border border-border"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
