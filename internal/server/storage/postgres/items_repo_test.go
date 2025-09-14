package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestItemsRepo tests the ItemsRepo implementation
func TestItemsRepo(t *testing.T) {
	// Setup test database
	pool, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer pool.Close()

	repo := NewItemsRepo(pool)
	ownerID := uuid.New()

	t.Run("Create item", func(t *testing.T) {
		typ := int16(1) // TEXT type
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)
		alias := "test-alias"

		id, version, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, &alias)
		if err != nil {
			t.Errorf("Create() error = %v", err)
		}
		if id == uuid.Nil {
			t.Errorf("Create() returned nil UUID")
		}
		if version != 1 {
			t.Errorf("Create() version = %d, want 1", version)
		}
	})

	t.Run("Create item without alias", func(t *testing.T) {
		typ := int16(2) // LOGIN type
		payload := []byte("login payload")
		metaJSON := []byte(`{"title": "Login Item"}`)

		id, version, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Errorf("Create() error = %v", err)
		}
		if id == uuid.Nil {
			t.Errorf("Create() returned nil UUID")
		}
		if version != 1 {
			t.Errorf("Create() version = %d, want 1", version)
		}
	})

	t.Run("Get item by ID", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)

		// Create item
		id, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Get item
		item, err := repo.Get(context.Background(), ownerID, id)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if item == nil {
			t.Errorf("Get() returned nil item")
		}
		if item.ID != id {
			t.Errorf("Get() ID = %v, want %v", item.ID, id)
		}
		if item.Type != typ {
			t.Errorf("Get() Type = %d, want %d", item.Type, typ)
		}
	})

	t.Run("Get item by human ID", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)

		// Create item
		_, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Get item by human ID (should be 1 for first item)
		item, err := repo.GetByHuman(context.Background(), ownerID, 1)
		if err != nil {
			t.Errorf("GetByHuman() error = %v", err)
		}
		if item == nil {
			t.Errorf("GetByHuman() returned nil item")
		}
		if item.HumanID == nil || *item.HumanID != 1 {
			t.Errorf("GetByHuman() HumanID = %v, want 1", item.HumanID)
		}
	})

	t.Run("Get item by alias", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)
		alias := "unique-alias"

		// Create item
		_, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, &alias)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Get item by alias
		item, err := repo.GetByAlias(context.Background(), ownerID, alias)
		if err != nil {
			t.Errorf("GetByAlias() error = %v", err)
		}
		if item == nil {
			t.Errorf("GetByAlias() returned nil item")
		}
		if item.Alias == nil || *item.Alias != alias {
			t.Errorf("GetByAlias() Alias = %v, want %v", item.Alias, alias)
		}
	})

	t.Run("List items", func(t *testing.T) {
		// Create multiple items
		for i := 0; i < 3; i++ {
			typ := int16(1)
			payload := []byte("test payload")
			metaJSON := []byte(`{"title": "Test Item"}`)
			_, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		// List items
		items, err := repo.List(context.Background(), ownerID, 10)
		if err != nil {
			t.Errorf("List() error = %v", err)
		}
		if len(items) < 3 {
			t.Errorf("List() returned %d items, want at least 3", len(items))
		}
	})

	t.Run("List changes", func(t *testing.T) {
		// Create an item
		typ := int16(1)
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)
		id, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// List changes since before creation
		pastTime := time.Now().Add(-1 * time.Hour)
		items, err := repo.ListChanges(context.Background(), ownerID, pastTime, uuid.Nil, 10)
		if err != nil {
			t.Errorf("ListChanges() error = %v", err)
		}
		if len(items) == 0 {
			t.Errorf("ListChanges() returned no items")
		}

		// Check that our item is in the results
		found := false
		for _, item := range items {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListChanges() did not return created item")
		}
	})

	t.Run("Update item", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("original payload")
		metaJSON := []byte(`{"title": "Original Title"}`)

		// Create item
		id, version, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Update item
		newPayload := []byte("updated payload")
		newMetaJSON := []byte(`{"title": "Updated Title"}`)
		updatedItem, err := repo.Update(context.Background(), ownerID, id, newPayload, newMetaJSON, version)
		if err != nil {
			t.Errorf("Update() error = %v", err)
		}
		if updatedItem == nil {
			t.Errorf("Update() returned nil item")
		}
		if updatedItem.Version != version+1 {
			t.Errorf("Update() version = %d, want %d", updatedItem.Version, version+1)
		}
	})

	t.Run("Update item with wrong version", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("original payload")
		metaJSON := []byte(`{"title": "Original Title"}`)

		// Create item
		id, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to update with wrong version
		newPayload := []byte("updated payload")
		_, err = repo.Update(context.Background(), ownerID, id, newPayload, metaJSON, 999)
		if err == nil {
			t.Errorf("Update() with wrong version should return error")
		}
	})

	t.Run("Delete item", func(t *testing.T) {
		typ := int16(1)
		payload := []byte("test payload")
		metaJSON := []byte(`{"title": "Test Item"}`)

		// Create item
		id, _, err := repo.Create(context.Background(), ownerID, typ, payload, metaJSON, nil)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Delete item
		err = repo.Delete(context.Background(), ownerID, id)
		if err != nil {
			t.Errorf("Delete() error = %v", err)
		}

		// Try to get deleted item
		_, err = repo.Get(context.Background(), ownerID, id)
		if err == nil {
			t.Errorf("Get() deleted item should return error")
		}
	})

	t.Run("Get non-existing item", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.Get(context.Background(), ownerID, nonExistentID)
		if err == nil {
			t.Errorf("Get() non-existing item should return error")
		}
	})
}
