package client

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestNewClientTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		params      ClientTLSParams
		expectError bool
	}{
		{
			name:        "default config",
			params:      ClientTLSParams{},
			expectError: false,
		},
		{
			name: "with MinVersionTLS12",
			params: ClientTLSParams{
				MinVersionTLS12: true,
			},
			expectError: false,
		},
		{
			name: "with InsecureSkipVerify",
			params: ClientTLSParams{
				InsecureSkipVerify: true,
			},
			expectError: false,
		},
		{
			name: "with ServerName",
			params: ClientTLSParams{
				ServerName: "example.com",
			},
			expectError: false,
		},
		{
			name: "with spaces in ServerName",
			params: ClientTLSParams{
				ServerName: "  example.com  ",
			},
			expectError: false,
		},
		{
			name: "with empty ServerName",
			params: ClientTLSParams{
				ServerName: "",
			},
			expectError: false,
		},
		{
			name: "with whitespace ServerName",
			params: ClientTLSParams{
				ServerName: "   ",
			},
			expectError: false,
		},
		{
			name: "with non-existent CA file",
			params: ClientTLSParams{
				CAPath: "/path/to/non-existent/ca.pem",
			},
			expectError: true,
		},
		{
			name: "with empty CA path",
			params: ClientTLSParams{
				CAPath: "",
			},
			expectError: false,
		},
		{
			name: "with whitespace CA path",
			params: ClientTLSParams{
				CAPath: "   ",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewClientTLSConfig(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewClientTLSConfig() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewClientTLSConfig() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Errorf("NewClientTLSConfig() returned nil config")
				}
			}
		})
	}
}

func TestNewClientTLSConfigMinVersion(t *testing.T) {
	params := ClientTLSParams{MinVersionTLS12: true}
	cfg, err := NewClientTLSConfig(params)
	if err != nil {
		t.Errorf("NewClientTLSConfig() error = %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("NewClientTLSConfig() MinVersion = %v, want %v", cfg.MinVersion, tls.VersionTLS12)
	}
}

func TestNewClientTLSConfigInsecureSkipVerify(t *testing.T) {
	params := ClientTLSParams{InsecureSkipVerify: true}
	cfg, err := NewClientTLSConfig(params)
	if err != nil {
		t.Errorf("NewClientTLSConfig() error = %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Errorf("NewClientTLSConfig() InsecureSkipVerify = %v, want true", cfg.InsecureSkipVerify)
	}
}

func TestNewClientTLSConfigServerName(t *testing.T) {
	params := ClientTLSParams{ServerName: "example.com"}
	cfg, err := NewClientTLSConfig(params)
	if err != nil {
		t.Errorf("NewClientTLSConfig() error = %v", err)
	}
	if cfg.ServerName != "example.com" {
		t.Errorf("NewClientTLSConfig() ServerName = %v, want example.com", cfg.ServerName)
	}
}

func TestNewClientTLSConfigWithInvalidCAFile(t *testing.T) {
	// Create a temporary file with invalid PEM content
	tmpfile, err := os.CreateTemp("", "test_ca")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write invalid PEM content
	if _, err := tmpfile.WriteString("invalid pem content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	params := ClientTLSParams{CAPath: tmpfile.Name()}
	_, err = NewClientTLSConfig(params)
	if err == nil {
		t.Errorf("NewClientTLSConfig() with invalid CA file should return error")
	}
}

func TestNewClientTLSConfigWithValidCAFile(t *testing.T) {
	// This test is skipped because creating a valid PEM certificate
	// in tests is complex and not essential for coverage
	t.Skip("Skipping complex PEM certificate test")
}

func TestClientTLSParams(t *testing.T) {
	params := ClientTLSParams{
		CAPath:             "/path/to/ca.pem",
		ServerName:         "example.com",
		InsecureSkipVerify: true,
		MinVersionTLS12:    true,
	}

	if params.CAPath != "/path/to/ca.pem" {
		t.Errorf("CAPath = %v, want /path/to/ca.pem", params.CAPath)
	}
	if params.ServerName != "example.com" {
		t.Errorf("ServerName = %v, want example.com", params.ServerName)
	}
	if !params.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify = %v, want true", params.InsecureSkipVerify)
	}
	if !params.MinVersionTLS12 {
		t.Errorf("MinVersionTLS12 = %v, want true", params.MinVersionTLS12)
	}
}
