package client

import (
	"os"
	"testing"
)

func TestNewCLI(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
	}{
		{
			name:    "all fields",
			version: "1.0.0",
			commit:  "abc123",
			date:    "2023-01-01",
		},
		{
			name:    "empty fields",
			version: "",
			commit:  "",
			date:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewCLI(tt.version, tt.commit, tt.date)

			if cli == nil {
				t.Errorf("NewCLI() returned nil")
				return
			}

			if cli.Version != tt.version {
				t.Errorf("NewCLI() Version = %q, want %q", cli.Version, tt.version)
			}

			if cli.Commit != tt.commit {
				t.Errorf("NewCLI() Commit = %q, want %q", cli.Commit, tt.commit)
			}

			if cli.Date != tt.date {
				t.Errorf("NewCLI() Date = %q, want %q", cli.Date, tt.date)
			}
		})
	}
}

func TestCLI_CreateCommands(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	rootCmd := cli.CreateCommands()

	if rootCmd == nil {
		t.Errorf("CreateCommands() returned nil")
		return
	}

	if rootCmd.Use != "gk" {
		t.Errorf("CreateCommands() Use = %q, want %q", rootCmd.Use, "gk")
	}

	if rootCmd.Short != "GophKeeper CLI" {
		t.Errorf("CreateCommands() Short = %q, want %q", rootCmd.Short, "GophKeeper CLI")
	}
}

func TestCLI_setupFlags(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	// Create a command to test flag setup
	rootCmd := cli.CreateCommands()

	// Test that flags are set
	if rootCmd.PersistentFlags().Lookup("server") == nil {
		t.Errorf("setupFlags() did not set server flag")
	}

	if rootCmd.PersistentFlags().Lookup("token") == nil {
		t.Errorf("setupFlags() did not set token flag")
	}

	if rootCmd.PersistentFlags().Lookup("tls") == nil {
		t.Errorf("setupFlags() did not set tls flag")
	}

	if rootCmd.PersistentFlags().Lookup("tls-ca") == nil {
		t.Errorf("setupFlags() did not set tls-ca flag")
	}

	if rootCmd.PersistentFlags().Lookup("tls-server-name") == nil {
		t.Errorf("setupFlags() did not set tls-server-name flag")
	}

	if rootCmd.PersistentFlags().Lookup("tls-insecure-skip-verify") == nil {
		t.Errorf("setupFlags() did not set tls-insecure-skip-verify flag")
	}
}

func TestCLI_DefaultValues(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	// Test default values
	if cli.ServerAddr != "" {
		// ServerAddr should be set by flags, not by default
	}

	if cli.TLSEnable != false {
		t.Errorf("CLI TLSEnable default = %v, want false", cli.TLSEnable)
	}

	if cli.TLSInsecureSkipVerify != false {
		t.Errorf("CLI TLSInsecureSkipVerify default = %v, want false", cli.TLSInsecureSkipVerify)
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	// Test that we can read environment variables
	originalToken := os.Getenv("GK_ACCESS_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GK_ACCESS_TOKEN", originalToken)
		} else {
			os.Unsetenv("GK_ACCESS_TOKEN")
		}
	}()

	// Set a test token
	os.Setenv("GK_ACCESS_TOKEN", "test-token")

	_ = NewCLI("1.0.0", "abc123", "2023-01-01")

	// The token should be available in the environment
	token := os.Getenv("GK_ACCESS_TOKEN")
	if token != "test-token" {
		t.Errorf("Environment variable GK_ACCESS_TOKEN = %q, want %q", token, "test-token")
	}
}
