// file: client/auth_helpers_test.go
package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ---- fake AuthService ----

type fakeAuthServer struct {
	pb.UnimplementedAuthServiceServer
	access  string
	refresh string
	err     error
}

func (f *fakeAuthServer) Refresh(ctx context.Context, r *pb.RefreshRequest) (*pb.LoginResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &pb.LoginResponse{AccessToken: f.access, RefreshToken: f.refresh}, nil
}

func startFakeAuth(t *testing.T, srv pb.AuthServiceServer) (addr string, stop func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	g := grpc.NewServer()
	pb.RegisterAuthServiceServer(g, srv)
	go g.Serve(l)
	return l.Addr().String(), func() { g.Stop(); _ = l.Close() }
}

// ---- tests ----

func Test_attachAuth_WithAccessToken_setsMetadata(t *testing.T) {
	cli := &CLI{AccessToken: "abc123"}
	ctx := context.Background()

	ctx2, sess, err := cli.attachAuth(ctx)
	if err != nil {
		t.Fatalf("attachAuth err: %v", err)
	}
	if sess != nil {
		t.Fatalf("expected nil session when AccessToken flag is used")
	}
	md, ok := metadata.FromOutgoingContext(ctx2)
	if !ok {
		t.Fatalf("no outgoing metadata")
	}
	got := md.Get("authorization")
	if len(got) != 1 || got[0] != "Bearer abc123" {
		t.Fatalf("authorization metadata = %#v", got)
	}
}

func Test_attachAuth_WithoutToken_callsEnsureAuth_andFails(t *testing.T) {
	cli := &CLI{AccessToken: "   "}
	_, _, err := cli.attachAuth(context.Background())
	if err == nil {
		t.Fatalf("expected error from EnsureAuth when no session")
	}
}

func Test_refreshSession_Success(t *testing.T) {
	// Сгенерируем корректный RS256 JWT с exp в будущем.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	exp := time.Now().Add(2 * time.Minute).Unix()
	accessJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "u1", "scope": "access", "exp": exp,
	})
	access, err := accessJWT.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	addr, stop := startFakeAuth(t, &fakeAuthServer{
		access:  access,
		refresh: "refresh-token",
	})
	defer stop()

	cli := &CLI{
		ServerAddr: addr,
		TLSEnable:  false, // клиент должен пойти по insecure
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s, err := cli.refreshSession(ctx, "any-refresh")
	if err != nil {
		t.Fatalf("refreshSession err: %v", err)
	}
	if s.AccessToken == "" || s.RefreshToken == "" {
		t.Fatalf("session tokens must be set: %+v", s)
	}
	// Проверим, что exp распарсился как время в будущем.
	if time.Until(s.AccessExp) <= 0 {
		t.Fatalf("AccessExp not in future: %v", s.AccessExp)
	}
}

func Test_refreshSession_ServerError(t *testing.T) {
	addr, stop := startFakeAuth(t, &fakeAuthServer{
		err: grpc.Errorf(13, "boom"), // codes.Internal
	})
	defer stop()

	cli := &CLI{
		ServerAddr: addr,
		TLSEnable:  false,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := cli.refreshSession(ctx, "refresh")
	if err == nil {
		t.Fatalf("expected error from Refresh")
	}
}
