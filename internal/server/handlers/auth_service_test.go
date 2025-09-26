// file: handlers/auth_service_test.go
package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/crypto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- test helpers ----

type memUsers struct {
	createFn  func(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error)
	byEmailFn func(ctx context.Context, email string) (*storage.User, error)
}

func (m *memUsers) Create(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
	return m.createFn(ctx, email, passHash, passSalt)
}
func (m *memUsers) ByEmail(ctx context.Context, email string) (*storage.User, error) {
	return m.byEmailFn(ctx, email)
}

func newTestIssuer(t *testing.T) *auth.Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	// см. /internal/auth/jwt.go: func NewIssuer(priv *rsa.PrivateKey, access, refresh time.Duration) *Issuer
	return auth.NewIssuer(key, time.Minute, time.Hour)
}

// ---- Register ----

func TestRegister_InvalidEmail(t *testing.T) {
	s := NewAuthServer(&memUsers{}, newTestIssuer(t))
	_, err := s.Register(context.Background(), &pb.RegisterRequest{Email: "  \t", Password: "12345678"})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", st.Code())
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	s := NewAuthServer(&memUsers{}, newTestIssuer(t))
	_, err := s.Register(context.Background(), &pb.RegisterRequest{Email: "a@b.c", Password: "123"})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", st.Code())
	}
}

func TestRegister_AlreadyExists(t *testing.T) {
	store := &memUsers{
		createFn: func(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
			return uuid.Nil, &pgconn.PgError{Code: "23505"}
		},
	}
	s := NewAuthServer(store, newTestIssuer(t))
	_, err := s.Register(context.Background(), &pb.RegisterRequest{Email: "a@b.c", Password: "12345678"})
	st, _ := status.FromError(err)
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("got %v, want AlreadyExists", st.Code())
	}
}

func TestRegister_InternalOnCreate(t *testing.T) {
	store := &memUsers{
		createFn: func(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
			return uuid.Nil, errors.New("db down")
		},
	}
	s := NewAuthServer(store, newTestIssuer(t))
	_, err := s.Register(context.Background(), &pb.RegisterRequest{Email: "a@b.c", Password: "12345678"})
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("got %v, want Internal", st.Code())
	}
}

func TestRegister_OK(t *testing.T) {
	id := uuid.New()
	var gotEmail string
	store := &memUsers{
		createFn: func(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
			if len(passHash) == 0 || len(passSalt) == 0 {
				t.Fatalf("hash/salt empty")
			}
			gotEmail = email
			return id, nil
		},
	}
	s := NewAuthServer(store, newTestIssuer(t))
	resp, err := s.Register(context.Background(), &pb.RegisterRequest{Email: "  USER@Example.Com \n", Password: "12345678"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.GetUserId() != id.String() {
		t.Fatalf("userId=%q want %q", resp.GetUserId(), id.String())
	}
	if gotEmail != "user@example.com" {
		t.Fatalf("normalized email=%q", gotEmail)
	}
}

// ---- Login ----

func TestLogin_InvalidArgs(t *testing.T) {
	s := NewAuthServer(&memUsers{}, newTestIssuer(t))
	_, err := s.Login(context.Background(), &pb.LoginRequest{Email: "", Password: ""})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", st.Code())
	}
}

func TestLogin_NotFound(t *testing.T) {
	s := NewAuthServer(&memUsers{
		byEmailFn: func(ctx context.Context, email string) (*storage.User, error) { return nil, errors.New("no rows") },
	}, newTestIssuer(t))
	_, err := s.Login(context.Background(), &pb.LoginRequest{Email: "x@y.z", Password: "12345678"})
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("got %v, want NotFound", st.Code())
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	salt, hash, _ := crypto.HashPassword(crypto.DefaultArgon, []byte("right-pass"))
	user := &storage.User{ID: uuid.New(), Email: "x@y.z", PassSalt: salt, PassHash: hash}
	s := NewAuthServer(&memUsers{
		byEmailFn: func(ctx context.Context, email string) (*storage.User, error) { return user, nil },
	}, newTestIssuer(t))
	_, err := s.Login(context.Background(), &pb.LoginRequest{Email: "x@y.z", Password: "wrong"})
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v, want PermissionDenied", st.Code())
	}
}

func TestLogin_OK(t *testing.T) {
	salt, hash, _ := crypto.HashPassword(crypto.DefaultArgon, []byte("secret-123"))
	user := &storage.User{ID: uuid.New(), Email: "x@y.z", PassSalt: salt, PassHash: hash}
	iss := newTestIssuer(t)
	s := NewAuthServer(&memUsers{
		byEmailFn: func(ctx context.Context, email string) (*storage.User, error) { return user, nil },
	}, iss)

	resp, err := s.Login(context.Background(), &pb.LoginRequest{Email: "  X@Y.Z  ", Password: "secret-123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.GetAccessToken() == "" || resp.GetRefreshToken() == "" {
		t.Fatalf("empty tokens")
	}
}

// ---- Refresh ----

func TestRefresh_MissingToken(t *testing.T) {
	s := NewAuthServer(&memUsers{}, newTestIssuer(t))
	_, err := s.Refresh(context.Background(), &pb.RefreshRequest{RefreshToken: ""})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", st.Code())
	}
}

func TestRefresh_InvalidTokenFormat(t *testing.T) {
	s := NewAuthServer(&memUsers{}, newTestIssuer(t))
	_, err := s.Refresh(context.Background(), &pb.RefreshRequest{RefreshToken: "not-a-jwt"})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("got %v, want Unauthenticated", st.Code())
	}
}

func TestRefresh_InsufficientScope_UsingAccessToken(t *testing.T) {
	iss := newTestIssuer(t)
	s := NewAuthServer(&memUsers{}, iss)
	// получаем пару и подсовываем access как "refresh" -> подпись валидна, scope != refresh
	tp, err := iss.Issue("u1", "u@e")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	_, err = s.Refresh(context.Background(), &pb.RefreshRequest{RefreshToken: tp.AccessToken})
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v, want PermissionDenied", st.Code())
	}
}

func TestRefresh_OK(t *testing.T) {
	iss := newTestIssuer(t)
	s := NewAuthServer(&memUsers{}, iss)
	tp, err := iss.Issue("user-123", "u@e")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	resp, err := s.Refresh(context.Background(), &pb.RefreshRequest{RefreshToken: tp.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if resp.GetAccessToken() == "" || resp.GetRefreshToken() == "" {
		t.Fatalf("empty tokens")
	}
}

// ---- normalizeEmail ----

func TestNormalizeEmail(t *testing.T) {
	if normalizeEmail("") != "" {
		t.Fatalf("empty -> empty expected")
	}
	if got := normalizeEmail(" \tUser@Example.COM \r\n"); got != "user@example.com" {
		t.Fatalf("normalize=%q", got)
	}
}
