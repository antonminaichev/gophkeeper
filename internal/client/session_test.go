package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSession(t *testing.T) {
	tests := []struct {
		name        string
		session     *Session
		expectValid bool
	}{
		{
			name: "valid session",
			session: &Session{
				AccessToken:  "valid-token",
				RefreshToken: "valid-refresh-token",
				AccessExp:    time.Now().Add(time.Hour),
			},
			expectValid: true,
		},
		{
			name: "expired session",
			session: &Session{
				AccessToken:  "expired-token",
				RefreshToken: "valid-refresh-token",
				AccessExp:    time.Now().Add(-time.Hour),
			},
			expectValid: false,
		},
		{
			name: "session with empty access token",
			session: &Session{
				AccessToken:  "",
				RefreshToken: "valid-refresh-token",
				AccessExp:    time.Now().Add(time.Hour),
			},
			expectValid: false,
		},
		{
			name: "session with empty refresh token",
			session: &Session{
				AccessToken:  "valid-token",
				RefreshToken: "",
				AccessExp:    time.Now().Add(time.Hour),
			},
			expectValid: true, // Access token is still valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.session.AccessValid()
			if valid != tt.expectValid {
				t.Errorf("AccessValid() = %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

func TestSessionWithAuth(t *testing.T) {
	session := &Session{
		AccessToken: "test-token",
	}

	ctx := context.Background()
	ctxWithAuth := session.WithAuth(ctx)

	// Test that the context contains the authorization header
	// This is a basic test - in a real scenario you'd check the metadata
	if ctxWithAuth == nil {
		t.Errorf("WithAuth() returned nil context")
	}
}

func TestLoadSession(t *testing.T) {
	// Test loading session when no session file exists
	_, err := LoadSession()
	if err == nil {
		t.Errorf("LoadSession() expected error when no session file exists")
	}
}

func TestSaveSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalSessionFile := os.Getenv("GK_SESSION_FILE")
	defer func() {
		if originalSessionFile != "" {
			os.Setenv("GK_SESSION_FILE", originalSessionFile)
		} else {
			os.Unsetenv("GK_SESSION_FILE")
		}
	}()

	// Set session file to temp directory
	sessionFile := filepath.Join(tempDir, "session.json")
	os.Setenv("GK_SESSION_FILE", sessionFile)

	session := &Session{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh-token",
		AccessExp:    time.Now().Add(time.Hour),
	}

	// Test saving session
	err := SaveSession(session)
	if err != nil {
		t.Errorf("SaveSession() error = %v", err)
	}

	// Test loading the saved session
	loadedSession, err := LoadSession()
	if err != nil {
		t.Errorf("LoadSession() error = %v", err)
	}

	if loadedSession.AccessToken != session.AccessToken {
		t.Errorf("LoadSession() AccessToken = %v, want %v", loadedSession.AccessToken, session.AccessToken)
	}

	if loadedSession.RefreshToken != session.RefreshToken {
		t.Errorf("LoadSession() RefreshToken = %v, want %v", loadedSession.RefreshToken, session.RefreshToken)
	}
}

func TestClearSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalSessionFile := os.Getenv("GK_SESSION_FILE")
	defer func() {
		if originalSessionFile != "" {
			os.Setenv("GK_SESSION_FILE", originalSessionFile)
		} else {
			os.Unsetenv("GK_SESSION_FILE")
		}
	}()

	// Set session file to temp directory
	sessionFile := filepath.Join(tempDir, "session.json")
	os.Setenv("GK_SESSION_FILE", sessionFile)

	// Save a session first
	session := &Session{
		AccessToken: "test-token",
	}
	err := SaveSession(session)
	if err != nil {
		t.Errorf("SaveSession() error = %v", err)
	}

	// Clear the session
	err = ClearSession()
	if err != nil {
		t.Errorf("ClearSession() error = %v", err)
	}

	// Try to load the session - should fail
	_, err = LoadSession()
	if err == nil {
		t.Errorf("LoadSession() expected error after ClearSession()")
	}
}

func TestSaveLoginTokens(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalSessionFile := os.Getenv("GK_SESSION_FILE")
	defer func() {
		if originalSessionFile != "" {
			os.Setenv("GK_SESSION_FILE", originalSessionFile)
		} else {
			os.Unsetenv("GK_SESSION_FILE")
		}
	}()

	// Set session file to temp directory
	sessionFile := filepath.Join(tempDir, "session.json")
	os.Setenv("GK_SESSION_FILE", sessionFile)

	// Create valid JWT tokens for testing
	accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiZXhwIjoxNzA0MDk2MDAwfQ.test"
	refreshToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiZXhwIjoxNzA0MDk2MDAwfQ.test"

	// Test saving login tokens
	err := SaveLoginTokens(accessToken, refreshToken)
	if err != nil {
		t.Errorf("SaveLoginTokens() error = %v", err)
	}

	// Load the session and verify tokens
	session, err := LoadSession()
	if err != nil {
		t.Errorf("LoadSession() error = %v", err)
	}

	if session.AccessToken != accessToken {
		t.Errorf("LoadSession() AccessToken = %v, want %v", session.AccessToken, accessToken)
	}

	if session.RefreshToken != refreshToken {
		t.Errorf("LoadSession() RefreshToken = %v, want %v", session.RefreshToken, refreshToken)
	}
}

func TestSessionEdgeCases(t *testing.T) {
	t.Run("session with zero time", func(t *testing.T) {
		session := &Session{
			AccessToken:  "test-token",
			RefreshToken: "test-refresh-token",
			AccessExp:    time.Time{}, // zero time
		}

		valid := session.AccessValid()
		if valid {
			t.Errorf("AccessValid() with zero time should return false")
		}
	})

	t.Run("session with very old expiration", func(t *testing.T) {
		session := &Session{
			AccessToken:  "test-token",
			RefreshToken: "test-refresh-token",
			AccessExp:    time.Now().Add(-24 * time.Hour), // expired 24 hours ago
		}

		valid := session.AccessValid()
		if valid {
			t.Errorf("AccessValid() with old expiration should return false")
		}
	})

	t.Run("session with future expiration", func(t *testing.T) {
		session := &Session{
			AccessToken:  "test-token",
			RefreshToken: "test-refresh-token",
			AccessExp:    time.Now().Add(24 * time.Hour), // expires in 24 hours
		}

		valid := session.AccessValid()
		if !valid {
			t.Errorf("AccessValid() with future expiration should return true")
		}
	})
}
