import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTraces, Span } from '../../hooks/useTraces';
import { mutateJSON } from '../../lib/api';
import {
  GitCommit,
  Clock,
  Search,
  CheckCircle2,
  AlertTriangle,
  Code,
  FileText,
  Activity,
  Layers,
  X,
  Server,
  Terminal,
  Send,
  Zap,
  Copy,
  Check,
  ChevronDown,
  ChevronRight,
  ExternalLink,
} from 'lucide-react';

export function TracesPage() {
  const navigate = useNavigate();
  const [serviceFilter, setServiceFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTraceId, setSelectedTraceId] = useState<string>('trc_9a8b7c6d5e4f');
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [guideModalOpen, setGuideModalOpen] = useState(false);
  const [sendingTest, setSendingTest] = useState(false);
  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const [guideTab, setGuideTab] = useState<'curl' | 'collector' | 'python' | 'node' | 'go'>('curl');

  const { data: traces, isLoading, refetch } = useTraces({
    service: serviceFilter,
    status: statusFilter,
  });

  const filteredTraces = (traces || []).filter((t) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      t.trace_id.toLowerCase().includes(q) ||
      t.root_service.toLowerCase().includes(q) ||
      t.root_operation.toLowerCase().includes(q)
    );
  });

  const activeTrace =
    filteredTraces.find((t) => t.trace_id === selectedTraceId) ||
    filteredTraces[0] ||
    (traces && traces[0]);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCode(id);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const handleSendTestTrace = async () => {
    setSendingTest(true);
    try {
      const now = new Date();
      const testTraceId = 'trc_live_' + Math.random().toString(36).substring(2, 10);
      const rootSpanId = 'spn_' + Math.random().toString(36).substring(2, 10);
      const childSpanId = 'spn_' + Math.random().toString(36).substring(2, 10);

      await mutateJSON('/traces', 'POST', {
        trace_id: testTraceId,
        root_service: 'api-gateway',
        root_operation: 'GET /api/v2/orders/summary',
        duration_ms: 84.5,
        start_time: now.toISOString(),
        status: 'OK',
        span_count: 2,
        spans: [
          {
            span_id: rootSpanId,
            trace_id: testTraceId,
            service_name: 'api-gateway',
            operation: 'GET /api/v2/orders/summary',
            duration_ms: 84.5,
            start_time: now.toISOString(),
            status: 'OK',
            status_code: 200,
            attributes: {
              'http.method': 'GET',
              'http.route': '/api/v2/orders/summary',
              'http.status_code': 200,
              'client.ip': '127.0.0.1',
              'user_agent': 'SentriX-LiveProbe/1.2',
            },
          },
          {
            span_id: childSpanId,
            parent_span_id: rootSpanId,
            trace_id: testTraceId,
            service_name: 'orders-microservice',
            operation: 'FetchCustomerOrders',
            duration_ms: 62.1,
            start_time: new Date(now.getTime() + 5).toISOString(),
            status: 'OK',
            status_code: 200,
            attributes: {
              'rpc.system': 'grpc',
              'orders.count': 14,
              'cache.hit': true,
            },
          },
        ],
      });

      await refetch();
      setSelectedTraceId(testTraceId);
      setGuideModalOpen(false);
    } catch (err: any) {
      alert('Failed to send test trace: ' + (err.message || 'unknown error'));
    } finally {
      setSendingTest(false);
    }
  };

  const serverOrigin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight flex items-center gap-3">
            Distributed Tracing
            <span className="text-xs font-mono font-bold px-2.5 py-1 rounded-full bg-primary/20 text-primary-light border border-primary/30">
              OpenTelemetry OTLP
            </span>
          </h1>
          <p className="text-sm text-muted mt-1">
            End-to-end request propagation, microservice waterfall timing, and cross-signal correlation with centralized logs.
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            onClick={() => setGuideModalOpen(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white text-xs font-semibold shadow-glow-primary transition-all"
          >
            <Zap size={14} />
            <span>OTel Ingestion Guide</span>
          </button>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="glass-card rounded-2xl p-4 border border-border flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-2">
          {['ALL', 'ERROR', 'OK'].map((st) => (
            <button
              key={st}
              onClick={() => setStatusFilter(st)}
              className={`px-3 py-1.5 rounded-lg text-xs font-mono font-semibold transition-all ${
                statusFilter === st
                  ? st === 'ERROR'
                    ? 'bg-critical text-white'
                    : 'bg-primary text-white shadow-glow-primary'
                  : 'bg-surface text-muted hover:text-white border border-border'
              }`}
            >
              {st === 'ALL' ? 'ALL TRACES' : st === 'ERROR' ? 'ERRORS ONLY' : 'HEALTHY'}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-3 w-full md:max-w-md">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
            <input
              type="text"
              placeholder="Search trace ID, route, service..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 bg-background border border-border focus:border-primary rounded-xl text-xs text-white font-mono focus:outline-none"
            />
          </div>

          <select
            value={serviceFilter}
            onChange={(e) => setServiceFilter(e.target.value)}
            className="bg-background border border-border focus:border-primary rounded-xl px-3 py-1.5 text-xs text-white font-mono focus:outline-none"
          >
            <option value="">All Services</option>
            <option value="edge-ingress">edge-ingress</option>
            <option value="payments-service">payments-service</option>
            <option value="auth-gateway">auth-gateway</option>
            <option value="worker-runner">worker-runner</option>
            <option value="api-gateway">api-gateway</option>
          </select>
        </div>
      </div>

      {/* Main Split View: Traces List & Interactive Waterfall */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Col: Traces Explorer List */}
        <div className="lg:col-span-5 space-y-3">
          <div className="flex items-center justify-between text-xs font-mono text-muted uppercase tracking-wider px-1">
            <span>Recorded Traces ({filteredTraces.length})</span>
            <span>Latency</span>
          </div>

          <div className="space-y-2.5 max-h-[720px] overflow-y-auto pr-1">
            {isLoading && (
              <div className="py-12 text-center text-muted font-mono text-xs">
                Querying distributed trace index...
              </div>
            )}

            {!isLoading && filteredTraces.length === 0 && (
              <div className="p-8 text-center text-muted text-xs glass-card rounded-2xl border border-border">
                No matching traces found. Click <strong>OTel Ingestion Guide</strong> to send live traces.
              </div>
            )}

            {filteredTraces.map((t) => {
              const isSelected = activeTrace?.trace_id === t.trace_id;
              const isError = t.status === 'ERROR';

              return (
                <div
                  key={t.trace_id}
                  onClick={() => {
                    setSelectedTraceId(t.trace_id);
                    setSelectedSpan(null);
                  }}
                  className={`glass-card rounded-2xl p-4 border transition-all cursor-pointer ${
                    isSelected
                      ? 'border-primary shadow-glow-primary bg-surface-elevated/80'
                      : 'border-border hover:border-border-bright hover:bg-surface-elevated/40'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold border ${
                        isError
                          ? 'bg-critical/15 text-critical border-critical/30'
                          : 'bg-healthy/15 text-healthy border-healthy/30'
                      }`}
                    >
                      {t.status}
                    </span>
                    <span className="text-xs font-mono font-bold text-white">
                      {t.duration_ms.toFixed(1)} ms
                    </span>
                  </div>

                  <div className="mt-2.5">
                    <h4 className="text-sm font-bold text-white truncate">{t.root_operation}</h4>
                    <div className="flex items-center gap-2 text-xs text-muted font-mono mt-1">
                      <span className="text-primary-light font-semibold">{t.root_service}</span>
                      <span>•</span>
                      <span>{t.span_count} Spans</span>
                    </div>
                  </div>

                  <div className="mt-3 pt-2 border-t border-border/60 flex items-center justify-between text-[11px] font-mono text-muted">
                    <span>{new Date(t.start_time).toLocaleTimeString()}</span>
                    <span className="text-text-dim text-[10px] truncate max-w-[140px]">
                      {t.trace_id}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Right Col: Interactive Waterfall Canvas */}
        <div className="lg:col-span-7 space-y-4">
          {activeTrace ? (
            <div className="glass-card rounded-2xl p-6 border border-border shadow-glass space-y-6">
              {/* Waterfall Header */}
              <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3 pb-4 border-b border-border">
                <div className="space-y-1">
                  <div className="flex items-center gap-2 text-xs font-mono text-muted">
                    <span>Trace:</span>
                    <span className="text-white font-bold">{activeTrace.trace_id}</span>
                  </div>
                  <h3 className="text-lg font-bold text-white">{activeTrace.root_operation}</h3>
                  <div className="flex items-center gap-3 text-xs font-mono text-muted">
                    <span>
                      Total Latency: <strong className="text-white">{activeTrace.duration_ms.toFixed(1)} ms</strong>
                    </span>
                    <span>•</span>
                    <span>
                      Spans: <strong className="text-white">{activeTrace.span_count}</strong>
                    </span>
                  </div>
                </div>

                <div className="flex sm:flex-col items-end gap-2 text-right text-xs font-mono text-muted">
                  <div>{new Date(activeTrace.start_time).toLocaleTimeString()}</div>
                  <Link
                    to={`/logs?search=${activeTrace.trace_id}`}
                    className="inline-flex items-center gap-1 text-[11px] text-primary-light hover:underline"
                  >
                    <FileText size={12} />
                    <span>Correlated Logs</span>
                  </Link>
                </div>
              </div>

              {/* Waterfall Timeline Visualization */}
              <div className="space-y-3">
                <div className="flex items-center justify-between text-xs font-mono text-muted uppercase tracking-wider">
                  <span>Microservice Span Hierarchy & Waterfall</span>
                  <span>Timeline (0 → {activeTrace.duration_ms.toFixed(0)}ms)</span>
                </div>

                <div className="space-y-2 font-mono text-xs">
                  {activeTrace.spans.map((span, idx) => {
                    const traceTotal = activeTrace.duration_ms || 1;
                    const spanStartOffset =
                      (new Date(span.start_time).getTime() -
                        new Date(activeTrace.start_time).getTime()) /
                      1;
                    const offsetPct = Math.max(0, Math.min(90, (spanStartOffset / traceTotal) * 100));
                    const widthPct = Math.max(8, Math.min(100 - offsetPct, (span.duration_ms / traceTotal) * 100));
                    const isSpanError = span.status === 'ERROR';
                    const hasParent = !!span.parent_span_id;

                    return (
                      <div
                        key={span.span_id}
                        onClick={() => setSelectedSpan(span)}
                        className={`p-3 rounded-xl border transition-all cursor-pointer group ${
                          selectedSpan?.span_id === span.span_id
                            ? 'bg-surface-elevated border-primary shadow-glow-primary'
                            : 'bg-background/60 border-border/80 hover:bg-surface-elevated/50'
                        } ${hasParent ? 'ml-3 border-l-2' : ''}`}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2 truncate">
                            <span
                              className={`w-2 h-2 rounded-full shrink-0 ${
                                isSpanError ? 'bg-critical' : 'bg-healthy'
                              }`}
                            />
                            <span className="font-bold text-white text-[11px] truncate">
                              {span.service_name}
                            </span>
                            <span className="text-muted text-[11px] truncate">
                              {span.operation}
                            </span>
                          </div>

                          <div className="flex items-center gap-2 text-[11px] shrink-0">
                            {span.status_code && (
                              <span
                                className={`px-1.5 py-0.2 rounded text-[10px] font-mono ${
                                  isSpanError
                                    ? 'bg-critical/20 text-critical border border-critical/30'
                                    : 'bg-healthy/10 text-healthy border border-healthy/20'
                                }`}
                              >
                                {span.status_code}
                              </span>
                            )}
                            <span className="font-bold text-white">
                              {span.duration_ms.toFixed(1)} ms
                            </span>
                          </div>
                        </div>

                        {/* Timing Bar */}
                        <div className="w-full bg-surface-highlight/70 h-2 rounded-full relative overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${
                              isSpanError
                                ? 'bg-critical shadow-glow-critical'
                                : idx === 0
                                ? 'bg-primary'
                                : 'bg-cyan'
                            }`}
                            style={{
                              marginLeft: `${offsetPct}%`,
                              width: `${widthPct}%`,
                            }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Span Attributes Inspector Box */}
              {selectedSpan && (
                <div className="p-4 rounded-xl bg-surface/70 border border-primary/30 space-y-3 animate-in fade-in duration-150">
                  <div className="flex items-center justify-between border-b border-border/60 pb-2">
                    <div className="flex items-center gap-2 text-xs font-mono">
                      <Code size={14} className="text-primary-light" />
                      <span className="text-white font-bold">Span Attributes:</span>
                      <span className="text-primary-light">{selectedSpan.operation}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <Link
                        to={`/logs?search=${selectedSpan.service_name}`}
                        className="text-[11px] font-mono text-primary-light hover:underline flex items-center gap-1"
                      >
                        <FileText size={11} /> Logs
                      </Link>
                      <button
                        onClick={() => setSelectedSpan(null)}
                        className="text-xs text-muted hover:text-white"
                      >
                        <X size={13} />
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-[11px] font-mono">
                    <div className="text-muted">Span ID:</div>
                    <div className="text-white truncate">{selectedSpan.span_id}</div>
                    <div className="text-muted">Parent Span ID:</div>
                    <div className="text-white truncate">{selectedSpan.parent_span_id || 'root'}</div>
                    <div className="text-muted">Microservice:</div>
                    <div className="text-white">{selectedSpan.service_name}</div>
                    <div className="text-muted">Duration:</div>
                    <div className="text-white">{selectedSpan.duration_ms.toFixed(2)} ms</div>
                    {selectedSpan.attributes &&
                      Object.entries(selectedSpan.attributes).map(([k, v]) => (
                        <div key={k} className="contents">
                          <div className="text-muted truncate">{k}:</div>
                          <div className="text-text-dim truncate">{String(v)}</div>
                        </div>
                      ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="glass-card rounded-2xl p-12 text-center text-muted font-mono text-xs">
              Select a trace on the left to inspect microservice waterfall timing.
            </div>
          )}
        </div>
      </div>

      {/* OpenTelemetry Integration Modal */}
      {guideModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in duration-150">
          <div className="bg-surface border border-border rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-surface-highlight/30">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-xl bg-primary/20 text-primary-light border border-primary/30">
                  <GitCommit size={18} />
                </div>
                <div>
                  <h3 className="text-base font-bold text-white">OpenTelemetry Ingestion Setup</h3>
                  <p className="text-xs text-muted">Point your OTel Collector or SDK to SentriX</p>
                </div>
              </div>
              <button
                onClick={() => setGuideModalOpen(false)}
                className="p-1.5 rounded-lg text-muted hover:text-white"
              >
                <X size={18} />
              </button>
            </div>

            <div className="p-6 overflow-y-auto space-y-4">
              <div className="flex gap-2 border-b border-border pb-2">
                {(['curl', 'collector', 'python', 'node', 'go'] as const).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setGuideTab(tab)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-mono font-semibold uppercase transition-all ${
                      guideTab === tab
                        ? 'bg-primary text-white shadow-glow-primary'
                        : 'bg-surface text-muted hover:text-white'
                    }`}
                  >
                    {tab}
                  </button>
                ))}
              </div>

              {guideTab === 'curl' && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs font-mono text-muted">
                    <span>Send test OTLP/HTTP Trace via cURL:</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          `curl -X POST "${serverOrigin}/v1/traces" \\\n  -H "Content-Type: application/json" \\\n  -d '{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"order-service"}}]},"scopeSpans":[{"spans":[{"traceId":"4bf92f3577b34da6a3ce929d0e0e4736","spanId":"00f067aa0ba902b7","name":"HTTP POST /v1/checkout","startTimeUnixNano":"1732381200000000000","endTimeUnixNano":"1732381200120000000","status":{"code":1}}]}]}]}'`,
                          'curl'
                        )
                      }
                      className="text-primary-light flex items-center gap-1 hover:underline"
                    >
                      {copiedCode === 'curl' ? <Check size={12} /> : <Copy size={12} />}
                      <span>{copiedCode === 'curl' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-3.5 rounded-xl bg-background border border-border font-mono text-xs text-emerald-400 overflow-x-auto">
{`curl -X POST "${serverOrigin}/v1/traces" \\
  -H "Content-Type: application/json" \\
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{"key": "service.name", "value": {"stringValue": "order-service"}}]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
          "spanId": "00f067aa0ba902b7",
          "name": "HTTP POST /v1/checkout",
          "startTimeUnixNano": "1732381200000000000",
          "endTimeUnixNano": "1732381200120000000",
          "status": {"code": 1}
        }]
      }]
    }]
  }'`}
                  </pre>
                </div>
              )}

              {guideTab === 'collector' && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs font-mono text-muted">
                    <span>otel-collector-config.yaml exporter:</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          `exporters:\n  otlphttp/sentrix:\n    endpoint: "${serverOrigin}"\n\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [otlphttp/sentrix]`,
                          'collector'
                        )
                      }
                      className="text-primary-light flex items-center gap-1 hover:underline"
                    >
                      {copiedCode === 'collector' ? <Check size={12} /> : <Copy size={12} />}
                      <span>{copiedCode === 'collector' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-3.5 rounded-xl bg-background border border-border font-mono text-xs text-cyan overflow-x-auto">
{`exporters:
  otlphttp/sentrix:
    endpoint: "${serverOrigin}"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/sentrix]`}
                  </pre>
                </div>
              )}

              {guideTab === 'python' && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs font-mono text-muted">
                    <span>Python OpenTelemetry setup:</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          `from opentelemetry import trace\nfrom opentelemetry.sdk.trace import TracerProvider\nfrom opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter\nfrom opentelemetry.sdk.trace.export import BatchSpanProcessor\n\nprovider = TracerProvider()\nprocessor = BatchSpanProcessor(OTLPSpanExporter(endpoint="${serverOrigin}/v1/traces"))\nprovider.add_span_processor(processor)\ntrace.set_tracer_provider(provider)`,
                          'python'
                        )
                      }
                      className="text-primary-light flex items-center gap-1 hover:underline"
                    >
                      {copiedCode === 'python' ? <Check size={12} /> : <Copy size={12} />}
                      <span>{copiedCode === 'python' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-3.5 rounded-xl bg-background border border-border font-mono text-xs text-amber-300 overflow-x-auto">
{`from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace.export import BatchSpanProcessor

provider = TracerProvider()
processor = BatchSpanProcessor(OTLPSpanExporter(endpoint="${serverOrigin}/v1/traces"))
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)`}
                  </pre>
                </div>
              )}

              {guideTab === 'node' && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs font-mono text-muted">
                    <span>Node.js OpenTelemetry setup:</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          `const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');\nconst exporter = new OTLPTraceExporter({ url: '${serverOrigin}/v1/traces' });`,
                          'node'
                        )
                      }
                      className="text-primary-light flex items-center gap-1 hover:underline"
                    >
                      {copiedCode === 'node' ? <Check size={12} /> : <Copy size={12} />}
                      <span>{copiedCode === 'node' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-3.5 rounded-xl bg-background border border-border font-mono text-xs text-purple-300 overflow-x-auto">
{`const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');

const exporter = new OTLPTraceExporter({
  url: '${serverOrigin}/v1/traces'
});`}
                  </pre>
                </div>
              )}

              {guideTab === 'go' && (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs font-mono text-muted">
                    <span>Go OpenTelemetry setup:</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          `exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL("${serverOrigin}/v1/traces"))`,
                          'go'
                        )
                      }
                      className="text-primary-light flex items-center gap-1 hover:underline"
                    >
                      {copiedCode === 'go' ? <Check size={12} /> : <Copy size={12} />}
                      <span>{copiedCode === 'go' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="p-3.5 rounded-xl bg-background border border-border font-mono text-xs text-blue-300 overflow-x-auto">
{`import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

exporter, err := otlptracehttp.New(ctx,
    otlptracehttp.WithEndpointURL("${serverOrigin}/v1/traces"),
)`}
                  </pre>
                </div>
              )}

              <div className="p-4 rounded-xl bg-primary/10 border border-primary/20 space-y-2 text-xs">
                <div className="font-bold text-white flex items-center gap-2">
                  <Send size={14} className="text-primary-light" />
                  <span>Send a Live Test Trace</span>
                </div>
                <p className="text-muted">
                  Click the button below to instantly simulate a multi-span microservice checkout call and view it in real-time on your trace waterfall.
                </p>
                <button
                  onClick={handleSendTestTrace}
                  disabled={sendingTest}
                  className="mt-2 px-4 py-2 rounded-xl bg-primary hover:bg-primary-hover text-white font-semibold text-xs transition-colors flex items-center gap-1.5 shadow-glow-primary"
                >
                  <Zap size={13} />
                  <span>{sendingTest ? 'Sending Trace...' : 'Generate & Send Test Trace'}</span>
                </button>
              </div>
            </div>

            <div className="px-6 py-3 border-t border-border bg-surface-highlight/20 flex justify-end">
              <button
                onClick={() => setGuideModalOpen(false)}
                className="px-4 py-1.5 rounded-xl bg-surface border border-border text-xs font-mono text-white hover:bg-surface-highlight"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
