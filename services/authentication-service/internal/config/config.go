package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GRPCAddr         string
	DatabaseURL      string
	JWTIssuer        string
	JWTAudience      string
	JWTSecret        string
	AccessTokenTTL   time.Duration
	AllowAutoMigrate bool
}

func Load() Config {
	ttlMinutes := envInt("AUTH_ACCESS_TOKEN_TTL_MINUTES", 15)
	return Config{
		GRPCAddr:         env("AUTH_GRPC_ADDR", ":50052"),
		DatabaseURL:      databaseURL(),
		JWTIssuer:        env("AUTH_JWT_ISSUER", "logisynapse-auth"),
		JWTAudience:      env("AUTH_JWT_AUDIENCE", "logisynapse-gateway"),
		JWTSecret:        env("AUTH_JWT_SECRET", "dev-only-change-me"),
		AccessTokenTTL:   time.Duration(ttlMinutes) * time.Minute,
		AllowAutoMigrate: envBool("AUTH_AUTO_MIGRATE", true),
	}
}

func databaseURL() string {
	if v := os.Getenv("AUTH_DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	user := env("DB_USER", "postgres")
	password := env("DB_PASSWORD", "yourpassword")
	host := env("DB_HOST", "localhost")
	port := env("DB_PORT", "5432")
	name := env("DB_NAME", "logisynapse")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cleanEnv(user), cleanEnv(password), cleanEnv(host), cleanEnv(port), cleanEnv(name))
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return cleanEnv(v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(env(key, ""))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func cleanEnv(v string) string {
	if idx := strings.Index(v, "#"); idx >= 0 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}
