package metrics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxReportedProcesses = 50

type agentProcess struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	User        string  `json:"user"`
	CPU         float64 `json:"cpu"`
	MemRSSBytes float64 `json:"mem_rss_bytes"`
	State       string  `json:"state"`
}

type processSnapshot struct {
	Total     int            `json:"total"`
	Processes []agentProcess `json:"processes"`
}

// HandleAgentProcesses stores the agent's latest top-process snapshot.
// POST /api/v1/agent/processes (agent credential)
func HandleAgentProcesses(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID, ok := AgentIDFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var snap processSnapshot
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&snap); err != nil {
			http.Error(w, "invalid process snapshot", http.StatusBadRequest)
			return
		}
		if len(snap.Processes) > maxReportedProcesses {
			snap.Processes = snap.Processes[:maxReportedProcesses]
		}
		payload, err := json.Marshal(snap.Processes)
		if err != nil {
			http.Error(w, "invalid process snapshot", http.StatusBadRequest)
			return
		}

		tag, err := pool.Exec(r.Context(), `
			INSERT INTO server_process_snapshots (server_id, collected_at, total, processes)
			SELECT id, NOW(), $2, $3 FROM servers WHERE agent_id = $1
			ON CONFLICT (server_id) DO UPDATE
			SET collected_at = EXCLUDED.collected_at,
			    total = EXCLUDED.total,
			    processes = EXCLUDED.processes
		`, agentID, snap.Total, payload)
		if err != nil {
			slog.Error("failed to store process snapshot", "agent_id", agentID, "error", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if tag.RowsAffected() == 0 {
			http.Error(w, "server not found for agent", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type processView struct {
	PID       int     `json:"pid"`
	Name      string  `json:"name"`
	User      string  `json:"user"`
	CPU       float64 `json:"cpu"`
	MemoryRSS string  `json:"memory_rss"`
	State     string  `json:"state"`
}

// HandleServerProcesses returns the latest process snapshot for the server page.
// GET /api/v1/servers/{serverID}/processes
func HandleServerProcesses(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID := chi.URLParam(r, "serverID")
		if _, err := uuid.Parse(serverID); err != nil {
			http.Error(w, "invalid server ID format", http.StatusBadRequest)
			return
		}

		var (
			collectedAt time.Time
			total       int
			raw         []byte
		)
		err := pool.QueryRow(r.Context(), `
			SELECT collected_at, total, processes
			FROM server_process_snapshots WHERE server_id = $1
		`, serverID).Scan(&collectedAt, &total, &raw)

		resp := map[string]any{"count": 0, "processes": []processView{}, "collected_at": nil}
		if err == pgx.ErrNoRows {
			// No snapshot yet: the agent hasn't reported (or is an older version).
			writeJSON(w, resp)
			return
		}
		if err != nil {
			slog.Error("failed to load process snapshot", "server_id", serverID, "error", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		var procs []agentProcess
		_ = json.Unmarshal(raw, &procs)
		views := make([]processView, 0, len(procs))
		for _, p := range procs {
			views = append(views, processView{
				PID:       p.PID,
				Name:      p.Name,
				User:      p.User,
				CPU:       p.CPU,
				MemoryRSS: formatBytes(p.MemRSSBytes),
				State:     p.State,
			})
		}
		resp["count"] = total
		resp["processes"] = views
		resp["collected_at"] = collectedAt
		writeJSON(w, resp)
	}
}

func formatBytes(b float64) string {
	const unit = 1024.0
	switch {
	case b >= unit*unit*unit:
		return fmt.Sprintf("%.1f GB", b/(unit*unit*unit))
	case b >= unit*unit:
		return fmt.Sprintf("%.1f MB", b/(unit*unit))
	case b >= unit:
		return fmt.Sprintf("%.0f KB", b/unit)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
