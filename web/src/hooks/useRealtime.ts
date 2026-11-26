import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { API_BASE, mutateJSON } from '../lib/api';

export type RealtimeStatus = 'connecting' | 'connected' | 'disconnected';

const WS_URL = `${API_BASE.replace(/^http/, 'ws')}/ws`;

// A connection must stay open this long before the reconnect backoff resets.
const STABLE_CONNECTION_MS = 30_000;
const MAX_BACKOFF_MS = 30_000;

interface WsTicketResponse {
  ticket: string;
  expires_in: number;
}

interface RealtimeEvent {
  type: string;
  timestamp: string;
  payload?: Record<string, any>;
}

export function useRealtime(): RealtimeStatus {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<RealtimeStatus>('disconnected');

  useEffect(() => {
    let disposed = false;
    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let attempts = 0;

    function invalidateForEvent(event: RealtimeEvent) {
      switch (event.type) {
        case 'dashboard.updated':
          queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });
          queryClient.invalidateQueries({ queryKey: ['servers'] });
          queryClient.invalidateQueries({ queryKey: ['incidents'] });
          queryClient.invalidateQueries({ queryKey: ['checks'] });
          break;

        case 'server.updated':
        case 'server.telemetry':
          queryClient.invalidateQueries({ queryKey: ['servers'] });
          queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });

          if (event.payload?.server_id) {
            queryClient.invalidateQueries({
              queryKey: ['server-metrics', event.payload.server_id],
            });
          }
          break;

        case 'incident.updated':
          queryClient.invalidateQueries({ queryKey: ['incidents'] });
          queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });
          break;

        case 'check.updated':
          queryClient.invalidateQueries({ queryKey: ['checks'] });
          queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });
          break;

        case 'alert.state_changed':
          queryClient.invalidateQueries({ queryKey: ['incidents'] });
          queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });
          break;

        default:
          break;
      }
    }

    function scheduleReconnect() {
      if (disposed) return;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      setStatus('disconnected');
      // Exponential backoff with jitter, so many tabs don't reconnect in lockstep.
      const base = Math.min(1000 * 2 ** attempts, MAX_BACKOFF_MS);
      const delay = base / 2 + Math.random() * (base / 2);
      attempts += 1;
      reconnectTimer = setTimeout(connect, delay);
    }

    async function connect() {
      if (disposed) {
        return;
      }

      setStatus('connecting');

      try {
        const ticketResponse = await mutateJSON<WsTicketResponse>(
          '/auth/ws-ticket',
          'POST'
        );

        if (disposed) {
          return;
        }

        const url = `${WS_URL}?ticket=${encodeURIComponent(ticketResponse.ticket)}`;
        ws = new WebSocket(url);
        let openedAt = 0;

        ws.onopen = () => {
          if (disposed) {
            return;
          }

          // Don't reset the backoff here: a socket the server closes right after
          // the handshake would then reconnect every second forever. The reset
          // happens in onclose, only if the connection stayed up long enough.
          openedAt = Date.now();
          setStatus('connected');
        };

        ws.onmessage = (messageEvent) => {
          try {
            const parsed = JSON.parse(messageEvent.data) as RealtimeEvent;
            invalidateForEvent(parsed);
          } catch {
            // Ignore malformed realtime messages.
          }
        };

        ws.onclose = (closeEvent) => {
          if (disposed) {
            return;
          }
          const uptime = openedAt ? Date.now() - openedAt : 0;
          if (uptime >= STABLE_CONNECTION_MS) {
            attempts = 0;
          }
          // Logged so the cause of server-side closes can be diagnosed.
          console.debug(
            `[realtime] socket closed code=${closeEvent.code} reason="${closeEvent.reason}" uptime=${uptime}ms`
          );
          scheduleReconnect();
        };

        ws.onerror = () => {
          // WebSocket specification triggers onclose automatically upon transport error
        };
      } catch {
        if (disposed) {
          return;
        }
        scheduleReconnect();
      }
    }

    connect();

    return () => {
      disposed = true;

      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }

      if (ws) {
        ws.onclose = null;
        ws.close();
      }
    };
  }, [queryClient]);

  return status;
}
