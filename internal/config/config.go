// Configuration loader for the gRPC server.
package config

import (
	"os"
	"strings"
	"time"
)

// Config
type Config struct {
	GRPCAddr          string        // server listens, e.g. ":8090"
	DBDSN             string        // PostgreSQL DSN
	JWTPrivateKeyPath string        // path to RSA private key (PEM)
	JWTPublicKeyPath  string        // path to RSA public key (PEM)
	AccessTTL         time.Duration // access token lifetime
	RefreshTTL        time.Duration // refresh token TTL
	RunMigrations     bool          // apply embedded DB migrations on startup
}

func FromEnv() Config {
	return Config{
		GRPCAddr:          getstr("GK_GRPC_ADDR", ":8090"),
		DBDSN:             firstNonEmpty(os.Getenv("GK_DB_DSN"), os.Getenv("DATABASE_URL"), "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable"),
		JWTPrivateKeyPath: getstr("GK_JWT_PRIVATE_KEY_PATH", "jwt/private.pem"),
		JWTPublicKeyPath:  getstr("GK_JWT_PUBLIC_KEY_PATH", ""),
		AccessTTL:         getdur("GK_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:        getdur("GK_REFRESH_TTL", 72*time.Hour), // 3d
		RunMigrations:     getbool("GK_RUN_MIGRATIONS", false),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func getstr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// getdur parses a time.Duration from env or returns fallback.
func getdur(k string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// getbool parses a boolean flag from env or returns fallback.
func getbool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return def
	}
}
