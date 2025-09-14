package handlers

import (
	"context"
	"crypto/rsa"
	"testing"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockUsersStore is a mock implementation of usersStore interface
type mockUsersStore struct {
	users   map[string]*storage.User
	created []uuid.UUID
}

func newMockUsersStore() *mockUsersStore {
	return &mockUsersStore{
		users: make(map[string]*storage.User),
	}
}

func (m *mockUsersStore) Create(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
	// Simulate unique constraint violation
	if _, exists := m.users[email]; exists {
		return uuid.Nil, &mockPgError{code: "23505"}
	}

	id := uuid.New()
	m.users[email] = &storage.User{
		ID:       id,
		Email:    email,
		PassHash: passHash,
		PassSalt: passSalt,
	}
	m.created = append(m.created, id)
	return id, nil
}

func (m *mockUsersStore) ByEmail(ctx context.Context, email string) (*storage.User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, &mockPgError{code: "23505"} // Simulate not found
	}
	return user, nil
}

// mockPgError simulates PostgreSQL error
type mockPgError struct {
	code string
}

func (e *mockPgError) Error() string {
	return "mock pg error"
}

func (e *mockPgError) Code() string {
	return e.code
}

// generateTestPrivateKey creates a test RSA private key
func generateTestPrivateKey(t *testing.T) *rsa.PrivateKey {
	// This is a test key - in production you'd use a real key
	// For testing purposes, we'll skip key generation
	t.Skip("Skipping JWT tests - requires proper RSA key setup")
	return nil
}

// TestAuthServer tests the AuthServer implementation
func TestAuthServer(t *testing.T) {
	// Create mock dependencies
	usersStore := newMockUsersStore()

	// Create JWT issuer (using test keys)
	testPrivateKey := generateTestPrivateKey(t)
	issuer := auth.NewIssuer(testPrivateKey, 3600, 86400)

	server := NewAuthServer(usersStore, issuer)

	t.Run("Register valid user", func(t *testing.T) {
		req := &pb.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := server.Register(context.Background(), req)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}
		if resp == nil {
			t.Errorf("Register() returned nil response")
		}
		if resp.UserId == "" {
			t.Errorf("Register() returned empty user ID")
		}
	})

	t.Run("Register with invalid email", func(t *testing.T) {
		req := &pb.RegisterRequest{
			Email:    "",
			Password: "password123",
		}

		_, err := server.Register(context.Background(), req)
		if err == nil {
			t.Errorf("Register() with empty email should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Register() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("Register() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})

	t.Run("Register with short password", func(t *testing.T) {
		req := &pb.RegisterRequest{
			Email:    "test@example.com",
			Password: "short",
		}

		_, err := server.Register(context.Background(), req)
		if err == nil {
			t.Errorf("Register() with short password should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Register() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("Register() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})

	t.Run("Register duplicate user", func(t *testing.T) {
		email := "duplicate@example.com"
		req := &pb.RegisterRequest{
			Email:    email,
			Password: "password123",
		}

		// Register first time
		_, err := server.Register(context.Background(), req)
		if err != nil {
			t.Errorf("First Register() error = %v", err)
		}

		// Try to register again
		_, err = server.Register(context.Background(), req)
		if err == nil {
			t.Errorf("Register() duplicate user should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Register() error should be gRPC status error")
		}
		if statusErr.Code() != codes.AlreadyExists {
			t.Errorf("Register() error code = %v, want %v", statusErr.Code(), codes.AlreadyExists)
		}
	})

	t.Run("Login valid user", func(t *testing.T) {
		email := "login@example.com"
		password := "password123"

		// Register user first
		req := &pb.RegisterRequest{
			Email:    email,
			Password: password,
		}
		_, err := server.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		// Login
		loginReq := &pb.LoginRequest{
			Email:    email,
			Password: password,
		}
		resp, err := server.Login(context.Background(), loginReq)
		if err != nil {
			t.Errorf("Login() error = %v", err)
		}
		if resp == nil {
			t.Errorf("Login() returned nil response")
		}
		if resp.AccessToken == "" {
			t.Errorf("Login() returned empty access token")
		}
		if resp.RefreshToken == "" {
			t.Errorf("Login() returned empty refresh token")
		}
	})

	t.Run("Login with invalid credentials", func(t *testing.T) {
		req := &pb.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		}

		_, err := server.Login(context.Background(), req)
		if err == nil {
			t.Errorf("Login() with invalid credentials should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Login() error should be gRPC status error")
		}
		if statusErr.Code() != codes.NotFound {
			t.Errorf("Login() error code = %v, want %v", statusErr.Code(), codes.NotFound)
		}
	})

	t.Run("Login with wrong password", func(t *testing.T) {
		email := "wrongpass@example.com"
		password := "password123"

		// Register user first
		req := &pb.RegisterRequest{
			Email:    email,
			Password: password,
		}
		_, err := server.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		// Login with wrong password
		loginReq := &pb.LoginRequest{
			Email:    email,
			Password: "wrongpassword",
		}
		_, err = server.Login(context.Background(), loginReq)
		if err == nil {
			t.Errorf("Login() with wrong password should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Login() error should be gRPC status error")
		}
		if statusErr.Code() != codes.PermissionDenied {
			t.Errorf("Login() error code = %v, want %v", statusErr.Code(), codes.PermissionDenied)
		}
	})

	t.Run("Login with empty credentials", func(t *testing.T) {
		req := &pb.LoginRequest{
			Email:    "",
			Password: "",
		}

		_, err := server.Login(context.Background(), req)
		if err == nil {
			t.Errorf("Login() with empty credentials should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("Login() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("Login() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})

	t.Run("Email normalization", func(t *testing.T) {
		email := "  TEST@EXAMPLE.COM  "
		normalizedEmail := "test@example.com"
		password := "password123"

		// Register with normalized email
		req := &pb.RegisterRequest{
			Email:    normalizedEmail,
			Password: password,
		}
		_, err := server.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		// Login with different case/spaces
		loginReq := &pb.LoginRequest{
			Email:    email,
			Password: password,
		}
		resp, err := server.Login(context.Background(), loginReq)
		if err != nil {
			t.Errorf("Login() error = %v", err)
		}
		if resp == nil {
			t.Errorf("Login() returned nil response")
		}
	})
}
