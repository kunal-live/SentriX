import { ReactNode, useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Grid,
  Activity,
  FileText,
  GitCommit,
  Layers,
  Cpu,
  Boxes,
  Box,
  Radio,
  AlertTriangle,
  BellRing,
  CheckCircle2,
  Terminal,
  Database,
  Globe,
  Shield,
  ShieldCheck,
  Users,
  Bell,
  Code,
  LogOut,
  Search,
  ChevronRight,
  ChevronDown,
} from 'lucide-react';
import { useAuth } from '../hooks/useAuth';
import { useRealtime } from '../hooks/useRealtime';
import { useDashboardSummary } from '../hooks/useDashboard';
import { CommandPalette } from './ui/CommandPalette';

export interface NavItem {
  path: string;
  label: string;
  icon: any;
  badge?: 'incidents' | null;
}

export interface NavSection {
  id: string;
  title: string;
  items: NavItem[];
}

export const navSections: NavSection[] = [
  {
    id: 'observe',
    title: 'Observability',
    items: [
      { path: '/', label: 'Fleet Overview', icon: LayoutDashboard },
      { path: '/dashboards', label: 'Custom Dashboards', icon: Grid },
      { path: '/metrics', label: 'Metrics Explorer', icon: Activity },
      { path: '/logs', label: 'Central Logs', icon: FileText },
      { path: '/traces', label: 'Distributed Tracing', icon: GitCommit },
      { path: '/services', label: 'Services Catalog', icon: Layers },
      { path: '/infrastructure', label: 'Host Topology', icon: Cpu },
    ],
  },
  {
    id: 'workloads',
    title: 'Cloud & Workloads',
    items: [
      { path: '/kubernetes', label: 'Kubernetes Clusters', icon: Boxes },
      { path: '/containers', label: 'Docker Containers', icon: Box },
      { path: '/agents', label: 'Agent Fleet v2', icon: Radio },
    ],
  },
  {
    id: 'reliability',
    title: 'Reliability & Ops',
    items: [
      { path: '/incidents', label: 'Incidents', icon: AlertTriangle, badge: 'incidents' },
      { path: '/alerts', label: 'Alert Rules', icon: BellRing },
      { path: '/checks', label: 'Health Checks', icon: CheckCircle2 },
      { path: '/runbooks', label: 'Runbook Automation', icon: Terminal },
    ],
  },
  {
    id: 'diagnostics',
    title: 'Diagnostics & AI',
    items: [
      { path: '/databases', label: 'Database Health', icon: Database },
      { path: '/network', label: 'Network Diagnostics', icon: Globe },
    ],
  },
  {
    id: 'platform',
    title: 'Platform & Security',
    items: [
      { path: '/security', label: 'Security Center', icon: Shield },
      { path: '/audit', label: 'Audit Logs', icon: ShieldCheck },
      { path: '/users', label: 'Team & RBAC', icon: Users },
      { path: '/notifications', label: 'Notification Channels', icon: Bell },
      { path: '/api-keys', label: 'Developer Platform', icon: Code },
    ],
  },
];

const navItems = navSections.flatMap((section) => section.items);

