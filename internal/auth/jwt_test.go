package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestKeyPair creates a test RSA key pair
func generateTestKeyPair(t *testing.T) *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	return key
}

func TestNewIssuer(t *testing.T) {
	key := generateTestKeyPair(t)

	tests := []struct {
		name        string
		priv        *rsa.PrivateKey
		access      time.Duration
		refresh     time.Duration
		wantAccess  time.Duration
		wantRefresh time.Duration
	}{
		{
			name:        "custom durations",
			priv:        key,
			access:      30 * time.Minute,
			refresh:     24 * time.Hour,
			wantAccess:  30 * time.Minute,
			wantRefresh: 24 * time.Hour,
		},
		{
			name:        "zero access duration",
			priv:        key,
			access:      0,
			refresh:     24 * time.Hour,
			wantAccess:  15 * time.Minute, // default
			wantRefresh: 24 * time.Hour,
		},
		{
			name:        "zero refresh duration",
			priv:        key,
			access:      30 * time.Minute,
			refresh:     0,
			wantAccess:  30 * time.Minute,
			wantRefresh: 30 * 24 * time.Hour, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer := NewIssuer(tt.priv, tt.access, tt.refresh)

			if issuer == nil {
				t.Errorf("NewIssuer() returned nil")
				return
			}

			if issuer.ttl.Access != tt.wantAccess {
				t.Errorf("NewIssuer() access TTL = %v, want %v", issuer.ttl.Access, tt.wantAccess)
			}

			if issuer.ttl.Refresh != tt.wantRefresh {
				t.Errorf("NewIssuer() refresh TTL = %v, want %v", issuer.ttl.Refresh, tt.wantRefresh)
			}

			if issuer.priv != tt.priv {
				t.Errorf("NewIssuer() private key not set correctly")
			}
		})
	}
}

func TestIssue(t *testing.T) {
	key := generateTestKeyPair(t)
	issuer := NewIssuer(key, 15*time.Minute, 24*time.Hour)

	tests := []struct {
		name    string
		userID  string
		email   string
		wantErr bool
	}{
		{
			name:    "valid user",
			userID:  "user123",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			email:   "test@example.com",
			wantErr: false, // JWT allows empty sub
		},
		{
			name:    "empty email",
			userID:  "user123",
			email:   "",
			wantErr: false, // JWT allows empty email
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair, err := issuer.Issue(tt.userID, tt.email)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Issue() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Issue() unexpected error = %v", err)
				return
			}

			if pair.AccessToken == "" {
				t.Errorf("Issue() access token is empty")
			}

			if pair.RefreshToken == "" {
				t.Errorf("Issue() refresh token is empty")
			}

			// Verify tokens can be parsed
			if err := verifyToken(pair.AccessToken, issuer.Public()); err != nil {
				t.Errorf("Issue() access token verification failed: %v", err)
			}

			if err := verifyToken(pair.RefreshToken, issuer.Public()); err != nil {
				t.Errorf("Issue() refresh token verification failed: %v", err)
			}
		})
	}
}

func TestPublic(t *testing.T) {
	key := generateTestKeyPair(t)
	issuer := NewIssuer(key, 15*time.Minute, 24*time.Hour)

	publicKey := issuer.Public()
	if publicKey == nil {
		t.Errorf("Public() returned nil")
		return
	}

	// Check that public key matches private key
	if publicKey.N.Cmp(key.N) != 0 {
		t.Errorf("Public() key modulus doesn't match private key")
	}

	if publicKey.E != key.E {
		t.Errorf("Public() key exponent doesn't match private key")
	}
}

func TestTokenClaims(t *testing.T) {
	key := generateTestKeyPair(t)
	issuer := NewIssuer(key, 15*time.Minute, 24*time.Hour)

	userID := "user123"
	email := "test@example.com"

	pair, err := issuer.Issue(userID, email)
	if err != nil {
		t.Fatalf("Issue() failed: %v", err)
	}

	// Parse access token
	accessToken, err := jwt.Parse(pair.AccessToken, func(token *jwt.Token) (interface{}, error) {
		return issuer.Public(), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse access token: %v", err)
	}

	accessClaims, ok := accessToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("Failed to parse access token claims")
	}

	// Check access token claims
	if sub, ok := accessClaims["sub"].(string); !ok || sub != userID {
		t.Errorf("Access token sub = %v, want %s", accessClaims["sub"], userID)
	}

	if emailClaim, ok := accessClaims["email"].(string); !ok || emailClaim != email {
		t.Errorf("Access token email = %v, want %s", accessClaims["email"], email)
	}

	if scope, ok := accessClaims["scope"].(string); !ok || scope != "access" {
		t.Errorf("Access token scope = %v, want access", accessClaims["scope"])
	}

	// Parse refresh token
	refreshToken, err := jwt.Parse(pair.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return issuer.Public(), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse refresh token: %v", err)
	}

	refreshClaims, ok := refreshToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("Failed to parse refresh token claims")
	}

	// Check refresh token claims
	if sub, ok := refreshClaims["sub"].(string); !ok || sub != userID {
		t.Errorf("Refresh token sub = %v, want %s", refreshClaims["sub"], userID)
	}

	if emailClaim, ok := refreshClaims["email"].(string); !ok || emailClaim != email {
		t.Errorf("Refresh token email = %v, want %s", refreshClaims["email"], email)
	}

	if scope, ok := refreshClaims["scope"].(string); !ok || scope != "refresh" {
		t.Errorf("Refresh token scope = %v, want refresh", refreshClaims["scope"])
	}

	if jti, ok := refreshClaims["jti"].(string); !ok || jti == "" {
		t.Errorf("Refresh token jti = %v, want non-empty string", refreshClaims["jti"])
	}
}

func TestTokenExpiration(t *testing.T) {
	key := generateTestKeyPair(t)
	accessTTL := 1 * time.Second
	refreshTTL := 2 * time.Second
	issuer := NewIssuer(key, accessTTL, refreshTTL)

	pair, err := issuer.Issue("user123", "test@example.com")
	if err != nil {
		t.Fatalf("Issue() failed: %v", err)
	}

	// Parse tokens and check expiration
	accessToken, err := jwt.Parse(pair.AccessToken, func(token *jwt.Token) (interface{}, error) {
		return issuer.Public(), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse access token: %v", err)
	}

	refreshToken, err := jwt.Parse(pair.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return issuer.Public(), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse refresh token: %v", err)
	}

	// Check that tokens are currently valid
	if !accessToken.Valid {
		t.Errorf("Access token should be valid")
	}

	if !refreshToken.Valid {
		t.Errorf("Refresh token should be valid")
	}

	// Wait for access token to expire
	time.Sleep(accessTTL + 100*time.Millisecond)

	// Parse access token again to check expiration
	accessToken2, err := jwt.Parse(pair.AccessToken, func(token *jwt.Token) (interface{}, error) {
		return issuer.Public(), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse access token after expiration: %v", err)
	}

	if accessToken2.Valid {
		t.Errorf("Access token should be expired")
	}
}

// verifyToken verifies a JWT token with the given public key
func verifyToken(tokenString string, publicKey *rsa.PublicKey) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return publicKey, nil
	})

	if err != nil {
		return err
	}

	if !token.Valid {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

// Benchmark tests
func BenchmarkIssue(b *testing.B) {
	key := generateTestKeyPair(&testing.T{})
	issuer := NewIssuer(key, 15*time.Minute, 24*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := issuer.Issue("user123", "test@example.com")
		if err != nil {
			b.Fatalf("Issue() failed: %v", err)
		}
	}
}

