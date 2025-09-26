package client

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

// --- утилиты для тестов ---

// createTempDir создаёт временную директорию
func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gophkeeper_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// cleanupTempDir удаляет временную директорию
func cleanupTempDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

// setTempCachePath устанавливает GK_CACHE_FILE на файл в tempDir и возвращает путь
func setTempCachePath(t *testing.T, tempDir, name string) string {
	t.Helper()
	orig := os.Getenv("GK_CACHE_FILE")
	t.Cleanup(func() {
		if orig != "" {
			_ = os.Setenv("GK_CACHE_FILE", orig)
		} else {
			_ = os.Unsetenv("GK_CACHE_FILE")
		}
	})
	p := filepath.Join(tempDir, name)
	_ = os.Setenv("GK_CACHE_FILE", p)
	return p
}

// --- тесты ---

func TestLoadCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	cachePath := setTempCachePath(t, tempDir, "cache.db")

	t.Run("load non-existent cache db -> create new", func(t *testing.T) {
		if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
			t.Fatalf("expected no db file yet, got: %v", err)
		}

		c, err := LoadCache()
		if err != nil {
			t.Fatalf("LoadCache() error = %v", err)
		}
		if c == nil || c.db == nil {
			t.Fatalf("LoadCache() returned nil cache/db")
		}
		defer c.Close()

		// SaveCache просто пишет saved_at — проверим, что не падает.
		if err := SaveCache(c); err != nil {
			t.Fatalf("SaveCache() error = %v", err)
		}

		// Должен появиться файл БД
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatalf("LoadCache() did not create db file")
		}
	})
}

func TestSaveCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	cachePath := setTempCachePath(t, tempDir, "cache.db")

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	defer c.Close()

	if err := SaveCache(c); err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatalf("SaveCache() did not create db file")
	}
}

func TestApplyChanges(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)
	setTempCachePath(t, tempDir, "cache.db")

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	defer c.Close()

	now := time.Now().Unix()
	items := []*pb.Item{
		{
			Id:            "item1",
			HumanId:       1,
			Alias:         "test1",
			Type:          pb.ItemType_TEXT,
			Version:       1,
			UpdatedAtUnix: now,
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
			UpdatedAtUnix: now,
			Deleted:       false,
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item 2"},
			},
		},
	}
	c.ApplyChanges(items)

	// Проверим индексацию через LookupID
	if id, ok := c.LookupID("@test1"); !ok || id != "item1" {
		t.Fatalf("LookupID(@test1) = %q,%v; want item1,true", id, ok)
	}
	if id, ok := c.LookupID("1"); !ok || id != "item1" {
		t.Fatalf("LookupID(1) = %q,%v; want item1,true", id, ok)
	}
	if id, ok := c.LookupID("@test2"); !ok || id != "item2" {
		t.Fatalf("LookupID(@test2) = %q,%v; want item2,true", id, ok)
	}
	if id, ok := c.LookupID("2"); !ok || id != "item2" {
		t.Fatalf("LookupID(2) = %q,%v; want item2,true", id, ok)
	}
}

func TestApplyChangesDelete(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)
	setTempCachePath(t, tempDir, "cache.db")

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	defer c.Close()

	// Добавим
	c.ApplyChanges([]*pb.Item{
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
	})

	// Удалим
	c.ApplyChanges([]*pb.Item{
		{Id: "item1", Deleted: true},
	})

	// LookupID должен больше не находить по alias и human_id (deleted=0 фильтр)
	if _, ok := c.LookupID("@test1"); ok {
		t.Fatalf("LookupID(@test1) found deleted item, want not found")
	}
	if _, ok := c.LookupID("1"); ok {
		t.Fatalf("LookupID(1) found deleted item, want not found")
	}
}

func TestUpsertFromItem(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)
	setTempCachePath(t, tempDir, "cache.db")

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	defer c.Close()

	// Создаем валидный UUID для теста
	itemUUID := "550e8400-e29b-41d4-a716-446655440000"
	item := &pb.Item{
		Id:            itemUUID,
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
	c.UpsertFromItem(item)

	// Проверим, что можно найти по трём селекторам
	if id, ok := c.LookupID(itemUUID); !ok || id != itemUUID {
		t.Fatalf("LookupID(uuid) = %q,%v; want %s,true", id, ok, itemUUID)
	}
	if id, ok := c.LookupID("@test1"); !ok || id != itemUUID {
		t.Fatalf("LookupID(@test1) = %q,%v; want %s,true", id, ok, itemUUID)
	}
	if id, ok := c.LookupID("1"); !ok || id != itemUUID {
		t.Fatalf("LookupID(1) = %q,%v; want %s,true", id, ok, itemUUID)
	}
}

func TestLookupID(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)
	setTempCachePath(t, tempDir, "cache.db")

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	defer c.Close()

	// Наполним кэш данными
	c.ApplyChanges([]*pb.Item{
		{
			Id:            "item1",
			HumanId:       1,
			Alias:         "test1",
			Type:          pb.ItemType_TEXT,
			Version:       1,
			UpdatedAtUnix: time.Now().Unix(),
			Deleted:       false,
			Meta:          []*pb.ItemMetaEntry{{Key: "title", Value: "Test Item 1"}},
		},
	})

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
			gotID, gotOK := c.LookupID(tt.selector)
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
		},
		{
			name: "item with only UUID",
			item: &CachedItem{
				ID:      "12345678-aaaa-bbbb-cccc-0123456789ab",
				Type:    "LOGIN",
				Version: 2,
				Meta:    map[string]string{"title": "Login Item"},
			},
		},
		{
			name: "item without title",
			item: &CachedItem{
				ID:      "item1",
				HumanID: 1,
				Type:    "TEXT",
				Version: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.DebugString()
			if got == "" {
				t.Fatalf("DebugString() returned empty string")
			}
			// Простейшие инварианты формата:
			if !strings.Contains(got, tt.item.Type) {
				t.Errorf("DebugString() = %q; must contain type %q", got, tt.item.Type)
			}
			if tt.item.Alias != "" && !strings.Contains(got, "@"+tt.item.Alias) {
				t.Errorf("DebugString() = %q; must contain alias @%s", got, tt.item.Alias)
			}
			if tt.item.Alias == "" && tt.item.HumanID > 0 && !strings.Contains(got, "#"+itoa(tt.item.HumanID)) {
				t.Errorf("DebugString() = %q; must contain human id #%d", got, tt.item.HumanID)
			}
		})
	}
}

func TestClearCache(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	cachePath := setTempCachePath(t, tempDir, "cache.db")

	// Создадим файл БД
	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}
	// Закрыть перед удалением — важно для Windows
	_ = c.Close()

	// Очистка
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	// Файл должен быть удалён
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("ClearCache() did not remove db file")
	}
}

func TestClearCacheNonExistent(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	setTempCachePath(t, tempDir, "nonexistent.db")

	// Очистка несуществующего кэша не должна возвращать ошибку
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}
}

// маленький хелпер для форматирования human id
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
