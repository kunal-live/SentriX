package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/security"
)

// sessionView is one live login session: the current (unrevoked, unexpired)
// refresh token of a rotation chain. Field names match the Security Center UI.
type sessionView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Role      string    `json:"role"`
	Device    string    `json:"device"`
	IPAddress string    `json:"ip_address"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// HandleListSessions lists real login sessions from refresh_tokens.
// ADMINs see every session; other users see only their own.
// Without a database (demo mode) it falls back to the demo session list.
func HandleListSessions(pool *pgxpool.Pool) http.HandlerFunc {
	if pool == nil {
		return security.HandleListSessions(pool)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		actor := GetUser(r)
		if actor == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT rt.id, u.id, u.email, u.role, rt.created_at
			FROM refresh_tokens rt
			JOIN users u ON u.id = rt.user_id
			WHERE rt.revoked_at IS NULL
			  AND rt.expires_at > NOW()
			  AND ($1 OR rt.user_id = $2::uuid)
			ORDER BY rt.created_at DESC
			LIMIT 200
		`, actor.Role == "ADMIN", actor.ID)
		if err != nil {
			slog.Error("failed to list sessions", "error", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := []sessionView{}
		for rows.Next() {
			var s sessionView
			if err := rows.Scan(&s.ID, &s.UserID, &s.UserEmail, &s.Role, &s.CreatedAt); err != nil {
				slog.Error("failed to scan session", "error", err)
				continue
			}
			// Refresh tokens rotate on use, so the newest token's creation time is
			// the last time this session was active.
			s.LastSeen = s.CreatedAt
			s.Device = "Unknown"
			s.IPAddress = "Unknown"
			s.Location = "Unknown"
			out = append(out, s)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// HandleRevokeSession revokes a real refresh token so the session can no longer
// be refreshed. ADMINs may revoke any session; other users only their own.
// The short-lived access token (15 min) stays valid until it expires.
func HandleRevokeSession(pool *pgxpool.Pool) http.HandlerFunc {
	if pool == nil {
		return security.HandleRevokeSession(pool)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		actor := GetUser(r)
		if actor == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")
		if _, err := uuid.Parse(id); err != nil {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}

		var ownerID string
		err := pool.QueryRow(r.Context(), `
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE id = $1
			  AND revoked_at IS NULL
			  AND ($2 OR user_id = $3::uuid)
			RETURNING user_id
		`, id, actor.Role == "ADMIN", actor.ID).Scan(&ownerID)
		if err != nil {
			// Not found, already revoked, or not the caller's session.
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		actorID := actor.ID
		WriteAudit(r.Context(), pool, &actorID, "session.revoke", "session", id, ClientIP(r),
			map[string]any{"session_user_id": ownerID})

		w.WriteHeader(http.StatusNoContent)
	}
}
