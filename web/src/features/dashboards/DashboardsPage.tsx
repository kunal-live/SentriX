import { FormEvent, useState } from 'react';
import { useDashboards, useCreateDashboard, Dashboard, DashboardWidget } from '../../hooks/useDashboards';
import { useServers } from '../../hooks/useDashboard';
import { useServerMetrics } from '../../hooks/useMetrics';
import { useCentralLogs } from '../../hooks/useCentralLogs';
import { useIncidents } from '../../hooks/useAlerts';
import {
  LayoutDashboard,
  Plus,
  Activity,
  AlertTriangle,
  FileText,
  Clock,
  Sparkles,
  Layers,
  BarChart2,
  Trash2,
  TrendingUp,
  Sliders,
  CheckCircle2,
  Server as ServerIcon,
} from 'lucide-react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';

export function DashboardsPage() {
  const { data: dashboards, isLoading } = useDashboards();
  const createDashboard = useCreateDashboard();

  const { data: servers = [] } = useServers();
  const [selectedServerId, setSelectedServerId] = useState<string>('');
  const activeServerId = selectedServerId || (servers.length > 0 ? servers[0].id : undefined);
  const activeServer = servers.find((s) => s.id === activeServerId) || servers[0];

  const { data: cpuMetrics } = useServerMetrics(activeServerId, 'cpu', '1h');
  const { data: diskMetrics } = useServerMetrics(activeServerId, 'disk', '1h');
  const { data: logsData } = useCentralLogs({ limit: 12 }, true);
  const { data: openIncidents } = useIncidents('open');

  const [activeDashboardId, setActiveDashboardId] = useState<string>('dash-primary');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAddWidgetModal, setShowAddWidgetModal] = useState(false);

  const [dashTitle, setDashTitle] = useState('');
  const [dashDescription, setDashDescription] = useState('');

  // Add widget form
  const [widgetTitle, setWidgetTitle] = useState('');
  const [widgetType, setWidgetType] = useState<DashboardWidget['type']>('METRIC_LINE');
  const [widgetMetric, setWidgetMetric] = useState('system.cpu.utilization');
  const [widgetColSpan, setWidgetColSpan] = useState(2);

  const activeDashboard =
    (dashboards || []).find((d) => d.id === activeDashboardId) ||
    (dashboards && dashboards[0]);

  async function handleCreateDashboard(e: FormEvent) {
    e.preventDefault();
    await createDashboard.mutateAsync({
      title: dashTitle,
      description: dashDescription,
      is_default: false,
      widgets: [
        {
          id: 'w-' + Date.now(),
          type: 'METRIC_LINE',
          title: 'Fleet CPU Saturation',
          metric: 'system.cpu.utilization',
          col_span: 2,
        },
      ],
    });
    setDashTitle('');
    setDashDescription('');
    setShowCreateModal(false);
  }

  function handleAddWidget(e: FormEvent) {
    e.preventDefault();
    if (!activeDashboard) return;
    activeDashboard.widgets.push({
      id: 'w-' + Date.now(),
      type: widgetType,
      title: widgetTitle || 'Custom Widget',
      metric: widgetMetric,
      col_span: widgetColSpan,
    });
    setWidgetTitle('');
    setShowAddWidgetModal(false);
  }

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {/* Header & Dashboard Picker */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight flex items-center gap-3">
            Observability Workspaces
            <span className="text-xs font-mono font-bold px-2.5 py-1 rounded-full bg-primary/20 text-primary-light border border-primary/30">
              Custom Dashboards
            </span>
          </h1>
          <p className="text-sm text-muted mt-1">
            Customizable operational dashboards, multi-metric widgets, and unified monitoring canvases.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {servers.length > 0 && (
            <div className="flex items-center gap-1.5 bg-surface border border-border px-3 py-1.5 rounded-xl text-xs font-mono">
              <ServerIcon size={14} className="text-primary-light shrink-0" />
              <select
                value={activeServerId || ''}
                onChange={(e) => setSelectedServerId(e.target.value)}
                className="bg-transparent text-white focus:outline-none cursor-pointer text-xs"
              >
                {servers.map((s) => (
                  <option key={s.id} value={s.id} className="bg-surface text-white">
                    {s.name || s.hostname} ({s.status})
                  </option>
                ))}
              </select>
            </div>
          )}

          <select
            value={activeDashboard?.id || ''}
            onChange={(e) => setActiveDashboardId(e.target.value)}
            className="bg-surface border border-border focus:border-primary rounded-xl px-3.5 py-2 text-xs text-white font-mono focus:outline-none"
          >
            {(dashboards || []).map((d) => (
              <option key={d.id} value={d.id}>
                {d.title}
              </option>
            ))}
          </select>

          <button
            onClick={() => setShowAddWidgetModal(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-surface hover:bg-surface-elevated text-xs font-semibold text-text border border-border transition-colors"
          >
            <Plus size={14} className="text-primary-light" />
            <span>Add Widget</span>
          </button>

          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary transition-all"
          >
            <Plus size={14} />
            <span>New Dashboard</span>
          </button>
        </div>
      </div>

      {/* Active Dashboard Meta Banner */}
      {activeDashboard && (
        <div className="glass-card rounded-2xl p-4 border border-border flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs font-mono">
          <div>
            <span className="text-white font-bold text-sm">{activeDashboard.title}</span>
            <p className="text-muted text-[11px] mt-0.5">{activeDashboard.description}</p>
          </div>
          <div className="flex items-center gap-2 text-text-dim text-[11px]">
            <span>{activeDashboard.widgets.length} Widgets Active</span>
            <span>•</span>
            <span>Auto-refresh: 10s</span>
          </div>
        </div>
      )}

      {/* Dynamic Widget Grid */}
      {activeDashboard && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          {activeDashboard.widgets.map((widget) => {
            const isSpan2 = widget.col_span === 2;
            const isSpan3 = widget.col_span === 3;

            return (
              <div
                key={widget.id}
                className={`glass-card rounded-2xl p-5 border border-border flex flex-col justify-between space-y-4 ${
                  isSpan3
                    ? 'md:col-span-3'
                    : isSpan2
                    ? 'md:col-span-2'
                    : 'md:col-span-1'
                }`}
              >
                <div className="flex items-center justify-between border-b border-border/60 pb-2.5">
                  <div className="flex items-center gap-2 text-xs font-bold text-white">
                    {widget.type === 'METRIC_LINE' && <Activity size={14} className="text-primary-light" />}
                    {widget.type === 'METRIC_GAUGE' && <TrendingUp size={14} className="text-warning" />}
                    {widget.type === 'LOG_STREAM' && <FileText size={14} className="text-cyan" />}
                    {widget.type === 'ALERT_LIST' && <AlertTriangle size={14} className="text-critical" />}
                    <span>{widget.title}</span>
                  </div>
                  <span className="text-[10px] font-mono text-muted uppercase">{widget.type}</span>
                </div>

                {/* Widget Body Content */}
                <div className="py-2">
                  {widget.type === 'METRIC_LINE' && (() => {
                    const isDisk = widget.metric?.includes('disk');
                    const realPoints = isDisk ? diskMetrics?.points : cpuMetrics?.points;
                    const chartData = (realPoints && realPoints.length > 0)
                      ? realPoints.map((p) => ({
                          time: new Date(p.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
                          value: Math.round(p.value * 10) / 10,
                        }))
                      : [];

                    if (chartData.length === 0) {
                      return (
                        <div className="h-56 w-full flex flex-col items-center justify-center text-muted text-xs font-mono border border-dashed border-border/60 rounded-xl space-y-2">
                          <Activity size={22} className="text-muted/40 animate-pulse" />
                          <span>Awaiting live {isDisk ? 'disk' : 'CPU'} metrics for {activeServer?.name || activeServer?.hostname || 'node'}</span>
                          <span className="text-[10px] text-text-dim">Streaming from TimescaleDB</span>
                        </div>
                      );
                    }

                    return (
                      <div className="h-56 w-full">
                        <ResponsiveContainer width="100%" height="100%">
                          <LineChart data={chartData}>
                            <CartesianGrid strokeDasharray="3 3" stroke="#262b35" vertical={false} />
                            <XAxis dataKey="time" stroke="#717a8c" tick={{ fontSize: 10 }} />
                            <YAxis stroke="#717a8c" tick={{ fontSize: 10 }} domain={[0, 100]} />
                            <Tooltip
                              contentStyle={{
                                backgroundColor: '#161920',
                                borderColor: '#303642',
                                borderRadius: '8px',
                                fontSize: '11px',
                              }}
                            />
                            <Line
                              type="monotone"
                              dataKey="value"
                              stroke="#6366f1"
                              strokeWidth={2.5}
                              dot={{ fill: '#6366f1', r: 3 }}
                            />
                          </LineChart>
                        </ResponsiveContainer>
                      </div>
                    );
                  })()}

                  {widget.type === 'METRIC_GAUGE' && (() => {
                    const isMem = widget.metric?.includes('memory') || widget.title?.toLowerCase().includes('pool') || widget.title?.toLowerCase().includes('database') || widget.title?.toLowerCase().includes('memory');
                    const rawVal = isMem ? (activeServer?.memory ?? 0) : (activeServer?.cpu ?? 0);
                    const gaugeVal = Math.round(rawVal * 10) / 10;
                    const gaugeColor = gaugeVal >= 85 ? 'text-critical' : gaugeVal >= 70 ? 'text-warning' : 'text-healthy';
                    const barColor = gaugeVal >= 85 ? 'bg-critical' : gaugeVal >= 70 ? 'bg-warning' : 'bg-healthy';

                    return (
                      <div className="space-y-4 py-4 text-center">
                        <div className={`text-4xl font-extrabold font-mono ${gaugeColor}`}>{gaugeVal}%</div>
                        <div className="w-full bg-surface-highlight rounded-full h-3 overflow-hidden max-w-xs mx-auto">
                          <div
                            className={`h-full rounded-full transition-all duration-500 ${barColor}`}
                            style={{ width: `${Math.min(100, Math.max(0, gaugeVal))}%` }}
                          />
                        </div>
                        <p className="text-[11px] font-mono text-muted">
                          {activeServer
                            ? `${activeServer.name || activeServer.hostname} • ${isMem ? 'RAM Allocation' : 'CPU Load'}`
                            : 'Awaiting host telemetry...'}
                        </p>
                      </div>
                    );
                  })()}

                  {widget.type === 'LOG_STREAM' && (
                    <div className="p-3 rounded-xl bg-background/80 border border-border font-mono text-xs space-y-2 max-h-56 overflow-y-auto">
                      {logsData?.logs && logsData.logs.length > 0 ? (
                        logsData.logs.map((log) => (
                          <div key={log.id} className="truncate flex items-center gap-2">
                            <span
                              className={`font-bold text-[10px] shrink-0 ${
                                log.level === 'ERROR'
                                  ? 'text-critical'
                                  : log.level === 'WARN'
                                  ? 'text-warning'
                                  : 'text-text-dim'
                              }`}
                            >
                              [{log.level}]
                            </span>
                            <span className="text-muted text-[10px] shrink-0">
                              {new Date(log.timestamp).toLocaleTimeString()}
                            </span>
                            <span className="text-white font-medium truncate max-w-[120px] shrink-0">
                              {log.service || log.server || 'host'}:
                            </span>
                            <span className="text-text truncate">{log.message}</span>
                          </div>
                        ))
                      ) : (
                        <div className="text-muted text-center py-6 italic text-xs">
                          No recent warning or error logs recorded. Log stream nominal.
                        </div>
                      )}
                    </div>
                  )}

                  {widget.type === 'ALERT_LIST' && (
                    <div className="space-y-2 font-mono text-xs">
                      {openIncidents && openIncidents.length > 0 ? (
                        openIncidents.slice(0, 5).map((inc) => (
                          <div
                            key={inc.id}
                            className="p-2.5 rounded-lg bg-critical/10 border border-critical/30 flex items-center justify-between"
                          >
                            <span className="text-critical font-bold truncate">{inc.title}</span>
                            <span className="text-[10px] px-1.5 py-0.5 rounded bg-critical text-white uppercase">
                              {inc.status || 'FIRING'}
                            </span>
                          </div>
                        ))
                      ) : (
                        <div className="p-3.5 rounded-xl bg-healthy/10 border border-healthy/30 text-healthy flex items-center gap-2.5 text-xs">
                          <CheckCircle2 size={16} className="shrink-0" />
                          <span>All Systems Nominal — Zero active alerts</span>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                <div className="pt-2 border-t border-border/40 text-[10px] font-mono text-muted text-right flex items-center justify-end gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-healthy animate-pulse" />
                  <span>{activeServer ? `Live Telemetry • ${activeServer.name || activeServer.hostname}` : 'Live Telemetry Fleet'}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* New Dashboard Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="glass-card rounded-2xl p-6 border border-primary/40 shadow-glow-primary max-w-md w-full space-y-4">
            <h2 className="text-base font-bold text-white flex items-center gap-2">
              <LayoutDashboard size={16} className="text-primary-light" />
              <span>Create Observability Workspace</span>
            </h2>

            <form onSubmit={handleCreateDashboard} className="space-y-3">
              <div>
                <label className="block text-xs font-mono text-muted mb-1">Dashboard Title</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Edge Ingress & Network Gateway"
                  value={dashTitle}
                  onChange={(e) => setDashTitle(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-xs font-mono text-muted mb-1">Description</label>
                <textarea
                  rows={2}
                  placeholder="Focus areas, microservice groups, and key metrics..."
                  value={dashDescription}
                  onChange={(e) => setDashDescription(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-muted hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createDashboard.isPending}
                  className="px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary"
                >
                  Create Dashboard
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Widget Modal */}
      {showAddWidgetModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="glass-card rounded-2xl p-6 border border-primary/40 shadow-glow-primary max-w-md w-full space-y-4">
            <h2 className="text-base font-bold text-white flex items-center gap-2">
              <Plus size={16} className="text-primary-light" />
              <span>Add Visualization Widget</span>
            </h2>

            <form onSubmit={handleAddWidget} className="space-y-3">
              <div>
                <label className="block text-xs font-mono text-muted mb-1">Widget Title</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Cluster Ingress Throughput"
                  value={widgetTitle}
                  onChange={(e) => setWidgetTitle(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Widget Type</label>
                  <select
                    value={widgetType}
                    onChange={(e) => setWidgetType(e.target.value as any)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white font-mono focus:outline-none"
                  >
                    <option value="METRIC_LINE">Metric Line Chart</option>
                    <option value="METRIC_GAUGE">Radial/Bar Gauge</option>
                    <option value="LOG_STREAM">Live Log Stream</option>
                    <option value="ALERT_LIST">Active Incidents</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Grid Column Span</label>
                  <select
                    value={widgetColSpan}
                    onChange={(e) => setWidgetColSpan(Number(e.target.value))}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white font-mono focus:outline-none"
                  >
                    <option value={1}>1 Column</option>
                    <option value={2}>2 Columns (Wide)</option>
                    <option value={3}>3 Columns (Full Width)</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-xs font-mono text-muted mb-1">Telemetry Stream</label>
                <select
                  value={widgetMetric}
                  onChange={(e) => setWidgetMetric(e.target.value)}
                  className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white font-mono focus:outline-none"
                >
                  <option value="system.cpu.utilization">system.cpu.utilization</option>
                  <option value="system.memory.utilization">system.memory.utilization</option>
                  <option value="system.disk.utilization">system.disk.utilization</option>
                </select>
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAddWidgetModal(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-muted hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary"
                >
                  Add to Dashboard
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
