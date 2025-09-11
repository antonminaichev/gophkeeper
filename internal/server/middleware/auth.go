// Middleware package describes interceptors.
package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	gkauth "github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthUnary(publicKey any) grpc.UnaryServerInterceptor {
	var keyFunc jwt.Keyfunc
	switch k := publicKey.(type) {
	case jwt.Keyfunc:
		keyFunc = k
	default:
		keyFunc = func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		fullMethod := info.FullMethod
		if isPublicMethod(fullMethod) {
			return handler(ctx, req)
		}

		raw, err := bearerFromMD(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		// Parse + validate token.
		claims := jwt.MapClaims{}
		tok, err := jwt.ParseWithClaims(
			raw,
			claims,
			keyFunc,
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithLeeway(2*time.Second),
		)
		if err != nil || !tok.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Basic claims we expect to see.
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		scope, _ := claims["scope"].(string)
		if sub == "" || scope != "access" {
			return nil, status.Error(codes.PermissionDenied, "insufficient token scope")
		}

		// Pass user identity downstream.
		ctx = gkauth.WithUser(ctx, sub, email)
		return handler(ctx, req)
	}
}

// bearerFromMD extracts a Bearer token from gRPC metadata.
func bearerFromMD(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	var authVals []string
	for k, vals := range md {
		if strings.EqualFold(k, "authorization") && len(vals) > 0 {
			authVals = append(authVals, vals...)
		}
	}
	if len(authVals) == 0 {
		return "", fmt.Errorf("missing authorization")
	}

	// Find the first "Bearer <token>" header.
	for _, v := range authVals {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(v)), "bearer ") {
			return strings.TrimSpace(v[len("Bearer "):]), nil
		}
	}
	return "", fmt.Errorf("invalid authorization scheme")
}

// isPublicMethod detects methods that do not require auth.
func isPublicMethod(full string) bool {
	// Health and reflection should always be reachable.
	if full == "/grpc.health.v1.Health/Check" ||
		strings.Contains(full, "grpc.reflection.v1alpha.ServerReflection") {
		return true
	}
	// Allow common auth endpoints by suffix to survive package changes.
	if strings.HasSuffix(full, "/Register") ||
		strings.HasSuffix(full, "/Login") ||
		strings.HasSuffix(full, "/Refresh") {
		return true
	}
	return false
}
