package storage

import (
	"time"

	"github.com/google/uuid"
)

// Доменные модели хранилища

type User struct {
	ID       uuid.UUID
	Email    string
	PassHash []byte
	PassSalt []byte
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
