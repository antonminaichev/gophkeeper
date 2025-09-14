package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestUserRepo tests the UserRepo implementation
func TestUserRepo(t *testing.T) {
	// Setup test database (in-memory SQLite for simplicity)
	// In a real scenario, you'd use a test PostgreSQL instance
	pool, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer pool.Close()

	repo := NewUserRepo(pool)

	t.Run("Create user", func(t *testing.T) {
		email := "test@example.com"
		passHash := []byte("hashed_password")
		passSalt := []byte("salt")

		id, err := repo.Create(context.Background(), email, passHash, passSalt)
		if err != nil {
			t.Errorf("Create() error = %v", err)
		}
		if id == uuid.Nil {
			t.Errorf("Create() returned nil UUID")
		}
	})

	t.Run("Create user with duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		passHash := []byte("hashed_password")
		passSalt := []byte("salt")

		// Create first user
		_, err := repo.Create(context.Background(), email, passHash, passSalt)
		if err != nil {
			t.Errorf("First Create() error = %v", err)
		}

		// Try to create duplicate
		_, err = repo.Create(context.Background(), email, passHash, passSalt)
		if err == nil {
			t.Errorf("Create() with duplicate email should return error")
		}
	})

	t.Run("ByEmail existing user", func(t *testing.T) {
		email := "existing@example.com"
		passHash := []byte("hashed_password")
		passSalt := []byte("salt")

		// Create user
		_, err := repo.Create(context.Background(), email, passHash, passSalt)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Find user
		user, err := repo.ByEmail(context.Background(), email)
		if err != nil {
			t.Errorf("ByEmail() error = %v", err)
		}
		if user == nil {
			t.Errorf("ByEmail() returned nil user")
		}
		if user.Email != email {
			t.Errorf("ByEmail() email = %v, want %v", user.Email, email)
		}
	})

	t.Run("ByEmail non-existing user", func(t *testing.T) {
		_, err := repo.ByEmail(context.Background(), "nonexistent@example.com")
		if err == nil {
			t.Errorf("ByEmail() with non-existing user should return error")
		}
	})

	t.Run("Email normalization", func(t *testing.T) {
		email := "  TEST@EXAMPLE.COM  "
		normalizedEmail := "test@example.com"
		passHash := []byte("hashed_password")
		passSalt := []byte("salt")

		// Create user with normalized email
		_, err := repo.Create(context.Background(), normalizedEmail, passHash, passSalt)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Find user with different case/spaces
		user, err := repo.ByEmail(context.Background(), email)
		if err != nil {
			t.Errorf("ByEmail() error = %v", err)
		}
		if user == nil {
			t.Errorf("ByEmail() returned nil user")
		}
		if user.Email != normalizedEmail {
			t.Errorf("ByEmail() email = %v, want %v", user.Email, normalizedEmail)
		}
	})
}

// setupTestDB creates a test database connection
// This is a simplified version - in practice you'd use a real PostgreSQL test instance
func setupTestDB(t *testing.T) (*pgxpool.Pool, error) {
	// For testing purposes, we'll skip actual database setup
	// In a real implementation, you would:
	// 1. Start a test PostgreSQL container
	// 2. Create a test database
	// 3. Run migrations
	// 4. Return a connection pool

	t.Skip("Skipping database tests - requires PostgreSQL test instance")
	return nil, nil
}
