import { FormEvent, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useServices, useCreateService, ServiceEntity } from '../../hooks/useServices';
import {
  Layers,
  Plus,
  Server as ServerIcon,
  ShieldCheck,
  AlertTriangle,
  GitBranch,
  User,
  Users,
  ExternalLink,
  Search,
  Activity,
  CheckCircle2,
  AlertOctagon,
  Zap,
  Clock,
  Flame,
  ArrowRight,
  Code2,
  FileCode,
  Terminal,
  Check,
  Copy,
} from 'lucide-react';

export function ServicesPage() {
  const navigate = useNavigate();
  const { data: services, isLoading } = useServices();
  const createService = useCreateService();

  const [activeTab, setActiveTab] = useState<'apm' | 'catalog' | 'instrument'>('apm');
  const [search, setSearch] = useState('');
  const [envFilter, setEnvFilter] = useState<string>('ALL');
  const [showModal, setShowModal] = useState(false);
  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const [sdkLanguage, setSdkLanguage] = useState<'go' | 'node' | 'python' | 'docker'>('go');

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [environment, setEnvironment] = useState('Production');
  const [tier, setTier] = useState('Tier 1 (Mission Critical)');
  const [criticality, setCriticality] = useState('High');
  const [owner, setOwner] = useState('');
  const [team, setTeam] = useState('');
  const [repository, setRepository] = useState('');

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    await createService.mutateAsync({
      name,
      description,
      environment,
      tier,
      criticality,
      owner: owner || 'Platform Team',
      team: team || 'Core Engineering',
      repository: repository || 'github.com/sentrix/service',
      server_count: 2,
      health: 'HEALTHY',
      rps: 120.0,
      p50_ms: 12.0,
      p95_ms: 45.0,
      p99_ms: 110.0,
      error_rate_pct: 0.01,
      apdex_score: 0.99,
      throughput_mb_s: 2.5,
    });
    setName('');
    setDescription('');
    setShowModal(false);
  }

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCode(id);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const filtered = (services || []).filter((s) => {
    if (envFilter !== 'ALL' && s.environment !== envFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        s.name.toLowerCase().includes(q) ||
        s.description.toLowerCase().includes(q) ||
        s.team.toLowerCase().includes(q) ||
        (s.endpoints && s.endpoints.some((ep) => ep.path.toLowerCase().includes(q)))
      );
    }
    return true;
  });

  // Calculate fleet-wide APM rollups
  const totalRPS = filtered.reduce((acc, s) => acc + (s.rps || 0), 0);
  const avgP95 = filtered.length > 0
    ? filtered.reduce((acc, s) => acc + (s.p95_ms || 0), 0) / filtered.length
    : 0;
  const avgErrorRate = filtered.length > 0
    ? filtered.reduce((acc, s) => acc + (s.error_rate_pct || 0), 0) / filtered.length
    : 0;
  const avgApdex = filtered.length > 0
    ? filtered.reduce((acc, s) => acc + (s.apdex_score || 0.95), 0) / filtered.length
    : 0.98;

  return (
    <div className="max-w-7xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight flex items-center gap-3">
              Application Performance Monitoring (APM)
            </h1>
            <span className="text-xs font-mono font-bold px-2.5 py-1 rounded-full bg-primary/20 text-primary-light border border-primary/30">
              {services?.length ?? 0} Services
            </span>
          </div>
          <p className="text-sm text-muted mt-1">
            Service-centric golden signals: request throughput (RPS), P95/P99 latency, error rates, Apdex scores, and correlated distributed traces.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab('instrument')}
            className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-surface hover:bg-[#252932] border border-border text-xs font-semibold text-text-dim hover:text-white transition-all"
          >
            <Code2 size={15} className="text-primary-light" />
            <span>OTel SDK Guide</span>
          </button>
          <button
            onClick={() => setShowModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary transition-all"
          >
            <Plus size={15} />
            <span>Register Service</span>
          </button>
        </div>
      </div>

      {/* Primary View Switcher Tabs */}
      <div className="flex items-center justify-between border-b border-border/80 pb-3">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab('apm')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all flex items-center gap-2 ${
              activeTab === 'apm'
                ? 'bg-primary text-white shadow-glow-primary'
                : 'bg-surface text-muted hover:text-white border border-border'
            }`}
          >
            <Activity size={14} />
            <span>APM Golden Signals & Fleet</span>
          </button>
          <button
            onClick={() => setActiveTab('catalog')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all flex items-center gap-2 ${
              activeTab === 'catalog'
                ? 'bg-primary text-white shadow-glow-primary'
                : 'bg-surface text-muted hover:text-white border border-border'
            }`}
          >
            <Layers size={14} />
            <span>Catalog & Ownership</span>
          </button>
          <button
            onClick={() => setActiveTab('instrument')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all flex items-center gap-2 ${
              activeTab === 'instrument'
                ? 'bg-primary text-white shadow-glow-primary'
                : 'bg-surface text-muted hover:text-white border border-border'
            }`}
          >
            <Terminal size={14} />
            <span>Auto-Instrumentation (OTel / eBPF)</span>
          </button>
        </div>

        <div className="hidden md:flex items-center gap-2 text-xs font-mono text-muted">
          <span className="w-2 h-2 rounded-full bg-healthy animate-ping" />
          <span>OTLP Ingestion Active</span>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="glass-card rounded-2xl p-5 border border-border">
          <div className="flex items-center justify-between text-muted text-xs font-mono mb-2">
            <span>Fleet Request Rate</span>
            <Zap size={16} className="text-primary-light" />
          </div>
          <div className="text-2xl font-bold text-white font-mono">
            {totalRPS.toLocaleString('en-US', { maximumFractionDigits: 1 })} <span className="text-xs font-normal text-muted">req/s</span>
          </div>
          <div className="text-[11px] text-muted mt-1">Total incoming HTTP & gRPC traffic</div>
        </div>

        <div className="glass-card rounded-2xl p-5 border border-border">
          <div className="flex items-center justify-between text-muted text-xs font-mono mb-2">
            <span>Fleet P95 Latency</span>
            <Clock size={16} className="text-amber-400" />
          </div>
          <div className="text-2xl font-bold text-amber-400 font-mono">
            {avgP95.toFixed(1)} <span className="text-xs font-normal text-muted">ms</span>
          </div>
          <div className="text-[11px] text-muted mt-1">Weighted 95th percentile response time</div>
        </div>

        <div className="glass-card rounded-2xl p-5 border border-border">
          <div className="flex items-center justify-between text-muted text-xs font-mono mb-2">
            <span>Global Error Rate</span>
            <Flame size={16} className="text-rose-400" />
          </div>
          <div className="text-2xl font-bold text-rose-400 font-mono">
            {avgErrorRate.toFixed(2)}%
          </div>
          <div className="text-[11px] text-muted mt-1">HTTP 5xx & unhandled RPC failures</div>
        </div>

        <div className="glass-card rounded-2xl p-5 border border-border">
          <div className="flex items-center justify-between text-muted text-xs font-mono mb-2">
            <span>Fleet Apdex Index</span>
            <ShieldCheck size={16} className="text-healthy" />
          </div>
          <div className="text-2xl font-bold text-healthy font-mono">
            {avgApdex.toFixed(2)} <span className="text-xs font-normal text-muted">/ 1.00</span>
          </div>
          <div className="text-[11px] text-muted mt-1">User experience satisfaction rating</div>
        </div>
      </div>

      {/* Tab 1: APM Golden Signals View */}
      {activeTab === 'apm' && (
        <div className="space-y-6">
          {/* Filters & Search */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              {['ALL', 'Production', 'Staging', 'Development'].map((env) => (
                <button
                  key={env}
                  onClick={() => setEnvFilter(env)}
                  className={`px-3 py-1.5 rounded-lg text-xs font-mono font-semibold transition-all ${
                    envFilter === env
                      ? 'bg-primary text-white shadow-glow-primary'
                      : 'bg-surface text-muted hover:text-white border border-border'
                  }`}
                >
                  {env}
                </button>
              ))}
            </div>

            <div className="relative w-full max-w-xs">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" size={14} />
              <input
                type="text"
                placeholder="Search services or endpoints..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-surface border border-border focus:border-primary rounded-xl pl-9 pr-3 py-1.5 text-xs text-white placeholder-muted focus:outline-none"
              />
            </div>
          </div>

          {/* APM Service Cards Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {isLoading && (
              <div className="col-span-full py-16 text-center text-muted font-mono text-xs">
                Aggregating live APM golden signals...
              </div>
            )}

            {filtered.map((service) => {
              const isHealthy = service.health === 'HEALTHY';
              const isDegraded = service.health === 'DEGRADED';
              const p95Color = (service.p95_ms || 45) > 150 ? 'text-rose-400' : (service.p95_ms || 45) > 80 ? 'text-amber-400' : 'text-emerald-400';
              const errColor = (service.error_rate_pct || 0) > 1.0 ? 'text-rose-400' : (service.error_rate_pct || 0) > 0.1 ? 'text-amber-400' : 'text-emerald-400';

              return (
                <div
                  key={service.id}
                  className="glass-card-interactive rounded-2xl p-6 border border-border flex flex-col justify-between space-y-5 group hover:border-primary/50 transition-all"
                >
                  <div className="space-y-4">
                    {/* Top Row: Meta & Status */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-surface border border-border text-primary-light">
                          {service.environment}
                        </span>
                        <span className="text-[10px] font-mono text-text-dim">
                          {service.tier}
                        </span>
                      </div>

                      <span
                        className={`inline-flex items-center gap-1.5 text-[11px] font-mono font-bold px-2.5 py-0.5 rounded-full border ${
                          isHealthy
                            ? 'bg-healthy/15 text-healthy border-healthy/30'
                            : isDegraded
                            ? 'bg-warning/15 text-warning border-warning/30 animate-pulse'
                            : 'bg-critical/15 text-critical border-critical/30'
                        }`}
                      >
                        {isHealthy ? <CheckCircle2 size={12} /> : <AlertTriangle size={12} />}
                        {service.health}
                      </span>
                    </div>

                    {/* Title & Description */}
                    <div>
                      <h3 className="text-lg font-bold text-white group-hover:text-primary-light transition-colors">
                        {service.name}
                      </h3>
                      <p className="text-xs text-muted mt-1 leading-relaxed line-clamp-2">
                        {service.description}
                      </p>
                    </div>

                    {/* Golden Signals 4-Box Grid */}
                    <div className="grid grid-cols-4 gap-2 p-3 rounded-xl bg-background/70 border border-border text-center">
                      <div className="space-y-1">
                        <div className="text-[10px] font-mono text-muted">Throughput</div>
                        <div className="text-sm font-bold text-white font-mono">
                          {(service.rps || 120).toFixed(0)} <span className="text-[9px] text-muted">rps</span>
                        </div>
                      </div>

                      <div className="space-y-1">
                        <div className="text-[10px] font-mono text-muted">P95 Latency</div>
                        <div className={`text-sm font-bold font-mono ${p95Color}`}>
                          {(service.p95_ms || 35.0).toFixed(1)} <span className="text-[9px] text-muted">ms</span>
                        </div>
                      </div>

                      <div className="space-y-1">
                        <div className="text-[10px] font-mono text-muted">Error Rate</div>
                        <div className={`text-sm font-bold font-mono ${errColor}`}>
                          {(service.error_rate_pct || 0.01).toFixed(2)}%
                        </div>
                      </div>

                      <div className="space-y-1">
                        <div className="text-[10px] font-mono text-muted">Apdex</div>
                        <div className="text-sm font-bold text-healthy font-mono">
                          {(service.apdex_score || 0.98).toFixed(2)}
                        </div>
                      </div>
                    </div>

                    {/* Latency Distribution Breakdown */}
                    <div className="space-y-1.5 text-xs font-mono">
                      <div className="flex items-center justify-between text-[11px] text-muted">
                        <span>Latency Distribution (P50 / P95 / P99)</span>
                        <span className="text-text-dim">
                          P50: {(service.p50_ms || 12.0).toFixed(1)}ms • P99: {(service.p99_ms || 95.0).toFixed(1)}ms
                        </span>
                      </div>
                      <div className="w-full bg-[#181c23] rounded-full h-2 overflow-hidden flex">
                        <div className="bg-emerald-500 h-full" style={{ width: '60%' }} title="P50 Normal" />
                        <div className="bg-amber-500 h-full" style={{ width: '25%' }} title="P95 Elevated" />
                        <div className="bg-rose-500 h-full" style={{ width: '15%' }} title="P99 Tail Latency" />
                      </div>
                    </div>

                    {/* Key Endpoints Preview */}
                    {service.endpoints && service.endpoints.length > 0 && (
                      <div className="space-y-2 pt-1">
                        <div className="text-[11px] font-mono text-muted uppercase tracking-wider">
                          Key Endpoints Telemetry
                        </div>
                        <div className="space-y-1.5">
                          {service.endpoints.slice(0, 3).map((ep, idx) => (
                            <div
                              key={idx}
                              className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[#14171d] border border-border/60 text-xs font-mono"
                            >
                              <div className="flex items-center gap-2 truncate">
                                <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
                                  ep.method === 'GET' ? 'bg-blue-500/20 text-blue-400' :
                                  ep.method === 'POST' ? 'bg-emerald-500/20 text-emerald-400' :
                                  'bg-purple-500/20 text-purple-400'
                                }`}>
                                  {ep.method}
                                </span>
                                <span className="text-gray-300 truncate">{ep.path}</span>
                              </div>
                              <div className="flex items-center gap-3 text-muted shrink-0">
                                <span>{ep.rps} rps</span>
                                <span className="text-white font-medium">{ep.p95_ms}ms</span>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Footer Action Buttons */}
                  <div className="pt-4 border-t border-border/70 flex items-center justify-between">
                    <div className="flex items-center gap-2 text-xs font-mono text-muted">
                      <span>{service.team}</span>
                      <span>•</span>
                      <span>{service.server_count} nodes</span>
                    </div>

                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => navigate(`/logs?search=${encodeURIComponent(service.name)}`)}
                        className="px-2.5 py-1.5 rounded-lg bg-[#1a1d24] hover:bg-[#252932] border border-border text-xs text-gray-300 hover:text-white transition-colors"
                      >
                        Logs
                      </button>
                      <button
                        onClick={() => navigate(`/traces?service=${encodeURIComponent(service.name)}`)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary/20 hover:bg-primary/30 border border-primary/40 text-xs font-semibold text-primary-light hover:text-white transition-colors"
                      >
                        <span>Traces</span>
                        <ArrowRight size={13} />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Tab 2: Service Catalog & Ownership View */}
      {activeTab === 'catalog' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {filtered.map((service) => (
              <div
                key={service.id}
                className="glass-card rounded-2xl p-6 border border-border space-y-4"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-surface border border-border text-primary-light">
                      {service.environment}
                    </span>
                    <span className="text-[10px] font-mono text-text-dim">
                      {service.tier}
                    </span>
                  </div>
                  <span className="text-xs font-mono font-bold text-healthy">
                    {service.uptime}% SLA
                  </span>
                </div>

                <div>
                  <h3 className="text-lg font-bold text-white">{service.name}</h3>
                  <p className="text-xs text-muted mt-1 leading-relaxed">{service.description}</p>
                </div>

                <div className="p-3 rounded-xl bg-background/60 border border-border grid grid-cols-3 gap-2 text-center text-xs font-mono">
                  <div>
                    <div className="text-[10px] text-muted">Availability</div>
                    <div className="font-bold text-healthy mt-0.5">{service.uptime}%</div>
                  </div>
                  <div>
                    <div className="text-[10px] text-muted">Nodes</div>
                    <div className="font-bold text-white mt-0.5">{service.server_count} instances</div>
                  </div>
                  <div>
                    <div className="text-[10px] text-muted">Incidents</div>
                    <div className={`font-bold mt-0.5 ${service.incident_count > 0 ? 'text-critical' : 'text-muted'}`}>
                      {service.incident_count}
                    </div>
                  </div>
                </div>

                <div className="pt-3 border-t border-border flex items-center justify-between text-xs font-mono text-muted">
                  <div className="flex items-center gap-1.5 text-text-dim truncate">
                    <User size={13} className="text-primary-light" />
                    <span>{service.owner}</span>
                    <span>•</span>
                    <span>{service.team}</span>
                  </div>
                  {service.repository && (
                    <span className="flex items-center gap-1 text-primary-light hover:underline truncate max-w-[160px]">
                      <GitBranch size={13} />
                      <span>{service.repository.replace('github.com/', '')}</span>
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tab 3: OTel & SDK Auto-Instrumentation */}
      {activeTab === 'instrument' && (
        <div className="space-y-6">
          <div className="glass-card rounded-2xl p-6 border border-border space-y-4">
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <Code2 size={18} className="text-primary-light" />
              <span>Instrument Your Services with OpenTelemetry & eBPF</span>
            </h2>
            <p className="text-xs text-muted leading-relaxed">
              SentriX natively ingests standard OpenTelemetry OTLP/HTTP traces, metrics, and logs. Any service instrumented with the OTel SDK or running on an agent host with eBPF auto-instrumentation will stream APM golden signals directly to your SentriX dashboard.
            </p>

            {/* Language Selector */}
            <div className="flex items-center gap-2 pt-2 border-b border-border pb-3">
              {[
                { id: 'go', label: 'Go (Golang)' },
                { id: 'node', label: 'Node.js / Express' },
                { id: 'python', label: 'Python (FastAPI / Flask)' },
                { id: 'docker', label: 'SentriX Agent / eBPF Auto-Trace' },
              ].map((lang) => (
                <button
                  key={lang.id}
                  onClick={() => setSdkLanguage(lang.id as any)}
                  className={`px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium transition-colors ${
                    sdkLanguage === lang.id
                      ? 'bg-primary text-white shadow-glow-primary'
                      : 'bg-surface text-muted hover:text-white border border-border'
                  }`}
                >
                  {lang.label}
                </button>
              ))}
            </div>

            {/* Code Snippets */}
            <div className="relative">
              {sdkLanguage === 'go' && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-muted">
                    <span>main.go - OpenTelemetry OTLP Exporter Configuration</span>
                    <button
                      onClick={() => copyToClipboard(`package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer() (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint("169.58.138.136:8080"),
        otlptracehttp.WithInsecure(),
        otlptracehttp.WithURLPath("/v1/traces"),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("my-microservice"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}`, 'go-code')}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#20242c] hover:bg-[#2b303b] text-text-dim hover:text-white"
                    >
                      {copiedCode === 'go-code' ? <Check size={13} className="text-healthy" /> : <Copy size={13} />}
                      <span>{copiedCode === 'go-code' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-4 rounded-xl bg-[#0e1116] border border-[#232730] text-xs font-mono text-gray-300 overflow-x-auto">
{`package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer() (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint("169.58.138.136:8080"),
        otlptracehttp.WithInsecure(),
        otlptracehttp.WithURLPath("/v1/traces"),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("my-microservice"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}`}
                  </pre>
                </div>
              )}

              {sdkLanguage === 'node' && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-muted">
                    <span>tracing.js - Node.js OTel SDK Setup</span>
                    <button
                      onClick={() => copyToClipboard(`const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

const sdk = new NodeSDK({
  serviceName: 'payments-api',
  traceExporter: new OTLPTraceExporter({
    url: 'http://169.58.138.136:8080/v1/traces',
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();`, 'node-code')}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#20242c] hover:bg-[#2b303b] text-text-dim hover:text-white"
                    >
                      {copiedCode === 'node-code' ? <Check size={13} className="text-healthy" /> : <Copy size={13} />}
                      <span>{copiedCode === 'node-code' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-4 rounded-xl bg-[#0e1116] border border-[#232730] text-xs font-mono text-gray-300 overflow-x-auto">
{`const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

const sdk = new NodeSDK({
  serviceName: 'payments-api',
  traceExporter: new OTLPTraceExporter({
    url: 'http://169.58.138.136:8080/v1/traces',
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();`}
                  </pre>
                </div>
              )}

              {sdkLanguage === 'python' && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-muted">
                    <span>Bash - Python Zero-Code Auto-Instrumentation</span>
                    <button
                      onClick={() => copyToClipboard(`pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install

export OTEL_SERVICE_NAME="customer-service"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://169.58.138.136:8080"
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="http://169.58.138.136:8080/v1/traces"

opentelemetry-instrument uvicorn main:app --port 8000`, 'py-code')}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#20242c] hover:bg-[#2b303b] text-text-dim hover:text-white"
                    >
                      {copiedCode === 'py-code' ? <Check size={13} className="text-healthy" /> : <Copy size={13} />}
                      <span>{copiedCode === 'py-code' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-4 rounded-xl bg-[#0e1116] border border-[#232730] text-xs font-mono text-gray-300 overflow-x-auto">
{`# 1. Install OpenTelemetry auto-instrumentation
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install

# 2. Configure endpoint to SentriX
export OTEL_SERVICE_NAME="customer-service"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://169.58.138.136:8080"
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="http://169.58.138.136:8080/v1/traces"

# 3. Launch application with zero code changes
opentelemetry-instrument uvicorn main:app --port 8000`}
                  </pre>
                </div>
              )}

              {sdkLanguage === 'docker' && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-muted">
                    <span>Host eBPF / Docker Auto-Tracing Daemon</span>
                    <button
                      onClick={() => copyToClipboard(`curl -sSL http://169.58.138.136:8080/install.sh | sudo bash -s -- --token YOUR_TOKEN --enable-ebpf`, 'ebpf-code')}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#20242c] hover:bg-[#2b303b] text-text-dim hover:text-white"
                    >
                      {copiedCode === 'ebpf-code' ? <Check size={13} className="text-healthy" /> : <Copy size={13} />}
                      <span>{copiedCode === 'ebpf-code' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-4 rounded-xl bg-[#0e1116] border border-[#232730] text-xs font-mono text-gray-300 overflow-x-auto">
{`# Enable eBPF kernel network & socket probe in SentriX Agent:
curl -sSL http://169.58.138.136:8080/install.sh | sudo bash -s -- --token YOUR_TOKEN --enable-ebpf

# The agent automatically captures HTTP/gRPC requests, latencies, and error codes
# across all local Docker containers and systemd services without SDK modifications.`}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Register Service Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="glass-card rounded-2xl p-6 border border-primary/40 shadow-glow-primary max-w-lg w-full space-y-5">
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <Layers size={18} className="text-primary-light" />
              <span>Register Microservice in Catalog</span>
            </h2>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-mono text-muted mb-1">Service Name</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Invoicing & Billing Engine"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-xs font-mono text-muted mb-1">Description</label>
                <textarea
                  rows={2}
                  placeholder="Role, primary responsibilities, and architecture..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Environment</label>
                  <select
                    value={environment}
                    onChange={(e) => setEnvironment(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                  >
                    <option value="Production">Production</option>
                    <option value="Staging">Staging</option>
                    <option value="Development">Development</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Tier</label>
                  <select
                    value={tier}
                    onChange={(e) => setTier(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                  >
                    <option value="Tier 1 (Mission Critical)">Tier 1 (Mission Critical)</option>
                    <option value="Tier 2 (Standard)">Tier 2 (Standard)</option>
                    <option value="Tier 3 (Auxiliary)">Tier 3 (Auxiliary)</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Owner</label>
                  <input
                    type="text"
                    placeholder="e.g. Alex Rivera"
                    value={owner}
                    onChange={(e) => setOwner(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs font-mono text-muted mb-1">Engineering Team</label>
                  <input
                    type="text"
                    placeholder="e.g. Core SRE"
                    value={team}
                    onChange={(e) => setTeam(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-mono text-muted mb-1">Git Repository</label>
                <input
                  type="text"
                  placeholder="github.com/sentrix/billing-service"
                  value={repository}
                  onChange={(e) => setRepository(e.target.value)}
                  className="w-full bg-background border border-border rounded-xl px-3 py-2 text-xs text-white focus:outline-none"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-muted hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createService.isPending}
                  className="px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary"
                >
                  {createService.isPending ? 'Registering...' : 'Register Service'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
