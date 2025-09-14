package client

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "gophkeeper_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// cleanupTempDir removes a temporary directory
func cleanupTempDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

func TestNewEmptyCache(t *testing.T) {
	cache := newEmptyCache()

	if cache.Items == nil {
		t.Errorf("newEmptyCache() Items should not be nil")
	}
	if cache.ByAlias == nil {
		t.Errorf("newEmptyCache() ByAlias should not be nil")
	}
	if cache.ByHuman == nil {
		t.Errorf("newEmptyCache() ByHuman should not be nil")
	}
	if cache.Version != 1 {
		t.Errorf("newEmptyCache() Version = %d, want 1", cache.Version)
	}
}

func TestLoadCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Set cache file path via environment variable
	originalEnv := os.Getenv("GK_CACHE_FILE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GK_CACHE_FILE", originalEnv)
		} else {
			os.Unsetenv("GK_CACHE_FILE")
		}
	}()
	os.Setenv("GK_CACHE_FILE", filepath.Join(tempDir, "cache.json"))

	t.Run("load non-existent cache", func(t *testing.T) {
		cache, err := LoadCache()
		if err != nil {
			t.Errorf("LoadCache() error = %v", err)
		}
		if cache == nil {
			t.Errorf("LoadCache() returned nil cache")
			return
		}
		if cache.Version != 1 {
			t.Errorf("LoadCache() Version = %d, want 1", cache.Version)
		}
	})
}

func TestSaveCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Set cache file path via environment variable
	originalEnv := os.Getenv("GK_CACHE_FILE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GK_CACHE_FILE", originalEnv)
		} else {
			os.Unsetenv("GK_CACHE_FILE")
		}
	}()
	os.Setenv("GK_CACHE_FILE", filepath.Join(tempDir, "cache.json"))

	cache := newEmptyCache()
	cache.Items["test-id"] = &CachedItem{
		ID:      "test-id",
		Type:    "TEXT",
		Version: 1,
	}

	err := SaveCache(cache)
	if err != nil {
		t.Errorf("SaveCache() error = %v", err)
	}

	// Check that file was created
	cachePath := filepath.Join(tempDir, "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("SaveCache() did not create cache file")
	}
}

func TestApplyChanges(t *testing.T) {
	cache := newEmptyCache()

	items := []*pb.Item{
		{
			Id:            "item1",
			HumanId:       1,
			Alias:         "test1",
			Type:          pb.ItemType_TEXT,
			Version:       1,
			UpdatedAtUnix: time.Now().Unix(),
			Deleted:       false,
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item 1"},
			},
		},
		{
			Id:            "item2",
			HumanId:       2,
			Alias:         "test2",
			Type:          pb.ItemType_LOGIN,
			Version:       1,
			UpdatedAtUnix: time.Now().Unix(),
			Deleted:       false,
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item 2"},
			},
		},
	}

	cache.ApplyChanges(items)

	if len(cache.Items) != 2 {
		t.Errorf("ApplyChanges() added %d items, want 2", len(cache.Items))
	}

	if cache.Items["item1"] == nil {
		t.Errorf("ApplyChanges() did not add item1")
	}
	if cache.Items["item2"] == nil {
		t.Errorf("ApplyChanges() did not add item2")
	}

	// Check indexing
	if cache.ByAlias["test1"] != "item1" {
		t.Errorf("ApplyChanges() did not index alias test1")
	}
	if cache.ByAlias["test2"] != "item2" {
		t.Errorf("ApplyChanges() did not index alias test2")
	}

	if cache.ByHuman[1] != "item1" {
		t.Errorf("ApplyChanges() did not index human ID 1")
	}
	if cache.ByHuman[2] != "item2" {
		t.Errorf("ApplyChanges() did not index human ID 2")
	}
}

func TestApplyChangesDelete(t *testing.T) {
	cache := newEmptyCache()

	// Add item first
	items := []*pb.Item{
		{
			Id:            "item1",
			HumanId:       1,
			Alias:         "test1",
			Type:          pb.ItemType_TEXT,
			Version:       1,
			UpdatedAtUnix: time.Now().Unix(),
			Deleted:       false,
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item 1"},
			},
		},
	}
	cache.ApplyChanges(items)

	// Now delete it
	deleteItems := []*pb.Item{
		{
			Id:      "item1",
			Deleted: true,
		},
	}
	cache.ApplyChanges(deleteItems)

	if len(cache.Items) != 0 {
		t.Errorf("ApplyChanges() should have removed item, got %d items", len(cache.Items))
	}

	if cache.ByAlias["test1"] != "" {
		t.Errorf("ApplyChanges() should have removed alias index")
	}

	if cache.ByHuman[1] != "" {
		t.Errorf("ApplyChanges() should have removed human ID index")
	}
}

