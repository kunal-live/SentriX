package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func BootstrapAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	var userCount int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return nil
	}

	email := os.Getenv("SENTRIX_ADMIN_EMAIL")
	if email == "" {
		email = "admin@sentrix.local"
	}

	password := os.Getenv("SENTRIX_ADMIN_PASSWORD")
	isGenerated := false

	if password == "" {
		genPass, err := generateRandomPassword(20)
		if err != nil {
			return fmt.Errorf("failed to generate random admin password: %w", err)
		}
		password = genPass
		isGenerated = true
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	id := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO users (
			id,
			email,
			password_hash,
			role,
			status
		) VALUES (
			$1, lower($2), $3, 'ADMIN', 'ACTIVE'
		)
	`, id, email, passwordHash)

	if err != nil {
		return err
	}

	if isGenerated {
		slog.Warn("================================================================================")
		slog.Warn("FIRST-RUN ADMIN USER CREATED WITH SECURE RANDOM PASSWORD")
		slog.Warn("Please save these credentials now and change the password upon first login:")
		slog.Warn("Admin Email:    " + email)
		slog.Warn("Admin Password: " + password)
		slog.Warn("================================================================================")
	} else {
		slog.Info("Bootstrap admin user created", "email", email)
	}

	return nil
}
