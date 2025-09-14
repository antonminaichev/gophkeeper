package client

// Local cache for GophKeeper CLI.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

// cacheFile can be overridden via GK_CACHE_FILE.
func cacheFile() (string, error) {
	if p := os.Getenv("GK_CACHE_FILE"); strings.TrimSpace(p) != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gophkeeper", "cache.json"), nil
}

// Cache is a tiny on-disk index of user's items.
type Cache struct {
	Items   map[string]*CachedItem `json:"items"`
	ByAlias map[string]string      `json:"by_alias"`
	ByHuman map[int64]string       `json:"by_human"`
	SavedAt time.Time              `json:"saved_at"`
	Version int                    `json:"schema_version"`
}

// CachedItem is a denormalized snapshot (without payload).
type CachedItem struct {
	ID            string            `json:"id"`
	HumanID       int64             `json:"human_id,omitempty"`
	Alias         string            `json:"alias,omitempty"`
	Type          string            `json:"type"` // pb.ItemType.String()
	Version       int64             `json:"version"`
	UpdatedAtUnix int64             `json:"updated_at_unix"`
	Deleted       bool              `json:"deleted"`
	Meta          map[string]string `json:"meta,omitempty"` // flat map for convenience
}

// LoadCache reads cache from disk (or returns an empty one if file missing).
func LoadCache() (*Cache, error) {
	path, err := cacheFile()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newEmptyCache(), nil
		}
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	// Heal nil maps if needed
	if c.Items == nil {
		c.Items = make(map[string]*CachedItem)
	}
	if c.ByAlias == nil {
		c.ByAlias = make(map[string]string)
	}
	if c.ByHuman == nil {
		c.ByHuman = make(map[int64]string)
	}
	return &c, nil
}

func newEmptyCache() *Cache {
	return &Cache{
		Items:   make(map[string]*CachedItem),
		ByAlias: make(map[string]string),
		ByHuman: make(map[int64]string),
		Version: 1,
	}
}

// SaveCache writes cache to disk.
func SaveCache(c *Cache) error {
	if c == nil {
		return errors.New("nil cache")
	}
	path, err := cacheFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	c.SavedAt = time.Now()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil { // Windows ignores perms
		return err
	}
	return nil
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

// ApplyChanges applies a batch of server items to the cache.
func (c *Cache) ApplyChanges(items []*pb.Item) {
	for _, it := range items {
		c.upsertOrDelete(it)
	}
}

// UpsertFromItem updates cache from a single fetched item (e.g., after Create/Get/Update).
func (c *Cache) UpsertFromItem(it *pb.Item) {
	c.upsertOrDelete(it)
}

func (c *Cache) upsertOrDelete(it *pb.Item) {
	if it == nil || strings.TrimSpace(it.Id) == "" {
		return
	}
	id := strings.TrimSpace(it.Id)

	// If deleting: remove from maps and exit.
	if it.Deleted {
		if old, ok := c.Items[id]; ok {
			if old.Alias != "" {
				delete(c.ByAlias, strings.ToLower(old.Alias))
			}
			if old.HumanID > 0 {
				delete(c.ByHuman, old.HumanID)
			}
		}
		delete(c.Items, id)
		return
	}

	// Build/merge item.
	meta := make(map[string]string, len(it.Meta))
	for _, kv := range it.Meta {
		k := strings.TrimSpace(kv.GetKey())
		if k == "" {
			continue
		}
		meta[k] = kv.GetValue()
	}
	alias := strings.TrimSpace(it.Alias)

	ci, exists := c.Items[id]
	if !exists {
		ci = &CachedItem{ID: id}
		c.Items[id] = ci
	}
	// Remove old index entries if key changed.
	if ci.Alias != "" && !strings.EqualFold(ci.Alias, alias) {
		delete(c.ByAlias, strings.ToLower(ci.Alias))
	}
	if ci.HumanID > 0 && ci.HumanID != it.HumanId && it.HumanId > 0 {
		delete(c.ByHuman, ci.HumanID)
	}

	// Update fields.
	ci.HumanID = it.HumanId
	ci.Alias = alias
	ci.Type = it.Type.String()
	ci.Version = it.Version
	ci.UpdatedAtUnix = it.UpdatedAtUnix
	ci.Deleted = false
	ci.Meta = meta

	// Re-index new keys.
	if alias != "" {
		c.ByAlias[strings.ToLower(alias)] = id
	}
	if it.HumanId > 0 {
		c.ByHuman[it.HumanId] = id
	}
}

// LookupID resolves selector to canonical UUID via cache.
func (c *Cache) LookupID(selector string) (string, bool) {
	s := strings.TrimSpace(selector)
	if s == "" {
		return "", false
	}
	if LooksLikeUUID(s) {
		// We trust UUID; even if it's not in cache yet, caller may still use it.
		return s, true
	}
	if strings.HasPrefix(s, "@") {
		key := strings.ToLower(strings.TrimPrefix(s, "@"))
		id, ok := c.ByAlias[key]
		return id, ok
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		id, ok := c.ByHuman[n]
		return id, ok
	}
	return "", false
}

// DebugString builds a short one-line representation for printing lists/logs.
func (c *CachedItem) DebugString() string {
	tag := c.ID
	var parts []string
	if c.HumanID > 0 {
		parts = append(parts, fmt.Sprintf("#%d", c.HumanID))
	}
	if c.Alias != "" {
		parts = append(parts, "(@"+c.Alias+")")
	}
	if len(parts) > 0 {
		tag = strings.Join(parts, " ")
	}
	title := c.Meta["title"]
	return fmt.Sprintf("%-22s  %-6s  v%-3d  %s", tag, c.Type, c.Version, title)
}
