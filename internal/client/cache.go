package client

//Sqlite client cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Cache struct {
	db *sql.DB
}

type CachedItem struct {
	ID            string            `json:"id"`
	HumanID       int64             `json:"human_id,omitempty"`
	Alias         string            `json:"alias,omitempty"`
	Type          string            `json:"type"`
	Version       int64             `json:"version"`
	UpdatedAtUnix int64             `json:"updated_at_unix"`
	Deleted       bool              `json:"deleted"`
	Meta          map[string]string `json:"meta,omitempty"`
}

const cacheSchemaVersion = 1

func cacheFile() (string, error) {
	if p := os.Getenv("GK_CACHE_FILE"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gophkeeper", "cache.db"), nil
}

func LoadCache() (*Cache, error) {
	path, err := cacheFile()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	_, _ = db.Exec(`PRAGMA synchronous=NORMAL;`)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Cache{db: db}, nil
}

func SaveCache(c *Cache) error {
	if c == nil || c.db == nil {
		return errors.New("nil cache")
	}
	_, err := c.db.Exec(`
		INSERT INTO meta(key, value) VALUES ('saved_at', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, time.Now().Format(time.RFC3339Nano))
	return err
}

func ClearCache() error {
	path, err := cacheFile()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Cache) ApplyChanges(items []*pb.Item) {
	if c == nil || c.db == nil || len(items) == 0 {
		return
	}
	ctx := context.Background()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO items(
			id, human_id, alias, type, version, updated_at_unix, deleted, meta_json
		) VALUES(?, ?, NULLIF(TRIM(?),''), ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			human_id        = excluded.human_id,
			alias           = excluded.alias,
			type            = excluded.type,
			version         = excluded.version,
			updated_at_unix = excluded.updated_at_unix,
			deleted         = excluded.deleted,
			meta_json       = excluded.meta_json
	`)
	if err != nil {
		_ = tx.Rollback()
		return
	}
	for _, it := range items {
		if it == nil {
			continue
		}
		metaJSON, _ := encodeMeta(it.GetMeta())
		_, _ = stmt.ExecContext(ctx,
			strings.TrimSpace(it.GetId()),
			nullInt64(it.GetHumanId()),
			strings.TrimSpace(it.GetAlias()),
			it.GetType().String(),
			it.GetVersion(),
			it.GetUpdatedAtUnix(),
			boolToInt(it.GetDeleted()),
			metaJSON,
		)
	}
	_ = stmt.Close()
	_ = tx.Commit()
}

func (c *Cache) UpsertFromItem(it *pb.Item) {
	if c == nil || c.db == nil || it == nil {
		return
	}
	c.ApplyChanges([]*pb.Item{it})
}

func (c *Cache) LookupID(selector string) (string, bool) {
	s := strings.TrimSpace(selector)
	if s == "" {
		return "", false
	}

	if _, err := uuid.Parse(s); err == nil {
		return s, true
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		var id string
		err := c.db.QueryRow(`SELECT id FROM items WHERE human_id = ? AND deleted = 0`, n).Scan(&id)
		if err == nil && id != "" {
			return id, true
		}
		return "", false
	}

	a := strings.TrimPrefix(s, "@")
	if a == "" {
		return "", false
	}
	var id string
	err := c.db.QueryRow(`SELECT id FROM items WHERE LOWER(alias) = LOWER(?) AND deleted = 0`, a).Scan(&id)
	if err == nil && id != "" {
		return id, true
	}
	return "", false
}

func initSchema(db *sql.DB) error {
	// meta
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`); err != nil {
		return err
	}
	// items
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id               TEXT PRIMARY KEY,
			human_id         INTEGER,
			alias            TEXT UNIQUE,
			type             TEXT NOT NULL,
			version          INTEGER NOT NULL,
			updated_at_unix  INTEGER NOT NULL,
			deleted          INTEGER NOT NULL CHECK (deleted IN (0,1)),
			meta_json        TEXT
		);
	`); err != nil {
		return err
	}
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_items_alias_ci ON items(LOWER(alias)) WHERE alias IS NOT NULL;`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_items_human_id ON items(human_id) WHERE human_id IS NOT NULL;`)
	_, _ = db.Exec(`
		INSERT INTO meta(key, value) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, fmt.Sprint(cacheSchemaVersion))
	return nil
}

func encodeMeta(entries []*pb.ItemMetaEntry) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}
	m := make(map[string]string, len(entries))
	for _, kv := range entries {
		k := strings.TrimSpace(kv.GetKey())
		if k == "" {
			continue
		}
		m[k] = kv.GetValue()
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
