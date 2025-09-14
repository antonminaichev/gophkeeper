package postgres

// PostgreSQL-backed repository for vault items.

import (
	"context"
	"errors"
	"time"

	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemsRepo struct {
	pool *pgxpool.Pool
}

func NewItemsRepo(pool *pgxpool.Pool) *ItemsRepo { return &ItemsRepo{pool: pool} }

// Create stores a new item for an owner and returns its id and version.
func (r *ItemsRepo) Create(
	ctx context.Context,
	owner uuid.UUID,
	typ int16,
	payload, metaJSON []byte,
	alias *string,
) (uuid.UUID, int64, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Allocate per-owner human id: first call inserts (1), next calls increment.
	var humanID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO items_counter(owner_id, next_id)
		VALUES ($1, 1)
		ON CONFLICT (owner_id)
		DO UPDATE SET next_id = items_counter.next_id + 1
		RETURNING next_id
	`, owner).Scan(&humanID); err != nil {
		return uuid.Nil, 0, err
	}

	id := uuid.New()
	var version int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO items (id, owner_id, human_id, alias, type, payload, meta)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, '{}'::jsonb))
		RETURNING version
	`, id, owner, humanID, alias, typ, payload, metaJSON).Scan(&version); err != nil {
		return uuid.Nil, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, 0, err
	}
	return id, version, nil
}

// Get returns a single non-deleted item by id for the owner.
func (r *ItemsRepo) Get(ctx context.Context, owner uuid.UUID, id uuid.UUID) (*storage.Item, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, id, owner)
	return scanItem(row)
}

// GetByHuman returns an item by per-owner human id.
func (r *ItemsRepo) GetByHuman(ctx context.Context, owner uuid.UUID, hid int64) (*storage.Item, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND human_id = $2 AND deleted_at IS NULL
	`, owner, hid)
	return scanItem(row)
}

// GetByAlias returns an item by alias.
func (r *ItemsRepo) GetByAlias(ctx context.Context, owner uuid.UUID, alias string) (*storage.Item, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND alias = $2 AND deleted_at IS NULL
	`, owner, alias)
	return scanItem(row)
}

// List returns up to limit most recently updated non-deleted items.
func (r *ItemsRepo) List(ctx context.Context, owner uuid.UUID, limit int32) ([]storage.Item, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $2
	`, owner, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]storage.Item, 0, limit)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListChanges provides a stable, forward-only change feed since (after, afterID).
func (r *ItemsRepo) ListChanges(ctx context.Context, owner uuid.UUID, after time.Time, afterID uuid.UUID, limit int32) ([]storage.Item, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1
		  AND (
		       updated_at > $2
		       OR (updated_at = $2 AND id > $3)
		  )
		ORDER BY updated_at ASC, id ASC
		LIMIT $4
	`, owner, after, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]storage.Item, 0, limit)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// Update applies concurrency by version and returns the new row.
func (r *ItemsRepo) Update(
	ctx context.Context,
	owner uuid.UUID,
	id uuid.UUID,
	payload, metaJSON []byte,
	expectedVersion int64,
) (*storage.Item, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE items
		SET payload   = COALESCE($4, payload),
		    meta      = COALESCE($5, meta),
		    version   = version + 1,
		    updated_at = now()
		WHERE id = $1
		  AND owner_id = $2
		  AND deleted_at IS NULL
		  AND version = $3
		RETURNING id, owner_id, human_id, alias, type, payload, meta, version,
		          deleted_at, updated_at, created_at
	`, id, owner, expectedVersion, payload, metaJSON)

	it, err := scanItem(row)
	if err != nil {
		// Distinguish version conflicts from "not found".
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("not found or version conflict")
		}
		return nil, err
	}
	return it, nil
}

// Delete marks the item as deleted and bumps updated_at (tombstone entry).
func (r *ItemsRepo) Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE items
		SET deleted_at = $3,
		    updated_at = $3
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, id, owner, time.Now())
	return err
}

// scanItem maps a single Row into storage.Item.
func scanItem(row pgx.Row) (*storage.Item, error) {
	var it storage.Item
	if err := row.Scan(
		&it.ID,
		&it.OwnerID,
		&it.HumanID,
		&it.Alias,
		&it.Type,
		&it.Payload,
		&it.MetaJSON,
		&it.Version,
		&it.DeletedAt,
		&it.UpdatedAt,
		&it.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &it, nil
}
