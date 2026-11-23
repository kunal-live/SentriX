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
import { ServiceMapPage } from './features/servicemap/ServiceMapPage';
import { ServicesPage } from './features/services/ServicesPage';
import { InfrastructurePage } from './features/infrastructure/InfrastructurePage';
import { KubernetesPage } from './features/kubernetes/KubernetesPage';
import { ContainersPage } from './features/containers/ContainersPage';
import { AgentsPage } from './features/agents/AgentsPage';
import { IncidentsPage } from './features/incidents/IncidentsPage';
import { IncidentDetailPage } from './features/incidents/IncidentDetailPage';
import { AlertRulesPage } from './features/alerts/AlertRulesPage';
import { ChecksPage } from './features/checks/ChecksPage';
import { SLOPage } from './features/slo/SLOPage';
import { RunbooksPage } from './features/automation/RunbooksPage';
import { DeploymentsPage } from './features/deployments/DeploymentsPage';
import { IntelligencePage } from './features/intelligence/IntelligencePage';
import { DatabasesPage } from './features/databases/DatabasesPage';
import { NetworkPage } from './features/network/NetworkPage';
import { CapacityFinOpsPage } from './features/capacity/CapacityFinOpsPage';
import { ScaleBenchmarksPage } from './features/scale/ScaleBenchmarksPage';
import { SecurityCenterPage } from './features/security/SecurityCenterPage';
import { CompliancePage } from './features/compliance/CompliancePage';
import { AuditLogsPage } from './features/audit/AuditLogsPage';
import { UsersPage } from './features/users/UsersPage';
import { OrganizationsPage } from './features/organizations/OrganizationsPage';
import { IntegrationsPage } from './features/integrations/IntegrationsPage';
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
            <Route path="/servers" element={<Page><OverviewPage /></Page>} />
            <Route path="/servers/:serverId" element={<Page><ServerDetailPage /></Page>} />
            <Route path="/dashboards" element={<Page><DashboardsPage /></Page>} />
            <Route path="/metrics" element={<Page><MetricsExplorerPage /></Page>} />
            <Route path="/logs" element={<Page><CentralLogsPage /></Page>} />
            <Route path="/traces" element={<Page><TracesPage /></Page>} />
            <Route path="/service-map" element={<Page><ServiceMapPage /></Page>} />
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
            <Route path="/slos" element={<Page><SLOPage /></Page>} />
            <Route path="/runbooks" element={<Page><RunbooksPage /></Page>} />
            <Route path="/deployments" element={<Page><DeploymentsPage /></Page>} />
            <Route path="/changes" element={<Page><DeploymentsPage /></Page>} />

            {/* Diagnostics & AI */}
            <Route path="/intelligence" element={<Page><IntelligencePage /></Page>} />
            <Route path="/databases" element={<Page><DatabasesPage /></Page>} />
            <Route path="/network" element={<Page><NetworkPage /></Page>} />
            <Route path="/capacity" element={<Page><CapacityFinOpsPage /></Page>} />
            <Route path="/finops" element={<Page><CapacityFinOpsPage /></Page>} />
            <Route path="/scale" element={<Page><ScaleBenchmarksPage /></Page>} />

            {/* Platform, Security & Governance */}
            <Route path="/security" element={<Page><SecurityCenterPage /></Page>} />
            <Route path="/compliance" element={<Page><CompliancePage /></Page>} />
            <Route path="/audit" element={<Page><AuditLogsPage /></Page>} />
            <Route path="/users" element={<Page><UsersPage /></Page>} />
            <Route path="/organizations" element={<Page><OrganizationsPage /></Page>} />
            <Route path="/integrations" element={<Page><IntegrationsPage /></Page>} />
            <Route path="/notifications" element={<Page><NotificationsPage /></Page>} />
            <Route path="/api-keys" element={<Page><DeveloperPlatformPage /></Page>} />
            <Route path="/api-docs" element={<Page><DeveloperPlatformPage /></Page>} />

            {/* Legacy aliases */}
            <Route path="/billing" element={<Navigate to="/organizations" replace />} />
            <Route path="/plans" element={<Navigate to="/organizations" replace />} />

            {/* Catch-all */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export default App;
