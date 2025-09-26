package client

import (
	"context"
	"testing"
	"time"
)

func TestCLI_clientDial(t *testing.T) {
	cli := &CLI{
		ServerAddr: "localhost:8090",
		TLSEnable:  false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test without TLS
	conn, err := cli.clientDial(ctx)
	if err == nil {
		conn.Close()
	}
	// We expect an error since the server is not running, but the function should not panic
}

func TestCLI_clientDialWithTLS(t *testing.T) {
	cli := &CLI{
		ServerAddr: "localhost:8090",
		TLSEnable:  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test with TLS
	conn, err := cli.clientDial(ctx)
	if err == nil {
		conn.Close()
	}
	// We expect an error since the server is not running, but the function should not panic
}

func TestCLI_dialVaultClient(t *testing.T) {
	cli := &CLI{
		ServerAddr: "localhost:8090",
		TLSEnable:  false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test dialVaultClient
	client, conn, err := cli.dialVaultClient(ctx)
	if err == nil {
		conn.Close()
	}
	// We expect an error since the server is not running, but the function should not panic
	_ = client // Avoid unused variable warning
}

func TestCLI_dialAuthClient(t *testing.T) {
	cli := &CLI{
		ServerAddr: "localhost:8090",
		TLSEnable:  false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test dialAuthClient
	client, conn, err := cli.dialAuthClient(ctx)
	if err == nil {
		conn.Close()
	}
	// We expect an error since the server is not running, but the function should not panic
	_ = client // Avoid unused variable warning
}