export function Layout({ children }: { children: ReactNode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const wsStatus = useRealtime();
  const { data: summary, isLoading: isSummaryLoading, isError: isSummaryError } = useDashboardSummary();
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>({});

  // Global keyboard shortcut to summon command palette anywhere
  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setCommandPaletteOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handleGlobalKeyDown);
    return () => window.removeEventListener('keydown', handleGlobalKeyDown);
  }, []);

  const toggleSection = (sectionId: string) => {
    setCollapsedSections((prev) => ({
      ...prev,
      [sectionId]: !prev[sectionId],
    }));
  };

  const openIncidents = summary?.incidents?.open ?? 0;
  const criticalIncidents = summary?.incidents?.critical ?? 0;
  const offlineNodes = summary?.servers?.offline ?? 0;
  const suspectNodes = summary?.servers?.suspect ?? 0;

  let statusBadge = {
    to: '/incidents',
    bgColor: 'bg-healthy/10 hover:bg-healthy/15 border-healthy/30 text-healthy',
    icon: CheckCircle2,
    iconColor: 'text-healthy',
    label: 'All Systems Nominal',
    subtext: null as string | null,
    pulse: false,
  };

  if (isSummaryError) {
    statusBadge = {
      to: '/',
      bgColor: 'bg-critical/15 hover:bg-critical/25 border-critical/40 text-critical',
      icon: AlertTriangle,
      iconColor: 'text-critical',
      label: 'Telemetry Offline',
      subtext: 'API Error',
      pulse: false,
    };
  } else if (isSummaryLoading && !summary) {
    statusBadge = {
      to: '/',
      bgColor: 'bg-surface/80 hover:bg-surface border-border text-muted',
      icon: Activity,
      iconColor: 'text-muted animate-spin',
      label: 'Checking Telemetry...',
      subtext: null,
      pulse: false,
    };
  } else if (openIncidents > 0 || criticalIncidents > 0) {
    statusBadge = {
      to: '/incidents',
      bgColor: 'bg-critical/15 hover:bg-critical/25 border-critical/40 text-critical',
      icon: AlertTriangle,
      iconColor: 'text-critical',
      label: criticalIncidents > 0
        ? `${criticalIncidents} Critical Incident${criticalIncidents > 1 ? 's' : ''}`
        : `${openIncidents} Active Incident${openIncidents > 1 ? 's' : ''}`,
      subtext: 'Triage',
      pulse: true,
    };
  } else if (offlineNodes > 0) {
    statusBadge = {
      to: '/',
      bgColor: 'bg-critical/15 hover:bg-critical/25 border-critical/40 text-critical',
      icon: AlertTriangle,
      iconColor: 'text-critical',
      label: `${offlineNodes} Node${offlineNodes > 1 ? 's' : ''} Offline`,
      subtext: 'Fleet',
      pulse: true,
    };
  } else if (suspectNodes > 0) {
    statusBadge = {
      to: '/',
      bgColor: 'bg-warning/15 hover:bg-warning/25 border-warning/40 text-warning',
      icon: AlertTriangle,
      iconColor: 'text-warning',
      label: `${suspectNodes} Node${suspectNodes > 1 ? 's' : ''} Suspect`,
      subtext: 'Warning',
      pulse: false,
    };
  }

  async function handleLogout() {
    await logout();
    navigate('/login');
  }

  // Determine active breadcrumb name
  const currentNav = navItems.find((item) => {
    if (item.path === '/') return location.pathname === '/' || location.pathname.startsWith('/servers');
    return location.pathname.startsWith(item.path);
  });

  return (
    <div className="flex h-screen bg-background ambient-mesh text-text font-sans overflow-hidden">
      {/* Sleek Obsidian Sidebar */}
      <aside className="w-64 glass-panel border-r border-border flex flex-col z-20 select-none">
        {/* Logo & Platform Info */}
        <div className="p-5 border-b border-border flex items-center justify-between">
          <Link to="/" className="flex items-center gap-3 group">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-primary via-primary-hover to-cyan flex items-center justify-center shadow-glow-primary transition-transform group-hover:scale-105">
              <Shield className="text-white w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-1.5">
                <span className="font-extrabold text-base tracking-tight text-white group-hover:text-primary-light transition-colors">
                  SentriX
                </span>
                <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-primary/20 text-primary-light font-semibold border border-primary/30">
                  v1.2.0
                </span>
              </div>
              <p className="text-[11px] text-muted tracking-wide flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-healthy inline-block animate-pulse-glow" />
                Observability Workstation
              </p>
            </div>
          </Link>
        </div>

        {/* Hierarchical Navigation Sections */}
        <div className="px-3 py-3 flex-1 space-y-4 overflow-y-auto">
          {navSections.map((section) => {
            const hasActiveChild = section.items.some((item) =>
              item.path === '/'
                ? location.pathname === '/' || location.pathname.startsWith('/servers')
                : location.pathname.startsWith(item.path)
            );
            // Keep section open if user explicitly toggled it open, or if it has an active child
            const isExplicitlyToggled = collapsedSections[section.id] !== undefined;
            const isCollapsed = isExplicitlyToggled ? collapsedSections[section.id] : false;

            const sectionIncidentCount = section.items.some((i) => i.badge === 'incidents')
              ? openIncidents
              : 0;

            return (
              <div key={section.id} className="space-y-1">
                {/* Section Header with Accordion Toggle */}
                <button
                  type="button"
                  onClick={() => toggleSection(section.id)}
                  className="w-full flex items-center justify-between px-2 py-1 text-[10px] font-mono font-bold uppercase tracking-wider text-muted/70 hover:text-white transition-colors group select-none"
                >
                  <span className="flex items-center gap-1.5">
                    <ChevronDown
                      size={11}
                      className={`text-muted/60 group-hover:text-white transition-transform duration-150 ${
                        isCollapsed ? '-rotate-90' : 'rotate-0'
                      }`}
                    />
                    <span className={hasActiveChild ? 'text-primary-light font-bold' : ''}>
                      {section.title}
                    </span>
                  </span>

                  {/* Section-level badge when collapsed with active incidents */}
                  {isCollapsed && sectionIncidentCount > 0 && (
                    <span className="text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-critical/20 text-critical border border-critical/30 animate-pulse">
                      {sectionIncidentCount}
                    </span>
                  )}
                </button>

                {/* Section Links */}
                {!isCollapsed && (
                  <nav className="space-y-0.5 pt-0.5">
                    {section.items.map((item) => {
                      const isActive =
                        item.path === '/'
                          ? location.pathname === '/' || location.pathname.startsWith('/servers')
                          : location.pathname.startsWith(item.path);
                      const Icon = item.icon;

                      return (
                        <Link
                          key={item.path}
                          to={item.path}
                          className={`flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all group relative ${
                            isActive
                              ? 'bg-primary/20 text-white font-semibold border border-primary/35 shadow-sm'
                              : 'text-slate-400 hover:text-white hover:bg-surface/80 hover:border hover:border-border/60'
                          }`}
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            <div
                              className={`p-1 rounded-md transition-colors shrink-0 ${
                                isActive
                                  ? 'bg-primary text-white shadow-sm'
                                  : 'text-muted group-hover:text-white bg-surface-highlight/50'
                              }`}
                            >
                              <Icon size={14} />
                            </div>
                            <span className="truncate tracking-tight">{item.label}</span>
                          </div>

                          {/* Dynamic badges */}
                          {item.badge === 'incidents' && openIncidents > 0 && (
                            <span className="text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-critical/20 text-critical border border-critical/30 animate-pulse ml-2 shrink-0">
                              {openIncidents}
                            </span>
                          )}
                        </Link>
                      );
                    })}
                  </nav>
                )}
              </div>
            );
          })}

          {/* Quick Metrics Glance in Sidebar */}
          <div className="px-3 pt-2">
            <div className="p-3.5 rounded-xl bg-surface/60 border border-border space-y-2.5">
              <div className="flex items-center justify-between text-xs text-muted">
                <span className="flex items-center gap-1.5 font-medium">
                  <Layers size={13} className="text-primary-light" /> Fleet Nodes
                </span>
                <span className="font-mono text-white font-semibold">
                  {summary?.servers.online ?? 0} / {summary?.servers.total ?? 0}
                </span>
              </div>
              <div className="w-full bg-background rounded-full h-1.5 overflow-hidden flex">
                <div
                  className="bg-healthy h-full transition-all duration-500"
                  style={{
                    width: `${
                      summary?.servers.total
                        ? (summary.servers.online / summary.servers.total) * 100
                        : 0
                    }%`,
                  }}
                />
                <div
                  className="bg-warning h-full transition-all duration-500"
                  style={{
                    width: `${
                      summary?.servers.total
                        ? (summary.servers.suspect / summary.servers.total) * 100
                        : 0
                    }%`,
                  }}
                />
              </div>
              <div className="flex justify-between text-[10px] text-muted font-mono">
                <span className="flex items-center gap-1 text-healthy">
                  <span className="w-1.5 h-1.5 rounded-full bg-healthy" /> Online
                </span>
                <span className="flex items-center gap-1 text-warning">
                  <span className="w-1.5 h-1.5 rounded-full bg-warning" /> Warning
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Sidebar Footer: System heartbeat & status */}
        <div className="p-4 border-t border-border space-y-3 bg-background/40">
          <div className="flex items-center justify-between px-2 py-1.5 rounded-lg bg-surface/50 border border-border">
            <div className="flex items-center gap-2 text-xs">
              <span
                className={`w-2 h-2 rounded-full ${
                  wsStatus === 'connected'
                    ? 'bg-healthy animate-pulse-glow shadow-glow-healthy'
                    : 'bg-warning'
                }`}
              />
              <span className="text-muted font-mono text-[11px]">
                {wsStatus === 'connected' ? 'WS STREAM' : 'POLLING'}
              </span>
            </div>
            <span className="text-[10px] font-mono text-healthy bg-healthy/10 px-1.5 py-0.5 rounded border border-healthy/20">
              99.98% SLA
            </span>
          </div>

          <div className="flex items-center justify-between text-xs px-1">
            <div className="flex items-center gap-2">
              <div className="w-7 h-7 rounded-lg bg-gradient-to-tr from-primary to-cyan flex items-center justify-center font-bold text-white text-xs shadow-sm">
                {user?.email?.[0]?.toUpperCase() || 'A'}
              </div>
              <div className="overflow-hidden">
                <div className="text-white text-xs font-semibold truncate max-w-[105px]">
                  {user?.email?.split('@')[0] || 'admin'}
                </div>
                <div className="text-[10px] font-mono text-muted uppercase">
                  {user?.role || 'ADMIN'}
                </div>
              </div>
            </div>

            <button
              onClick={handleLogout}
              title="Sign Out"
              className="p-1.5 text-muted hover:text-critical hover:bg-critical/10 rounded-lg transition-colors border border-transparent hover:border-critical/20"
            >
              <LogOut size={15} />
            </button>
          </div>
        </div>
      </aside>

      {/* Main Container */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Modern Top Header Bar */}
        <header className="h-16 glass-panel border-b border-border px-8 flex items-center justify-between z-10">
          {/* Breadcrumb & Current Context */}
          <div className="flex items-center gap-2.5 text-sm">
            <Link to="/" className="text-muted hover:text-white transition-colors font-medium">
              SentriX
            </Link>
            <ChevronRight size={14} className="text-muted/60" />
            <span className="font-semibold text-white tracking-tight">
              {currentNav?.label || 'Workstation'}
            </span>
          </div>

          {/* Platform Status Badge & Search shortcut */}
          <div className="hidden md:flex items-center gap-3">
            <div
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface/80 border border-border text-xs font-mono"
            >
              <Shield size={13} className="text-primary-light" />
              <span className="text-white font-semibold tracking-tight">SentriX Core</span>
              <span className="px-1.5 py-0.5 rounded bg-primary/20 text-primary-light font-bold text-[10px] border border-primary/30">
                Production
              </span>
            </div>

            <button
              onClick={() => setCommandPaletteOpen(true)}
              className="relative w-64 flex items-center justify-between bg-surface/70 border border-border hover:border-primary/50 rounded-xl pl-9 pr-3 py-1.5 text-xs text-muted hover:text-white transition-all shadow-sm group text-left"
            >
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted group-hover:text-primary transition-colors" size={14} />
              <span>Search platform...</span>
              <kbd className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-surface-highlight text-muted border border-border group-hover:border-primary/40 group-hover:text-white transition-colors">
                ⌘K
              </kbd>
            </button>
          </div>

          {/* Right Live indicators & actions */}
          <div className="flex items-center gap-2.5">
            {/* Live Streaming Status Pill */}
            {wsStatus === 'connected' ? (
              <div
                className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-surface/80 border border-border text-[11px] font-mono text-muted shadow-sm"
                title="Connected to SentriX streaming ingestion gateway"
              >
                <span className="w-1.5 h-1.5 rounded-full bg-healthy animate-pulse" />
                <span className="text-white font-medium">Real-Time</span>
                <span className="text-healthy text-[10px]">WS Live</span>
              </div>
            ) : wsStatus === 'connecting' ? (
              <div
                className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-surface/80 border border-border text-[11px] font-mono text-muted shadow-sm"
                title="Connecting to streaming gateway..."
              >
                <span className="w-1.5 h-1.5 rounded-full bg-warning animate-pulse" />
                <span className="text-white font-medium">Reconnecting</span>
                <span className="text-warning text-[10px]">WS</span>
              </div>
            ) : (
              <div
                className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-surface/80 border border-border text-[11px] font-mono text-muted shadow-sm"
                title="WebSocket disconnected; falling back to 5s active polling"
              >
                <Radio size={12} className="text-muted" />
                <span className="text-white font-medium">Polling</span>
                <span className="text-muted text-[10px]">5s</span>
              </div>
            )}

            {/* Reactive System Status Badge */}
            <Link
              to={statusBadge.to}
              title={`System Status: ${statusBadge.label}. Click to investigate.`}
              className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-mono font-semibold border transition-all shadow-sm ${statusBadge.bgColor}`}
            >
              <statusBadge.icon
                size={13}
                className={`${statusBadge.iconColor} ${statusBadge.pulse ? 'animate-pulse' : ''}`}
              />
              <span className="tracking-tight">{statusBadge.label}</span>
              {statusBadge.subtext && (
                <span className="text-[10px] uppercase tracking-wider px-1 py-0.2 rounded bg-black/30 opacity-90">
                  {statusBadge.subtext}
                </span>
              )}
            </Link>
          </div>
        </header>

        {/* Scrollable Workstation Canvas */}
        <main className="flex-1 overflow-y-auto p-8 relative">
          {children}
        </main>
      </div>

      <CommandPalette
        isOpen={commandPaletteOpen}
        onClose={() => setCommandPaletteOpen(false)}
      />
    </div>
  );
}
