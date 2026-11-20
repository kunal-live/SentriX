import { ReactNode } from 'react';
import { BrowserRouter, Route, Routes, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { Layout } from './components/Layout';
import { AuthProvider, RequireAuth } from './hooks/useAuth';
import { OverviewPage } from './features/overview/OverviewPage';
import { ServerDetailPage } from './features/servers/ServerDetailPage';
import { AlertRulesPage } from './features/alerts/AlertRulesPage';
import { IncidentsPage } from './features/incidents/IncidentsPage';
import { IncidentDetailPage } from './features/incidents/IncidentDetailPage';
import { ChecksPage } from './features/checks/ChecksPage';
import { UsersPage } from './features/users/UsersPage';
import { NotificationsPage } from './features/notifications/NotificationsPage';
import { AuditLogsPage } from './features/audit/AuditLogsPage';
import { MetricsExplorerPage } from './features/metrics/MetricsExplorerPage';
import { CentralLogsPage } from './features/logs/CentralLogsPage';
import { DashboardsPage } from './features/dashboards/DashboardsPage';
import { DeveloperPlatformPage } from './features/developer/DeveloperPlatformPage';
import { LoginPage } from './features/auth/LoginPage';

const queryClient = new QueryClient();

function Protected({ children }: { children: ReactNode }) {
  return <RequireAuth>{children}</RequireAuth>;
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />

            {/* Core Observability */}
            <Route
              path="/"
              element={
                <Protected>
                  <Layout>
                    <OverviewPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/servers"
              element={
                <Protected>
                  <Layout>
                    <OverviewPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/servers/:serverId"
              element={
                <Protected>
                  <Layout>
                    <ServerDetailPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/dashboards"
              element={
                <Protected>
                  <Layout>
                    <DashboardsPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/metrics"
              element={
                <Protected>
                  <Layout>
                    <MetricsExplorerPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/logs"
              element={
                <Protected>
                  <Layout>
                    <CentralLogsPage />
                  </Layout>
                </Protected>
              }
            />

            {/* Reliability & Alerts */}
            <Route
              path="/incidents"
              element={
                <Protected>
                  <Layout>
                    <IncidentsPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/incidents/:incidentId"
              element={
                <Protected>
                  <Layout>
                    <IncidentDetailPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/alerts"
              element={
                <Protected>
                  <Layout>
                    <AlertRulesPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/checks"
              element={
                <Protected>
                  <Layout>
                    <ChecksPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/notifications"
              element={
                <Protected>
                  <Layout>
                    <NotificationsPage />
                  </Layout>
                </Protected>
              }
            />

            {/* Platform & Security */}
            <Route
              path="/users"
              element={
                <Protected>
                  <Layout>
                    <UsersPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/audit"
              element={
                <Protected>
                  <Layout>
                    <AuditLogsPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/api-keys"
              element={
                <Protected>
                  <Layout>
                    <DeveloperPlatformPage />
                  </Layout>
                </Protected>
              }
            />

            <Route
              path="/api-docs"
              element={
                <Protected>
                  <Layout>
                    <DeveloperPlatformPage />
                  </Layout>
                </Protected>
              }
            />

            {/* Redirects for redundant or mock routes */}
            <Route path="/infrastructure" element={<Navigate to="/" replace />} />
            <Route path="/agents" element={<Navigate to="/" replace />} />
            <Route path="/traces" element={<Navigate to="/dashboards" replace />} />
            <Route path="/service-map" element={<Navigate to="/" replace />} />
            <Route path="/services" element={<Navigate to="/" replace />} />
            <Route path="/containers" element={<Navigate to="/" replace />} />
            <Route path="/kubernetes" element={<Navigate to="/" replace />} />
            <Route path="/runbooks" element={<Navigate to="/" replace />} />
            <Route path="/integrations" element={<Navigate to="/alerts" replace />} />
            <Route path="/security" element={<Navigate to="/audit" replace />} />
            <Route path="/organizations" element={<Navigate to="/" replace />} />
            <Route path="/deployments" element={<Navigate to="/" replace />} />
            <Route path="/changes" element={<Navigate to="/" replace />} />
            <Route path="/scale" element={<Navigate to="/metrics" replace />} />
            <Route path="/capacity" element={<Navigate to="/" replace />} />
            <Route path="/finops" element={<Navigate to="/" replace />} />
            <Route path="/databases" element={<Navigate to="/metrics" replace />} />
            <Route path="/network" element={<Navigate to="/metrics" replace />} />
            <Route path="/slos" element={<Navigate to="/incidents" replace />} />
            <Route path="/compliance" element={<Navigate to="/audit" replace />} />
            <Route path="/intelligence" element={<Navigate to="/dashboards" replace />} />
            <Route path="/billing" element={<Navigate to="/" replace />} />
            <Route path="/plans" element={<Navigate to="/" replace />} />

            {/* Catch-all */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export default App;