func TestUpsertFromItem(t *testing.T) {
	cache := newEmptyCache()

	item := &pb.Item{
		Id:            "item1",
		HumanId:       1,
		Alias:         "test1",
		Type:          pb.ItemType_TEXT,
		Version:       1,
		UpdatedAtUnix: time.Now().Unix(),
		Deleted:       false,
		Meta: []*pb.ItemMetaEntry{
			{Key: "title", Value: "Test Item 1"},
		},
	}

	cache.UpsertFromItem(item)

	if len(cache.Items) != 1 {
		t.Errorf("UpsertFromItem() added %d items, want 1", len(cache.Items))
	}

	cachedItem := cache.Items["item1"]
	if cachedItem == nil {
		t.Errorf("UpsertFromItem() did not add item1")
		return
	}

	if cachedItem.ID != "item1" {
		t.Errorf("UpsertFromItem() ID = %s, want item1", cachedItem.ID)
	}
	if cachedItem.HumanID != 1 {
		t.Errorf("UpsertFromItem() HumanID = %d, want 1", cachedItem.HumanID)
	}
	if cachedItem.Alias != "test1" {
		t.Errorf("UpsertFromItem() Alias = %s, want test1", cachedItem.Alias)
	}
	if cachedItem.Type != "TEXT" {
		t.Errorf("UpsertFromItem() Type = %s, want TEXT", cachedItem.Type)
	}
}

func TestLookupID(t *testing.T) {
	cache := newEmptyCache()

	// Add test item
	item := &pb.Item{
		Id:            "item1",
		HumanId:       1,
		Alias:         "test1",
		Type:          pb.ItemType_TEXT,
		Version:       1,
		UpdatedAtUnix: time.Now().Unix(),
		Deleted:       false,
		Meta: []*pb.ItemMetaEntry{
			{Key: "title", Value: "Test Item 1"},
		},
	}
	cache.UpsertFromItem(item)

	tests := []struct {
		name     string
		selector string
		wantID   string
		wantOK   bool
	}{
		{
			name:     "UUID selector",
			selector: "123e4567-e89b-12d3-a456-426614174000",
			wantID:   "123e4567-e89b-12d3-a456-426614174000",
			wantOK:   true,
		},
		{
			name:     "alias selector",
			selector: "@test1",
			wantID:   "item1",
			wantOK:   true,
		},
		{
			name:     "human ID selector",
			selector: "1",
			wantID:   "item1",
			wantOK:   true,
		},
		{
			name:     "unknown alias",
			selector: "@unknown",
			wantID:   "",
			wantOK:   false,
		},
		{
			name:     "unknown human ID",
			selector: "999",
			wantID:   "",
			wantOK:   false,
		},
		{
			name:     "empty selector",
			selector: "",
			wantID:   "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := cache.LookupID(tt.selector)
			if gotID != tt.wantID {
				t.Errorf("LookupID() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("LookupID() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestCachedItemDebugString(t *testing.T) {
	tests := []struct {
		name string
		item *CachedItem
		want string
	}{
		{
			name: "item with human ID and alias",
			item: &CachedItem{
				ID:      "item1",
				HumanID: 1,
				Alias:   "test1",
				Type:    "TEXT",
				Version: 1,
				Meta:    map[string]string{"title": "Test Item"},
			},
			want: "#1 (@test1)             TEXT    v1    Test Item",
		},
		{
			name: "item with only UUID",
			item: &CachedItem{
				ID:      "item1",
				Type:    "LOGIN",
				Version: 2,
				Meta:    map[string]string{"title": "Login Item"},
			},
			want: "item1                   LOGIN   v2    Login Item",
		},
		{
			name: "item without title",
			item: &CachedItem{
				ID:      "item1",
				HumanID: 1,
				Type:    "TEXT",
				Version: 1,
			},
			want: "#1                      TEXT    v1    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.DebugString()
			if len(got) != len(tt.want) {
				t.Errorf("DebugString() length = %d, want %d", len(got), len(tt.want))
				t.Errorf("DebugString() = %q", got)
				t.Errorf("want = %q", tt.want)
			}
		})
	}
}

func TestClearCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Set cache file path via environment variable
	originalEnv := os.Getenv("GK_CACHE_FILE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GK_CACHE_FILE", originalEnv)
		} else {
			os.Unsetenv("GK_CACHE_FILE")
		}
	}()
	os.Setenv("GK_CACHE_FILE", filepath.Join(tempDir, "cache.json"))

	// Create a cache file
	cache := newEmptyCache()
	err := SaveCache(cache)
	if err != nil {
		t.Fatalf("SaveCache() failed: %v", err)
	}

	// Clear cache
	err = ClearCache()
	if err != nil {
		t.Errorf("ClearCache() error = %v", err)
	}

	// Check that file was removed
	cachePath := filepath.Join(tempDir, "cache.json")
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Errorf("ClearCache() did not remove cache file")
	}
}

func TestClearCacheNonExistent(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Set cache file path via environment variable
	originalEnv := os.Getenv("GK_CACHE_FILE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GK_CACHE_FILE", originalEnv)
		} else {
			os.Unsetenv("GK_CACHE_FILE")
		}
	}()
	os.Setenv("GK_CACHE_FILE", filepath.Join(tempDir, "nonexistent.json"))

	// Clear non-existent cache should not error
	err := ClearCache()
	if err != nil {
		t.Errorf("ClearCache() error = %v", err)
	}
}
