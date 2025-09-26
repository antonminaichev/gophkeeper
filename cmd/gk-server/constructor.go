package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/antonminaichev/gophkeeper/internal/config"
)

// NewServerTLSConfig builds a TLS config from env-backed config.
// Supports optional client CA for mTLS.
func NewServerTLSConfig(cfg config.Config) (*tls.Config, error) {
	if strings.TrimSpace(cfg.TLSCertPath) == "" || strings.TrimSpace(cfg.TLSKeyPath) == "" {
		return nil, errors.New("GK_TLS_CERT_PATH and GK_TLS_KEY_PATH are required when GK_TLS_ENABLE=true")
	}

	cert, err := tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if cfg.TLSMinVersion12 {
		tlsCfg.MinVersion = tls.VersionTLS12
	}

	// Optional client CA for mTLS
	if caPath := strings.TrimSpace(cfg.TLSClientCAPath); caPath != "" {
		caData, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("read client CA: %w", err)
		}
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM(caData) {
			return nil, errors.New("parse client CA: no certs found")
		}
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		tlsCfg.ClientCAs = cp
	}
	return tlsCfg, nil
}
