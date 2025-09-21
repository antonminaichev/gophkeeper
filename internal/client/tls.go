package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

type ClientTLSParams struct {
	CAPath             string
	ServerName         string
	InsecureSkipVerify bool
	MinVersionTLS12    bool
}

// NewClientTLSConfig constructor
func NewClientTLSConfig(p ClientTLSParams) (*tls.Config, error) {
	cfg := &tls.Config{}
	if p.MinVersionTLS12 {
		cfg.MinVersion = tls.VersionTLS12
	}
	cfg.InsecureSkipVerify = p.InsecureSkipVerify // ONLY for local

	if strings.TrimSpace(p.ServerName) != "" {
		cfg.ServerName = p.ServerName
	}
	if strings.TrimSpace(p.CAPath) != "" {
		data, err := os.ReadFile(p.CAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("parse CA file: no certs found")
		}
		cfg.RootCAs = cp
	}
	return cfg, nil
}
