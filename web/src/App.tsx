import { ReactNode } from 'react';
import { BrowserRouter, Route, Routes, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { Layout } from './components/Layout';
import { AuthProvider, RequireAuth } from './hooks/useAuth';
import { OverviewPage } from './features/overview/OverviewPage';
import { ServerDetailPage } from './features/servers/ServerDetailPage';
import { DashboardsPage } from './features/dashboards/DashboardsPage';
import { MetricsExplorerPage } from './features/metrics/MetricsExplorerPage';
import { CentralLogsPage } from './features/logs/CentralLogsPage';
import { TracesPage } from './features/traces/TracesPage';
import { ServicesPage } from './features/services/ServicesPage';
import { InfrastructurePage } from './features/infrastructure/InfrastructurePage';
import { KubernetesPage } from './features/kubernetes/KubernetesPage';
import { ContainersPage } from './features/containers/ContainersPage';
import { AgentsPage } from './features/agents/AgentsPage';
import { IncidentsPage } from './features/incidents/IncidentsPage';
import { IncidentDetailPage } from './features/incidents/IncidentDetailPage';
import { AlertRulesPage } from './features/alerts/AlertRulesPage';
import { ChecksPage } from './features/checks/ChecksPage';
import { RunbooksPage } from './features/automation/RunbooksPage';
import { DatabasesPage } from './features/databases/DatabasesPage';
import { NetworkPage } from './features/network/NetworkPage';
import { SecurityCenterPage } from './features/security/SecurityCenterPage';
import { AuditLogsPage } from './features/audit/AuditLogsPage';
import { UsersPage } from './features/users/UsersPage';
import { NotificationsPage } from './features/notifications/NotificationsPage';
import { DeveloperPlatformPage } from './features/developer/DeveloperPlatformPage';
import { LoginPage } from './features/auth/LoginPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function Protected({ children }: { children: ReactNode }) {
  return <RequireAuth>{children}</RequireAuth>;
}

function Page({ children }: { children: ReactNode }) {
  return (
    <Protected>
      <Layout>{children}</Layout>
    </Protected>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />

            {/* Observability & Topology */}
            <Route path="/" element={<Page><OverviewPage /></Page>} />
            <Route path="/servers" element={<Navigate to="/" replace />} />
            <Route path="/servers/:serverId" element={<Page><ServerDetailPage /></Page>} />
            <Route path="/dashboards" element={<Page><DashboardsPage /></Page>} />
            <Route path="/metrics" element={<Page><MetricsExplorerPage /></Page>} />
            <Route path="/logs" element={<Page><CentralLogsPage /></Page>} />
            <Route path="/traces" element={<Page><TracesPage /></Page>} />
            <Route path="/services" element={<Page><ServicesPage /></Page>} />
            <Route path="/infrastructure" element={<Page><InfrastructurePage /></Page>} />

            {/* Cloud & Workloads */}
            <Route path="/kubernetes" element={<Page><KubernetesPage /></Page>} />
            <Route path="/containers" element={<Page><ContainersPage /></Page>} />
            <Route path="/agents" element={<Page><AgentsPage /></Page>} />

            {/* Reliability & Operations */}
            <Route path="/incidents" element={<Page><IncidentsPage /></Page>} />
            <Route path="/incidents/:incidentId" element={<Page><IncidentDetailPage /></Page>} />
            <Route path="/alerts" element={<Page><AlertRulesPage /></Page>} />
            <Route path="/checks" element={<Page><ChecksPage /></Page>} />
            <Route path="/runbooks" element={<Page><RunbooksPage /></Page>} />

            {/* Diagnostics & AI */}
            <Route path="/databases" element={<Page><DatabasesPage /></Page>} />
            <Route path="/network" element={<Page><NetworkPage /></Page>} />

            {/* Platform, Security & Governance */}
            <Route path="/security" element={<Page><SecurityCenterPage /></Page>} />
            <Route path="/audit" element={<Page><AuditLogsPage /></Page>} />
            <Route path="/users" element={<Page><UsersPage /></Page>} />
            <Route path="/notifications" element={<Page><NotificationsPage /></Page>} />
            <Route path="/api-keys" element={<Page><DeveloperPlatformPage /></Page>} />
            <Route path="/api-docs" element={<Navigate to="/api-keys" replace />} />

            {/* Catch-all */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export default App;
