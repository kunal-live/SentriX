package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func insertRefreshToken(ctx context.Context, tx pgx.Tx, userID string) (string, string, error) {
	rawBytes := make([]byte, 32)

	_, err := rand.Read(rawBytes)
	if err != nil {
		return "", "", err
	}

	raw := base64.RawURLEncoding.EncodeToString(rawBytes)
	hash := hashRefreshToken(raw)

	id := uuid.New()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (
			id,
			user_id,
			token_hash,
			expires_at
		) VALUES (
			$1, $2, $3, $4
		)
	`, id, userID, hash, expiresAt)

	if err != nil {
		return "", "", err
	}

	return raw, id.String(), nil
}

func setRefreshTokenCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     "sentrix_refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// POST /api/v1/auth/login
func HandleLogin(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		if err := CheckLoginLocked(ctx, pool, req.Email); err != nil {
			http.Error(w, "account temporarily locked due to consecutive failed logins; please try again later", http.StatusTooManyRequests)
			return
		}

		var userID, email, role, passwordHash, status string

		err := pool.QueryRow(ctx, `
			SELECT id, email, role, password_hash, status
			FROM users
			WHERE lower(email) = lower($1)
		`, req.Email).Scan(&userID, &email, &role, &passwordHash, &status)

		if err != nil {
			_ = RecordLoginFailure(ctx, pool, req.Email)
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		if status != "ACTIVE" {
			http.Error(w, "account is disabled", http.StatusUnauthorized)
			return
		}

		valid, err := VerifyPassword(req.Password, passwordHash)
		if err != nil || !valid {
			_ = RecordLoginFailure(ctx, pool, req.Email)
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		_ = ResetLoginFailures(ctx, pool, req.Email)

		tx, err := pool.Begin(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		refreshToken, _, err := insertRefreshToken(ctx, tx, userID)
		if err != nil {
			http.Error(w, "failed to create refresh token", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		accessToken, expiresAt, err := GenerateAccessToken(userID, email, role)
		if err != nil {
			http.Error(w, "failed to generate access token", http.StatusInternalServerError)
			return
		}

		_, _ = pool.Exec(ctx, `
			UPDATE users
			SET last_login_at = NOW()
			WHERE id = $1
		`, userID)

		WriteAudit(
			ctx,
			pool,
			&userID,
			"auth.login",
			"user",
			userID,
			ClientIP(r),
			map[string]any{
				"email": email,
			},
		)

		resp := authResponse{
			User: userResponse{
				ID:    userID,
				Email: email,
				Role:  role,
			},
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
		}

		setRefreshTokenCookie(w, r, refreshToken, 7*86400)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// POST /api/v1/auth/refresh
func HandleRefresh(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.RefreshToken == "" {
			if cookie, err := r.Cookie("sentrix_refresh_token"); err == nil {
				req.RefreshToken = cookie.Value
			}
		}

		if req.RefreshToken == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		tokenHash := hashRefreshToken(req.RefreshToken)

		tx, err := pool.Begin(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		var oldTokenID, userID string
		var expiresAt time.Time
		var revokedAt *time.Time

		err = tx.QueryRow(ctx, `
			SELECT id, user_id, expires_at, revoked_at
			FROM refresh_tokens
			WHERE token_hash = $1
			FOR UPDATE
		`, tokenHash).Scan(&oldTokenID, &userID, &expiresAt, &revokedAt)

		if err != nil {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}

		if revokedAt != nil {
			http.Error(w, "refresh token revoked", http.StatusUnauthorized)
			return
		}

		if expiresAt.Before(time.Now()) {
			http.Error(w, "refresh token expired", http.StatusUnauthorized)
			return
		}

		var email, role, status string

		err = tx.QueryRow(ctx, `
			SELECT email, role, status
			FROM users
			WHERE id = $1
		`, userID).Scan(&email, &role, &status)

		if err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		if status != "ACTIVE" {
			http.Error(w, "account is disabled", http.StatusUnauthorized)
			return
		}

		newRefreshToken, newRefreshTokenID, err := insertRefreshToken(ctx, tx, userID)
		if err != nil {
			http.Error(w, "failed to create refresh token", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(ctx, `
			UPDATE refresh_tokens
			SET
				revoked_at = NOW(),
				replaced_by_token_id = $2
			WHERE id = $1
		`, oldTokenID, newRefreshTokenID)

		if err != nil {
			http.Error(w, "failed to rotate refresh token", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		accessToken, expiresAccessToken, err := GenerateAccessToken(userID, email, role)
		if err != nil {
			http.Error(w, "failed to generate access token", http.StatusInternalServerError)
			return
		}

		resp := authResponse{
			User: userResponse{
				ID:    userID,
				Email: email,
				Role:  role,
			},
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    int64(time.Until(expiresAccessToken).Seconds()),
		}

		setRefreshTokenCookie(w, r, newRefreshToken, 7*86400)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// POST /api/v1/auth/logout
func HandleLogout(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.RefreshToken == "" {
			if cookie, err := r.Cookie("sentrix_refresh_token"); err == nil {
				req.RefreshToken = cookie.Value
			}
		}

		if req.RefreshToken == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}

		tokenHash := hashRefreshToken(req.RefreshToken)

		_, err := pool.Exec(r.Context(), `
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE token_hash = $1
			  AND revoked_at IS NULL
		`, tokenHash)

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		setRefreshTokenCookie(w, r, "", -1)
		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /api/v1/auth/me
func HandleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)

		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		})
	}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// POST /api/v1/auth/password
func HandleChangePassword(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := GetUser(r)
		if actor == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req changePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if req.CurrentPassword == "" || req.NewPassword == "" {
			http.Error(w, "current_password and new_password are required", http.StatusBadRequest)
			return
		}

		if len(req.NewPassword) < 8 {
			http.Error(w, "new_password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var currentHash string
		err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, actor.ID).Scan(&currentHash)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		valid, err := VerifyPassword(req.CurrentPassword, currentHash)
		if err != nil || !valid {
			http.Error(w, "current password does not match", http.StatusUnauthorized)
			return
		}

		newHash, err := HashPassword(req.NewPassword)
		if err != nil {
			http.Error(w, "failed to hash new password", http.StatusInternalServerError)
			return
		}

		_, err = pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, newHash, actor.ID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		WriteAudit(
			ctx,
			pool,
			&actor.ID,
			"auth.password_change",
			"user",
			actor.ID,
			ClientIP(r),
			map[string]any{"email": actor.Email},
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"password updated successfully"}`))
	}
}
