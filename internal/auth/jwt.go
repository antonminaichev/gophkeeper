// Auth package is used for user auth.
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Pair of token for auth flow.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// TTLs controls.
type TTLs struct {
	Access  time.Duration
	Refresh time.Duration
}

// Issuer signs RS256 tokens with the given private key.
type Issuer struct {
	priv *rsa.PrivateKey
	ttl  TTLs
}

// NewIssuer constructs an Issuer with access/refresh TTLs.
func NewIssuer(priv *rsa.PrivateKey, access, refresh time.Duration) *Issuer {
	if access <= 0 {
		access = 15 * time.Minute
	}
	if refresh <= 0 {
		refresh = 30 * 24 * time.Hour
	}
	return &Issuer{
		priv: priv,
		ttl:  TTLs{Access: access, Refresh: refresh},
	}
}

// Issue creates and signs a fresh access/refresh pair for the user.
func (i *Issuer) Issue(userID, email string) (TokenPair, error) {
	now := time.Now()

	// Access token — short-lived, used for API calls.
	accessClaims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"scope": "access",
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(i.ttl.Access).Unix(),
	}
	access := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)

	// Refresh token — longer-lived, for re-issuing access tokens.
	jti := randHex(16)
	refreshClaims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"scope": "refresh",
		"jti":   jti,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(i.ttl.Refresh).Unix(),
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)

	at, err := access.SignedString(i.priv)
	if err != nil {
		return TokenPair{}, err
	}
	rt, err := refresh.SignedString(i.priv)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: at, RefreshToken: rt}, nil
}

// Public returns the matching public key for verification.
func (i *Issuer) Public() *rsa.PublicKey {
	return &i.priv.PublicKey
}

// randHex returns a secure random hex string of 2*n length.
func randHex(n int) string {
	if n <= 0 {
		n = 16
	}
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func Expiration(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, errors.New("empty token")
	}
	var claims jwt.MapClaims
	_, _, err := new(jwt.Parser).ParseUnverified(raw, &claims)
	if err != nil {
		return time.Time{}, err
	}
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return time.Time{}, err
	}
	return exp.Time, nil
}
