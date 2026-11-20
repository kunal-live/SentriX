package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenTTL = 15 * time.Minute

type AccessTokenClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

var (
	secretOnce   sync.Once
	activeSecret []byte
)

func jwtSecret() []byte {
	secretOnce.Do(func() {
		secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		env := strings.ToLower(os.Getenv("SENTRIX_ENV"))
		if env == "" {
			env = strings.ToLower(os.Getenv("ENV"))
		}
		isProd := (env == "production" || env == "prod")

		insecureDefaults := map[string]bool{
			"":                                  true,
			"change-me-to-a-long-random-string": true,
			"sentrix-dev-insecure-secret":       true,
			"secret":                            true,
			"sentrix":                           true,
		}

		if insecureDefaults[secret] || len(secret) < 32 {
			if isProd {
				slog.Error("FATAL: Insecure, missing, or weak (<32 bytes) JWT_SECRET detected in production environment! Aborting startup.")
				os.Exit(1)
			}

			slog.Warn("JWT_SECRET is unset, weak (<32 chars), or using default placeholder; generating secure 32-byte ephemeral key for session.")
			randBytes := make([]byte, 32)
			if _, err := rand.Read(randBytes); err != nil {
				slog.Error("FATAL: Failed to read from system CSPRNG", "error", err)
				os.Exit(1)
			}
			activeSecret = randBytes
		} else {
			activeSecret = []byte(secret)
		}
	})

	return activeSecret
}

// InitJWT validates the JWT secret configuration at startup, ensuring fail-closed behavior before serving requests.
func InitJWT() {
	_ = jwtSecret()
}

func GenerateAccessToken(userID, email, role string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(accessTokenTTL)

	claims := AccessTokenClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "sentrix",
			Audience:  jwt.ClaimStrings{"sentrix-api"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(jwtSecret())
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected or rejected signing method: %v", t.Header["alg"])
			}

			return jwtSecret(), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("sentrix"),
		jwt.WithAudience("sentrix-api"),
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
