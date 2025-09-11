package handlers

// Auth gRPC service: register/login/refresh.

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/crypto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type usersStore interface {
	Create(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error)
	ByEmail(ctx context.Context, email string) (*storage.User, error)
}

type AuthServer struct {
	pb.UnimplementedAuthServiceServer

	users  usersStore
	issuer *auth.Issuer
}

func NewAuthServer(users usersStore, issuer *auth.Issuer) *AuthServer {
	return &AuthServer{users: users, issuer: issuer}
}

// Register creates a new user and returns its id.
func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	email := normalizeEmail(req.GetEmail())
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid email")
	}
	if len(req.GetPassword()) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	salt, hash, err := crypto.HashPassword(crypto.DefaultArgon, []byte(req.GetPassword()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "weak password")
	}

	id, err := s.users.Create(ctx, email, hash, salt)
	if err != nil {
		// Unique violation -> user already exists.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "cannot create user")
	}

	return &pb.RegisterResponse{UserId: id.String()}, nil
}

// Login validates credentials and returns an access/refresh pair.
func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	email := normalizeEmail(req.GetEmail())
	if email == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	ok := crypto.VerifyPassword(crypto.DefaultArgon, []byte(req.GetPassword()), u.PassSalt, u.PassHash)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	tp, err := s.issuer.Issue(u.ID.String(), u.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "token issue failed")
	}

	return &pb.LoginResponse{
		AccessToken:  tp.AccessToken,
		RefreshToken: tp.RefreshToken,
	}, nil
}

// Refresh validates a refresh token and issues a fresh pair.
func (s *AuthServer) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.LoginResponse, error) {
	raw := req.GetRefreshToken()
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "missing refresh token")
	}

	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(t *jwt.Token) (any, error) {
			// Expect RS256
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected alg: %v", t.Header["alg"])
			}
			return s.issuer.Public(), nil
		},
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil || !tok.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	sub, _ := claims["sub"].(string)
	scope, _ := claims["scope"].(string)
	email, _ := claims["email"].(string)
	if sub == "" || scope != "refresh" {
		return nil, status.Error(codes.PermissionDenied, "insufficient token scope")
	}

	tp, err := s.issuer.Issue(sub, email)
	if err != nil {
		return nil, status.Error(codes.Internal, "token issue failed")
	}
	return &pb.LoginResponse{
		AccessToken:  tp.AccessToken,
		RefreshToken: tp.RefreshToken,
	}, nil
}

func normalizeEmail(s string) string {
	// Simple trim-lower; rigorous validation is done on the client.
	if s == "" {
		return ""
	}
	// Avoid importing strings for one-liners here; keep it tiny.
	b := []byte(s)
	// trim spaces
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	// lower ASCII
	for i := range b {
		if 'A' <= b[i] && b[i] <= 'Z' {
			b[i] = b[i] + ('a' - 'A')
		}
	}
	return string(b)
}
