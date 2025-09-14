package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID
	Email    string
	PassHash []byte
	PassSalt []byte
}

type UserRepository interface {
	Create(ctx context.Context, email string, passHash, passSalt []byte) (uuid.UUID, error)
	ByEmail(ctx context.Context, email string) (*User, error)
}

type Item struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Type      int16
	Payload   []byte
	MetaJSON  []byte
	Version   int64
	HumanID   *int64
	Alias     *string
	DeletedAt *time.Time
	UpdatedAt time.Time
	CreatedAt time.Time
}

type ItemsRepository interface {
	Create(ctx context.Context, owner uuid.UUID, typ int16, payload, metaJSON []byte, alias *string) (uuid.UUID, int64, error)
	Get(ctx context.Context, owner uuid.UUID, id uuid.UUID) (*Item, error)
	GetByHuman(ctx context.Context, owner uuid.UUID, hid int64) (*Item, error)
	GetByAlias(ctx context.Context, owner uuid.UUID, alias string) (*Item, error)
	List(ctx context.Context, owner uuid.UUID, limit int32) ([]Item, error)
	ListChanges(ctx context.Context, owner uuid.UUID, after time.Time, afterID uuid.UUID, limit int32) ([]Item, error)
	Update(ctx context.Context, owner uuid.UUID, id uuid.UUID, payload, metaJSON []byte, expectedVersion int64) (*Item, error)
	Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error
}
