package client

// Session management for GophKeeper CLI.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/metadata"
)

// sessionFile can be overridden via GK_SESSION_FILE.
func sessionFile() (string, error) {
	if p := os.Getenv("GK_SESSION_FILE"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "gophkeeper", "session.json")
	return path, nil
}

// Session represents persisted auth state (+ sync cursor).
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	AccessExp    time.Time `json:"access_exp"` // parsed from JWT "exp"
	SavedAt      time.Time `json:"saved_at"`
	SyncCursor   string    `json:"sync_cursor,omitempty"`
}

// LoadSession reads session from disk.
func LoadSession() (*Session, error) {
	path, err := sessionFile()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSession writes session to disk.
func SaveSession(s *Session) error {
	if s == nil {
		return errors.New("nil session")
	}
	path, err := sessionFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.SavedAt = time.Now()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// Best effort restrictive perms; on Windows this is ignored.
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return nil
}

// ClearSession removes the session file (logout).
func ClearSession() error {
	path, err := sessionFile()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AccessValid reports whether access token is still valid.
func (s *Session) AccessValid() bool {
	if s == nil || s.AccessToken == "" {
		return false
	}
	return time.Now().Add(30 * time.Second).Before(s.AccessExp)
}

// WithAuth attaches Authorization header to context when possible.
func (s *Session) WithAuth(ctx context.Context) context.Context {
	if s == nil || s.AccessToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+s.AccessToken)
}

// SaveLoginTokens parses exp from access token and persists session.
func SaveLoginTokens(access, refresh string) error {
	exp, err := parseExp(access)
	if err != nil {
		return fmt.Errorf("parse access token: %w", err)
	}
	return SaveSession(&Session{
		AccessToken:  access,
		RefreshToken: refresh,
		AccessExp:    exp,
	})
}
