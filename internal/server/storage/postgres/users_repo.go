package postgres

// User repository backed by PostgreSQL.

import (
	"context"
	"strings"

	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

// Create inserts a new user with pre-generated UUID.
func (r *UserRepo) Create(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error) {
	id := uuid.New()
	email = normalizeEmail(email)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, pass_hash, pass_salt) VALUES ($1, $2, $3, $4)`,
		id, email, passHash, passSalt,
	)
	return id, err
}

// ByEmail returns user by normalized email.
func (r *UserRepo) ByEmail(ctx context.Context, email string) (*storage.User, error) {
	email = normalizeEmail(email)

	row := r.pool.QueryRow(ctx,
		`SELECT id, email, pass_hash, pass_salt FROM users WHERE email = $1`,
		email,
	)
	var u storage.User
	if err := row.Scan(&u.ID, &u.Email, &u.PassHash, &u.PassSalt); err != nil {
		return nil, err
	}
	return &u, nil
}

// RunMigrations is a thin wrapper to execute embedded SQL migrations.
func RunMigrations(dsn string) error { return runMigrations(dsn) }

// normalizeEmail trims spaces and lowercases the address.
func normalizeEmail(s string) string {
	s = strings.TrimSpace(s)
	return strings.ToLower(s)
}
