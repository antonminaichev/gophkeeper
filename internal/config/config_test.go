package config

import (
	"os"
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"GK_GRPC_ADDR":            os.Getenv("GK_GRPC_ADDR"),
		"GK_DB_DSN":               os.Getenv("GK_DB_DSN"),
		"GK_JWT_PRIVATE_KEY_PATH": os.Getenv("GK_JWT_PRIVATE_KEY_PATH"),
		"GK_JWT_PUBLIC_KEY_PATH":  os.Getenv("GK_JWT_PUBLIC_KEY_PATH"),
		"GK_ACCESS_TTL":           os.Getenv("GK_ACCESS_TTL"),
		"GK_REFRESH_TTL":          os.Getenv("GK_REFRESH_TTL"),
		"GK_RUN_MIGRATIONS":       os.Getenv("GK_RUN_MIGRATIONS"),
		"GK_TLS_ENABLE":           os.Getenv("GK_TLS_ENABLE"),
		"GK_TLS_CERT_PATH":        os.Getenv("GK_TLS_CERT_PATH"),
		"GK_TLS_KEY_PATH":         os.Getenv("GK_TLS_KEY_PATH"),
		"GK_TLS_CLIENT_CA_PATH":   os.Getenv("GK_TLS_CLIENT_CA_PATH"),
		"GK_TLS_MIN_VERSION_12":   os.Getenv("GK_TLS_MIN_VERSION_12"),
		"DATABASE_URL":            os.Getenv("DATABASE_URL"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear environment
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	config := FromEnv()

	// Test default values
	if config.GRPCAddr != ":8090" {
		t.Errorf("FromEnv() GRPCAddr = %v, want :8090", config.GRPCAddr)
	}

	if config.DBDSN != "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" {
		t.Errorf("FromEnv() DBDSN = %v, want default postgres DSN", config.DBDSN)
	}

	if config.JWTPrivateKeyPath != "../private.pem" {
		t.Errorf("FromEnv() JWTPrivateKeyPath = %v, want ../private.pem", config.JWTPrivateKeyPath)
	}

	if config.AccessTTL != 15*time.Minute {
		t.Errorf("FromEnv() AccessTTL = %v, want 15m", config.AccessTTL)
	}

	if config.RefreshTTL != 720*time.Hour {
		t.Errorf("FromEnv() RefreshTTL = %v, want 720h", config.RefreshTTL)
	}

	if config.RunMigrations != false {
		t.Errorf("FromEnv() RunMigrations = %v, want false", config.RunMigrations)
	}

	if config.TLSEnable != false {
		t.Errorf("FromEnv() TLSEnable = %v, want false", config.TLSEnable)
	}

	if config.TLSMinVersion12 != true {
		t.Errorf("FromEnv() TLSMinVersion12 = %v, want true", config.TLSMinVersion12)
	}
}

func TestFromEnvWithValues(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"GK_GRPC_ADDR":            os.Getenv("GK_GRPC_ADDR"),
		"GK_DB_DSN":               os.Getenv("GK_DB_DSN"),
		"GK_JWT_PRIVATE_KEY_PATH": os.Getenv("GK_JWT_PRIVATE_KEY_PATH"),
		"GK_ACCESS_TTL":           os.Getenv("GK_ACCESS_TTL"),
		"GK_REFRESH_TTL":          os.Getenv("GK_REFRESH_TTL"),
		"GK_RUN_MIGRATIONS":       os.Getenv("GK_RUN_MIGRATIONS"),
		"GK_TLS_ENABLE":           os.Getenv("GK_TLS_ENABLE"),
		"DATABASE_URL":            os.Getenv("DATABASE_URL"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test values
	os.Setenv("GK_GRPC_ADDR", ":9090")
	os.Setenv("GK_DB_DSN", "postgres://test:test@localhost:5432/test")
	os.Setenv("GK_JWT_PRIVATE_KEY_PATH", "/path/to/key.pem")
	os.Setenv("GK_ACCESS_TTL", "30m")
	os.Setenv("GK_REFRESH_TTL", "48h")
	os.Setenv("GK_RUN_MIGRATIONS", "true")
	os.Setenv("GK_TLS_ENABLE", "true")

	config := FromEnv()

	if config.GRPCAddr != ":9090" {
		t.Errorf("FromEnv() GRPCAddr = %v, want :9090", config.GRPCAddr)
	}

	if config.DBDSN != "postgres://test:test@localhost:5432/test" {
		t.Errorf("FromEnv() DBDSN = %v, want postgres://test:test@localhost:5432/test", config.DBDSN)
	}

	if config.JWTPrivateKeyPath != "/path/to/key.pem" {
		t.Errorf("FromEnv() JWTPrivateKeyPath = %v, want /path/to/key.pem", config.JWTPrivateKeyPath)
	}

	if config.AccessTTL != 30*time.Minute {
		t.Errorf("FromEnv() AccessTTL = %v, want 30m", config.AccessTTL)
	}

	if config.RefreshTTL != 48*time.Hour {
		t.Errorf("FromEnv() RefreshTTL = %v, want 48h", config.RefreshTTL)
	}

	if config.RunMigrations != true {
		t.Errorf("FromEnv() RunMigrations = %v, want true", config.RunMigrations)
	}

	if config.TLSEnable != true {
		t.Errorf("FromEnv() TLSEnable = %v, want true", config.TLSEnable)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name string
		vals []string
		want string
	}{
		{
			name: "first non-empty",
			vals: []string{"", "hello", "world"},
			want: "hello",
		},
		{
			name: "all empty",
			vals: []string{"", "", ""},
			want: "",
		},
		{
			name: "single non-empty",
			vals: []string{"hello"},
			want: "hello",
		},
		{
			name: "single empty",
			vals: []string{""},
			want: "",
		},
		{
			name: "whitespace only",
			vals: []string{"   ", "hello"},
			want: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.vals...)
			if got != tt.want {
				t.Errorf("firstNonEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetstr(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_VAR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_VAR")
		} else {
			os.Setenv("TEST_VAR", originalValue)
		}
	}()

	tests := []struct {
		name   string
		envVal string
		defVal string
		want   string
	}{
		{
			name:   "environment variable set",
			envVal: "env_value",
			defVal: "default_value",
			want:   "env_value",
		},
		{
			name:   "environment variable not set",
			envVal: "",
			defVal: "default_value",
			want:   "default_value",
		},
		{
			name:   "environment variable whitespace",
			envVal: "   ",
			defVal: "default_value",
			want:   "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal == "" {
				os.Unsetenv("TEST_VAR")
			} else {
				os.Setenv("TEST_VAR", tt.envVal)
			}

			got := getstr("TEST_VAR", tt.defVal)
			if got != tt.want {
				t.Errorf("getstr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetdur(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_DURATION")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_DURATION")
		} else {
			os.Setenv("TEST_DURATION", originalValue)
		}
	}()

	tests := []struct {
		name   string
		envVal string
		defVal time.Duration
		want   time.Duration
	}{
		{
			name:   "valid duration",
			envVal: "30m",
			defVal: 15 * time.Minute,
			want:   30 * time.Minute,
		},
		{
			name:   "invalid duration",
			envVal: "invalid",
			defVal: 15 * time.Minute,
			want:   15 * time.Minute,
		},
		{
			name:   "empty value",
			envVal: "",
			defVal: 15 * time.Minute,
			want:   15 * time.Minute,
		},
		{
			name:   "whitespace value",
			envVal: "   ",
			defVal: 15 * time.Minute,
			want:   15 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal == "" {
				os.Unsetenv("TEST_DURATION")
			} else {
				os.Setenv("TEST_DURATION", tt.envVal)
			}

			got := getdur("TEST_DURATION", tt.defVal)
			if got != tt.want {
				t.Errorf("getdur() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetbool(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_BOOL")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_BOOL")
		} else {
			os.Setenv("TEST_BOOL", originalValue)
		}
	}()

	tests := []struct {
		name   string
		envVal string
		defVal bool
		want   bool
	}{
		{
			name:   "true values",
			envVal: "true",
			defVal: false,
			want:   true,
		},
		{
			name:   "1",
			envVal: "1",
			defVal: false,
			want:   true,
		},
		{
			name:   "t",
			envVal: "t",
			defVal: false,
			want:   true,
		},
		{
			name:   "yes",
			envVal: "yes",
			defVal: false,
			want:   true,
		},
		{
			name:   "y",
			envVal: "y",
			defVal: false,
			want:   true,
		},
		{
			name:   "on",
			envVal: "on",
			defVal: false,
			want:   true,
		},
		{
			name:   "false values",
			envVal: "false",
			defVal: true,
			want:   false,
		},
		{
			name:   "0",
			envVal: "0",
			defVal: true,
			want:   false,
		},
		{
			name:   "f",
			envVal: "f",
			defVal: true,
			want:   false,
		},
		{
			name:   "no",
			envVal: "no",
			defVal: true,
			want:   false,
		},
		{
			name:   "n",
			envVal: "n",
			defVal: true,
			want:   false,
		},
		{
			name:   "off",
			envVal: "off",
			defVal: true,
			want:   false,
		},
		{
			name:   "invalid value",
			envVal: "invalid",
			defVal: true,
			want:   true,
		},
		{
			name:   "empty value",
			envVal: "",
			defVal: true,
			want:   true,
		},
		{
			name:   "whitespace value",
			envVal: "   ",
			defVal: true,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal == "" {
				os.Unsetenv("TEST_BOOL")
			} else {
				os.Setenv("TEST_BOOL", tt.envVal)
			}

			got := getbool("TEST_BOOL", tt.defVal)
			if got != tt.want {
				t.Errorf("getbool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabaseURLFallback(t *testing.T) {
	// Save original environment
	originalDBDSN := os.Getenv("GK_DB_DSN")
	originalDatabaseURL := os.Getenv("DATABASE_URL")
	defer func() {
		if originalDBDSN == "" {
			os.Unsetenv("GK_DB_DSN")
		} else {
			os.Setenv("GK_DB_DSN", originalDBDSN)
		}
		if originalDatabaseURL == "" {
			os.Unsetenv("DATABASE_URL")
		} else {
			os.Setenv("DATABASE_URL", originalDatabaseURL)
		}
	}()

	// Clear both environment variables
	os.Unsetenv("GK_DB_DSN")
	os.Unsetenv("DATABASE_URL")

	config := FromEnv()

	// Should use default value
	expectedDefault := "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable"
	if config.DBDSN != expectedDefault {
		t.Errorf("FromEnv() DBDSN = %v, want %v", config.DBDSN, expectedDefault)
	}

	// Set DATABASE_URL only
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/mydb")

	config = FromEnv()

	if config.DBDSN != "postgres://user:pass@localhost:5432/mydb" {
		t.Errorf("FromEnv() DBDSN = %v, want postgres://user:pass@localhost:5432/mydb", config.DBDSN)
	}

	// Set both, GK_DB_DSN should take precedence
	os.Setenv("GK_DB_DSN", "postgres://admin:admin@localhost:5432/admin")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/mydb")

	config = FromEnv()

	if config.DBDSN != "postgres://admin:admin@localhost:5432/admin" {
		t.Errorf("FromEnv() DBDSN = %v, want postgres://admin:admin@localhost:5432/admin", config.DBDSN)
	}
}

